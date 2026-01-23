// Package lexical 实现优化词法分析适配器
package lexical

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	lexicalServices "echo/internal/modules/frontend/domain/lexical/services"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
)

// OptimizedLexerAdapter 优化词法分析适配器
// 提供缓存、性能监控、批量处理等优化功能
type OptimizedLexerAdapter struct {
	// 底层词法分析服务
	lexerService *lexicalServices.AdvancedLexerService

	// Token流缓存（基于文件内容hash）
	tokenCache map[string]*CachedTokenStream
	cacheMutex sync.RWMutex

	// 性能统计
	stats *LexerPerformanceStats
	statsMutex sync.RWMutex

	// 配置
	config *OptimizationConfig
}

// CachedTokenStream 缓存的Token流
type CachedTokenStream struct {
	TokenStream *lexicalVO.EnhancedTokenStream
	CachedAt    time.Time
	HitCount    int
}

// LexerPerformanceStats 词法分析性能统计
type LexerPerformanceStats struct {
	TotalRequests     int64
	CacheHits         int64
	CacheMisses       int64
	TotalDuration     time.Duration
	AverageDuration   time.Duration
	MaxDuration       time.Duration
	MinDuration       time.Duration
}

// OptimizationConfig 优化配置
type OptimizationConfig struct {
	EnableCache      bool          // 是否启用缓存
	CacheTTL         time.Duration // 缓存TTL
	MaxCacheSize     int           // 最大缓存条目数
	EnableStats      bool          // 是否启用性能统计
	BatchSize        int           // 批量处理大小
	EnableParallel   bool          // 是否启用并行处理
}

// DefaultOptimizationConfig 默认优化配置
func DefaultOptimizationConfig() *OptimizationConfig {
	return &OptimizationConfig{
		EnableCache:    true,
		CacheTTL:       5 * time.Minute,
		MaxCacheSize:   1000,
		EnableStats:    true,
		BatchSize:      10,
		EnableParallel: true,
	}
}

// NewOptimizedLexerAdapter 创建优化词法分析适配器
func NewOptimizedLexerAdapter(config *OptimizationConfig) *OptimizedLexerAdapter {
	if config == nil {
		config = DefaultOptimizationConfig()
	}

	return &OptimizedLexerAdapter{
		lexerService: lexicalServices.NewAdvancedLexerService(),
		tokenCache:   make(map[string]*CachedTokenStream),
		stats: &LexerPerformanceStats{
			MinDuration: time.Hour, // 初始化为很大的值
		},
		config: config,
	}
}

// Tokenize 执行词法分析（带缓存和性能监控）
func (ola *OptimizedLexerAdapter) Tokenize(
	ctx context.Context,
	sourceFile *lexicalVO.SourceFile,
) (*lexicalVO.EnhancedTokenStream, error) {

	startTime := time.Now()

	// 更新统计
	if ola.config.EnableStats {
		ola.statsMutex.Lock()
		ola.stats.TotalRequests++
		ola.statsMutex.Unlock()
	}

	// 检查缓存
	if ola.config.EnableCache {
		cached, err := ola.getFromCache(sourceFile)
		if err == nil && cached != nil {
			// 缓存命中
			if ola.config.EnableStats {
				ola.statsMutex.Lock()
				ola.stats.CacheHits++
				ola.statsMutex.Unlock()
			}
			return cached.TokenStream, nil
		}

		// 缓存未命中
		if ola.config.EnableStats {
			ola.statsMutex.Lock()
			ola.stats.CacheMisses++
			ola.statsMutex.Unlock()
		}
	}

	// 执行词法分析
	tokenStream, err := ola.lexerService.Tokenize(ctx, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("lexical analysis failed: %w", err)
	}

	duration := time.Since(startTime)

	// 更新性能统计
	if ola.config.EnableStats {
		ola.updateStats(duration)
	}

	// 写入缓存
	if ola.config.EnableCache {
		ola.putToCache(sourceFile, tokenStream)
	}

	return tokenStream, nil
}

// TokenizeBatch 批量执行词法分析
func (ola *OptimizedLexerAdapter) TokenizeBatch(
	ctx context.Context,
	sourceFiles []*lexicalVO.SourceFile,
) (map[string]*lexicalVO.EnhancedTokenStream, error) {

	results := make(map[string]*lexicalVO.EnhancedTokenStream)

	if ola.config.EnableParallel && len(sourceFiles) > ola.config.BatchSize {
		// 并行处理
		return ola.tokenizeBatchParallel(ctx, sourceFiles)
	}

	// 串行处理
	for _, sourceFile := range sourceFiles {
		tokenStream, err := ola.Tokenize(ctx, sourceFile)
		if err != nil {
			return nil, fmt.Errorf("failed to tokenize %s: %w", sourceFile.Filename(), err)
		}
		results[sourceFile.Filename()] = tokenStream
	}

	return results, nil
}

