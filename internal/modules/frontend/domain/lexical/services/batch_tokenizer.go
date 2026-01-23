// Package services 定义词法分析上下文的领域服务
package services

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
)

// BatchTokenizer 批量token化服务
// 支持并行处理多个源文件，提高词法分析性能
type BatchTokenizer struct {
	// 依赖的服务
	lexerService *AdvancedLexerService

	// 配置参数
	maxConcurrency int
	batchSize      int

	// 性能监控
	processedFiles int64
	totalTokens    int64
	mu             sync.RWMutex
}

// BatchTokenizationResult 批量token化结果
type BatchTokenizationResult struct {
	Results []*TokenizationResult
	Errors  []*BatchTokenizationError
	Stats   *BatchTokenizationStats
}

// TokenizationResult 单个文件token化结果
type TokenizationResult struct {
	SourceFile     *lexicalVO.SourceFile
	TokenStream    *lexicalVO.EnhancedTokenStream
	ProcessingTime int64 // 纳秒
	TokenCount     int
}

// BatchTokenizationError 批量token化错误
type BatchTokenizationError struct {
	SourceFile *lexicalVO.SourceFile
	Error      error
}

// BatchTokenizationStats 批量token化统计信息
type BatchTokenizationStats struct {
	TotalFiles        int
	ProcessedFiles    int
	FailedFiles       int
	TotalTokens       int64
	TotalProcessingTime int64 // 纳秒
	AverageTokensPerFile float64
	AverageProcessingTime float64 // 毫秒
	ConcurrencyLevel   int
}

// NewBatchTokenizer 创建新的批量token化服务
func NewBatchTokenizer(lexerService *AdvancedLexerService) *BatchTokenizer {
	return &BatchTokenizer{
		lexerService:   lexerService,
		maxConcurrency: runtime.NumCPU(), // 默认使用所有CPU核心
		batchSize:      10,               // 默认批次大小
		processedFiles: 0,
		totalTokens:    0,
	}
}

// SetMaxConcurrency 设置最大并发数
func (bt *BatchTokenizer) SetMaxConcurrency(concurrency int) {
	if concurrency > 0 {
		bt.maxConcurrency = concurrency
	}
}

// SetBatchSize 设置批次大小
func (bt *BatchTokenizer) SetBatchSize(size int) {
	if size > 0 {
		bt.batchSize = size
	}
}

// TokenizeBatch 批量token化多个源文件
func (bt *BatchTokenizer) TokenizeBatch(ctx context.Context, sourceFiles []*lexicalVO.SourceFile) (*BatchTokenizationResult, error) {
	if len(sourceFiles) == 0 {
		return &BatchTokenizationResult{
			Results: make([]*TokenizationResult, 0),
			Errors:  make([]*BatchTokenizationError, 0),
			Stats:   bt.createEmptyStats(),
		}, nil
	}

	// 创建工作通道
	fileChan := make(chan *lexicalVO.SourceFile, len(sourceFiles))
	resultChan := make(chan *TokenizationResult, len(sourceFiles))
	errorChan := make(chan *BatchTokenizationError, len(sourceFiles))

	// 发送文件到工作通道
	go func() {
		defer close(fileChan)
		for _, sourceFile := range sourceFiles {
			select {
			case fileChan <- sourceFile:
			case <-ctx.Done():
				return
			}
		}
	}()

	// 启动worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < bt.maxConcurrency; i++ {
		wg.Add(1)
		go bt.worker(ctx, fileChan, resultChan, errorChan, &wg)
	}

	// 收集结果
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// 收集所有结果
	results := make([]*TokenizationResult, 0, len(sourceFiles))
	errors := make([]*BatchTokenizationError, 0)

	resultsDone := make(chan struct{})
	errorsDone := make(chan struct{})

	// 收集成功结果
	go func() {
		defer close(resultsDone)
		for result := range resultChan {
			results = append(results, result)
			bt.updateStats(result.TokenCount, result.ProcessingTime)
		}
	}()

	// 收集错误结果
	go func() {
		defer close(errorsDone)
		for err := range errorChan {
			errors = append(errors, err)
		}
	}()

	// 等待收集完成
	<-resultsDone
	<-errorsDone

	// 创建统计信息
	stats := bt.createStats(len(sourceFiles), results, errors)

	return &BatchTokenizationResult{
		Results: results,
		Errors:  errors,
		Stats:   stats,
	}, nil
}

// worker 工作协程，处理单个文件的token化
func (bt *BatchTokenizer) worker(
	ctx context.Context,
	fileChan <-chan *lexicalVO.SourceFile,
	resultChan chan<- *TokenizationResult,
	errorChan chan<- *BatchTokenizationError,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		select {
		case sourceFile, ok := <-fileChan:
			if !ok {
				return // 通道已关闭
			}

			bt.processFile(ctx, sourceFile, resultChan, errorChan)

		case <-ctx.Done():
			return // 上下文取消
		}
	}
}

// processFile 处理单个文件
func (bt *BatchTokenizer) processFile(
	ctx context.Context,
	sourceFile *lexicalVO.SourceFile,
	resultChan chan<- *TokenizationResult,
	errorChan chan<- *BatchTokenizationError,
) {
	// 执行token化
	tokenStream, err := bt.lexerService.Tokenize(ctx, sourceFile)
	if err != nil {
		errorChan <- &BatchTokenizationError{
			SourceFile: sourceFile,
			Error:      err,
		}
		return
	}

	// 创建结果
	result := &TokenizationResult{
		SourceFile:  sourceFile,
		TokenStream: tokenStream,
		TokenCount:  tokenStream.Count(),
		// ProcessingTime 暂时设为0，实际使用可以测量时间
		ProcessingTime: 0,
	}

	resultChan <- result
}