// tokenizeBatchParallel 并行批量处理
func (ola *OptimizedLexerAdapter) tokenizeBatchParallel(
	ctx context.Context,
	sourceFiles []*lexicalVO.SourceFile,
) (map[string]*lexicalVO.EnhancedTokenStream, error) {

	results := make(map[string]*lexicalVO.EnhancedTokenStream)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(sourceFiles))

	// 分批处理
	for i := 0; i < len(sourceFiles); i += ola.config.BatchSize {
		end := i + ola.config.BatchSize
		if end > len(sourceFiles) {
			end = len(sourceFiles)
		}

		batch := sourceFiles[i:end]

		wg.Add(len(batch))
		for _, sourceFile := range batch {
			go func(sf *lexicalVO.SourceFile) {
				defer wg.Done()

				tokenStream, err := ola.Tokenize(ctx, sf)
				if err != nil {
					errChan <- fmt.Errorf("failed to tokenize %s: %w", sf.Filename(), err)
					return
				}

				mu.Lock()
				results[sf.Filename()] = tokenStream
				mu.Unlock()
			}(sourceFile)
		}

		wg.Wait()
	}

	// 检查错误
	close(errChan)
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// getFromCache 从缓存获取Token流
func (ola *OptimizedLexerAdapter) getFromCache(sourceFile *lexicalVO.SourceFile) (*CachedTokenStream, error) {
	ola.cacheMutex.RLock()
	defer ola.cacheMutex.RUnlock()

	cacheKey := ola.generateCacheKey(sourceFile)
	cached, exists := ola.tokenCache[cacheKey]

	if !exists {
		return nil, fmt.Errorf("cache miss")
	}

	// 检查TTL
	if time.Since(cached.CachedAt) > ola.config.CacheTTL {
		return nil, fmt.Errorf("cache expired")
	}

	// 更新命中计数
	cached.HitCount++

	return cached, nil
}

// putToCache 写入缓存
func (ola *OptimizedLexerAdapter) putToCache(
	sourceFile *lexicalVO.SourceFile,
	tokenStream *lexicalVO.EnhancedTokenStream,
) {
	ola.cacheMutex.Lock()
	defer ola.cacheMutex.Unlock()

	// 检查缓存大小
	if len(ola.tokenCache) >= ola.config.MaxCacheSize {
		ola.evictOldestCache()
	}

	cacheKey := ola.generateCacheKey(sourceFile)
	ola.tokenCache[cacheKey] = &CachedTokenStream{
		TokenStream: tokenStream,
		CachedAt:    time.Now(),
		HitCount:    0,
	}
}

// evictOldestCache 淘汰最旧的缓存条目
func (ola *OptimizedLexerAdapter) evictOldestCache() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, cached := range ola.tokenCache {
		if first || cached.CachedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.CachedAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(ola.tokenCache, oldestKey)
	}
}

// generateCacheKey 生成缓存键
func (ola *OptimizedLexerAdapter) generateCacheKey(sourceFile *lexicalVO.SourceFile) string {
	// 使用文件名和内容hash作为缓存键
	// 使用SHA256哈希确保内容变化时缓存键也会变化
	content := sourceFile.Content()
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("%s:%s", sourceFile.Filename(), hashStr)
}

// updateStats 更新性能统计
func (ola *OptimizedLexerAdapter) updateStats(duration time.Duration) {
	ola.statsMutex.Lock()
	defer ola.statsMutex.Unlock()

	ola.stats.TotalDuration += duration

	// 更新平均耗时
	if ola.stats.TotalRequests > 0 {
		ola.stats.AverageDuration = ola.stats.TotalDuration / time.Duration(ola.stats.TotalRequests)
	}

	// 更新最大耗时
	if duration > ola.stats.MaxDuration {
		ola.stats.MaxDuration = duration
	}

	// 更新最小耗时
	if duration < ola.stats.MinDuration {
		ola.stats.MinDuration = duration
	}
}

// GetStats 获取性能统计
func (ola *OptimizedLexerAdapter) GetStats() *LexerPerformanceStats {
	ola.statsMutex.RLock()
	defer ola.statsMutex.RUnlock()

	// 返回副本
	stats := *ola.stats
	return &stats
}

// ClearCache 清空缓存
func (ola *OptimizedLexerAdapter) ClearCache() {
	ola.cacheMutex.Lock()
	defer ola.cacheMutex.Unlock()

	ola.tokenCache = make(map[string]*CachedTokenStream)
}

// ResetStats 重置统计
func (ola *OptimizedLexerAdapter) ResetStats() {
	ola.statsMutex.Lock()
	defer ola.statsMutex.Unlock()

	ola.stats = &LexerPerformanceStats{
		MinDuration: time.Hour,
	}
}

// GetCacheSize 获取缓存大小
func (ola *OptimizedLexerAdapter) GetCacheSize() int {
	ola.cacheMutex.RLock()
	defer ola.cacheMutex.RUnlock()

	return len(ola.tokenCache)
}