// TokenizeBatchSequential 顺序批量token化（用于调试或低并发场景）
func (bt *BatchTokenizer) TokenizeBatchSequential(ctx context.Context, sourceFiles []*lexicalVO.SourceFile) (*BatchTokenizationResult, error) {
	results := make([]*TokenizationResult, 0, len(sourceFiles))
	errors := make([]*BatchTokenizationError, 0)

	for _, sourceFile := range sourceFiles {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		tokenStream, err := bt.lexerService.Tokenize(ctx, sourceFile)
		if err != nil {
			errors = append(errors, &BatchTokenizationError{
				SourceFile: sourceFile,
				Error:      err,
			})
			continue
		}

		result := &TokenizationResult{
			SourceFile:     sourceFile,
			TokenStream:    tokenStream,
			TokenCount:     tokenStream.Count(),
			ProcessingTime: 0, // 可以添加时间测量
		}

		results = append(results, result)
		bt.updateStats(result.TokenCount, result.ProcessingTime)
	}

	stats := bt.createStats(len(sourceFiles), results, errors)

	return &BatchTokenizationResult{
		Results: results,
		Errors:  errors,
		Stats:   stats,
	}, nil
}

// updateStats 更新统计信息
func (bt *BatchTokenizer) updateStats(tokenCount int, processingTime int64) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.processedFiles++
	bt.totalTokens += int64(tokenCount)
}

// createStats 创建统计信息
func (bt *BatchTokenizer) createStats(
	totalFiles int,
	results []*TokenizationResult,
	errors []*BatchTokenizationError,
) *BatchTokenizationStats {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	totalTokens := bt.totalTokens
	processedFiles := int(bt.processedFiles)

	var totalProcessingTime int64
	for _, result := range results {
		totalProcessingTime += result.ProcessingTime
	}

	averageTokensPerFile := float64(0)
	if processedFiles > 0 {
		averageTokensPerFile = float64(totalTokens) / float64(processedFiles)
	}

	averageProcessingTime := float64(0)
	if processedFiles > 0 {
		averageProcessingTime = float64(totalProcessingTime) / float64(processedFiles) / 1e6 // 转换为毫秒
	}

	return &BatchTokenizationStats{
		TotalFiles:           totalFiles,
		ProcessedFiles:       processedFiles,
		FailedFiles:          len(errors),
		TotalTokens:          totalTokens,
		TotalProcessingTime:  totalProcessingTime,
		AverageTokensPerFile: averageTokensPerFile,
		AverageProcessingTime: averageProcessingTime,
		ConcurrencyLevel:     bt.maxConcurrency,
	}
}

// createEmptyStats 创建空的统计信息
func (bt *BatchTokenizer) createEmptyStats() *BatchTokenizationStats {
	return &BatchTokenizationStats{
		TotalFiles:           0,
		ProcessedFiles:       0,
		FailedFiles:          0,
		TotalTokens:          0,
		TotalProcessingTime:  0,
		AverageTokensPerFile: 0,
		AverageProcessingTime: 0,
		ConcurrencyLevel:     bt.maxConcurrency,
	}
}

// GetStats 获取当前统计信息
func (bt *BatchTokenizer) GetStats() *BatchTokenizationStats {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	return &BatchTokenizationStats{
		TotalFiles:           int(bt.processedFiles),
		ProcessedFiles:       int(bt.processedFiles),
		FailedFiles:          0, // 只统计成功处理的
		TotalTokens:          bt.totalTokens,
		TotalProcessingTime:  0,
		AverageTokensPerFile: float64(bt.totalTokens) / float64(bt.processedFiles),
		AverageProcessingTime: 0,
		ConcurrencyLevel:     bt.maxConcurrency,
	}
}

// ResetStats 重置统计信息
func (bt *BatchTokenizer) ResetStats() {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.processedFiles = 0
	bt.totalTokens = 0
}

// 结果类型的方法

// String 返回BatchTokenizationResult的字符串表示
func (btr *BatchTokenizationResult) String() string {
	return fmt.Sprintf("BatchTokenizationResult{Success: %d, Errors: %d, TotalTokens: %d}",
		len(btr.Results), len(btr.Errors), btr.Stats.TotalTokens)
}

// HasErrors 检查是否有错误
func (btr *BatchTokenizationResult) HasErrors() bool {
	return len(btr.Errors) > 0
}

// SuccessRate 返回成功率（0.0-1.0）
func (btr *BatchTokenizationResult) SuccessRate() float64 {
	total := len(btr.Results) + len(btr.Errors)
	if total == 0 {
		return 1.0
	}
	return float64(len(btr.Results)) / float64(total)
}

// String 返回BatchTokenizationStats的字符串表示
func (bts *BatchTokenizationStats) String() string {
	return fmt.Sprintf("BatchTokenizationStats{Files: %d/%d, Tokens: %d, AvgTokens/File: %.1f, Concurrency: %d}",
		bts.ProcessedFiles, bts.TotalFiles, bts.TotalTokens, bts.AverageTokensPerFile, bts.ConcurrencyLevel)
}
