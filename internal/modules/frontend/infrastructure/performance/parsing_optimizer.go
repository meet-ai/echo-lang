// Package performance 实现解析性能优化器
package performance

import (
	"context"
	"sync"
	"time"

	"echo/internal/modules/frontend/domain/parser"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// ParsingOptimizer 解析性能优化器
// 提供批量解析、并行解析、性能监控等优化功能
type ParsingOptimizer struct {
	// 性能统计
	stats *ParsingPerformanceStats
	statsMutex sync.RWMutex

	// 配置
	config *ParsingOptimizationConfig
}

// ParsingPerformanceStats 解析性能统计
type ParsingPerformanceStats struct {
	TotalParses       int64
	SuccessfulParses  int64
	FailedParses      int64
	TotalDuration     time.Duration
	AverageDuration   time.Duration
	MaxDuration       time.Duration
	MinDuration       time.Duration
	TotalTokens       int64
	AverageTokensPerParse int64
}

// ParsingOptimizationConfig 解析优化配置
type ParsingOptimizationConfig struct {
	EnableStats      bool          // 是否启用性能统计
	BatchSize        int           // 批量处理大小
	EnableParallel   bool          // 是否启用并行处理
	MaxConcurrency   int           // 最大并发数
	EnableProfiling  bool          // 是否启用性能分析
}

// DefaultParsingOptimizationConfig 默认解析优化配置
func DefaultParsingOptimizationConfig() *ParsingOptimizationConfig {
	return &ParsingOptimizationConfig{
		EnableStats:     true,
		BatchSize:       10,
		EnableParallel:  true,
		MaxConcurrency:  4,
		EnableProfiling: false,
	}
}

// NewParsingOptimizer 创建解析性能优化器
func NewParsingOptimizer(config *ParsingOptimizationConfig) *ParsingOptimizer {
	if config == nil {
		config = DefaultParsingOptimizationConfig()
	}

	return &ParsingOptimizer{
		stats: &ParsingPerformanceStats{
			MinDuration: time.Hour, // 初始化为很大的值
		},
		config: config,
	}
}

// ParseSingle 解析单个文件（带性能监控）
func (po *ParsingOptimizer) ParseSingle(
	ctx context.Context,
	parserAggregate *parser.ModernParserAggregate,
	sourceCode string,
	filename string,
) (*sharedVO.ProgramAST, error) {

	startTime := time.Now()

	// 执行解析
	programAST, err := parserAggregate.Parse(ctx, sourceCode, filename)

	duration := time.Since(startTime)

	// 更新统计
	if po.config.EnableStats {
		po.updateStats(duration, err == nil, programAST)
	}

	return programAST, err
}

// ParseBatch 批量解析文件
func (po *ParsingOptimizer) ParseBatch(
	ctx context.Context,
	sourceFiles []*BatchParseRequest,
) ([]*BatchParseResult, error) {

	if po.config.EnableParallel && len(sourceFiles) > po.config.BatchSize {
		// 并行处理
		return po.parseBatchParallel(ctx, sourceFiles)
	}

	// 串行处理
	return po.parseBatchSequential(ctx, sourceFiles)
}

// BatchParseRequest 批量解析请求
type BatchParseRequest struct {
	SourceCode string
	Filename   string
	Parser     *parser.ModernParserAggregate
}

// BatchParseResult 批量解析结果
type BatchParseResult struct {
	Filename   string
	ProgramAST *sharedVO.ProgramAST
	Success    bool
	Error      error
	Duration   time.Duration
}

// parseBatchSequential 串行批量解析
func (po *ParsingOptimizer) parseBatchSequential(
	ctx context.Context,
	requests []*BatchParseRequest,
) ([]*BatchParseResult, error) {

	results := make([]*BatchParseResult, 0, len(requests))

	for _, req := range requests {
		startTime := time.Now()

		programAST, err := req.Parser.Parse(ctx, req.SourceCode, req.Filename)

		duration := time.Since(startTime)

		result := &BatchParseResult{
			Filename:   req.Filename,
			ProgramAST: programAST,
			Success:    err == nil,
			Error:      err,
			Duration:   duration,
		}

		results = append(results, result)

		// 更新统计
		if po.config.EnableStats {
			po.updateStats(duration, err == nil, programAST)
		}
	}

	return results, nil
}

// parseBatchParallel 并行批量解析
func (po *ParsingOptimizer) parseBatchParallel(
	ctx context.Context,
	requests []*BatchParseRequest,
) ([]*BatchParseResult, error) {

	results := make([]*BatchParseResult, len(requests))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, po.config.MaxConcurrency)

	// 分批处理
	for i := 0; i < len(requests); i += po.config.BatchSize {
		end := i + po.config.BatchSize
		if end > len(requests) {
			end = len(requests)
		}

		batch := requests[i:end]

		for idx, req := range batch {
			wg.Add(1)
			go func(j int, r *BatchParseRequest) {
				defer wg.Done()

				// 获取信号量
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				startTime := time.Now()

				programAST, err := r.Parser.Parse(ctx, r.SourceCode, r.Filename)

				duration := time.Since(startTime)

				result := &BatchParseResult{
					Filename:   r.Filename,
					ProgramAST: programAST,
					Success:    err == nil,
					Error:      err,
					Duration:   duration,
				}

				results[i+j] = result

				// 更新统计
				if po.config.EnableStats {
					po.updateStats(duration, err == nil, programAST)
				}
			}(idx, req)
		}

		wg.Wait()
	}

	return results, nil
}

// ParseInParallel 并行解析多个文件
func (po *ParsingOptimizer) ParseInParallel(
	ctx context.Context,
	parserAggregate *parser.ModernParserAggregate,
	sourceFiles map[string]string, // filename -> sourceCode
) (map[string]*BatchParseResult, error) {

	results := make(map[string]*BatchParseResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, po.config.MaxConcurrency)

	for filename, sourceCode := range sourceFiles {
		wg.Add(1)
		go func(fn string, code string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 为每个文件创建新的解析器实例（避免状态冲突）
			parserInstance := parser.NewModernParserAggregate()

			startTime := time.Now()
			programAST, err := parserInstance.Parse(ctx, code, fn)
			duration := time.Since(startTime)

			result := &BatchParseResult{
				Filename:   fn,
				ProgramAST: programAST,
				Success:    err == nil,
				Error:      err,
				Duration:   duration,
			}

			mu.Lock()
			results[fn] = result
			mu.Unlock()

			// 更新统计
			if po.config.EnableStats {
				po.updateStats(duration, err == nil, programAST)
			}
		}(filename, sourceCode)
	}

	wg.Wait()

	return results, nil
}

// updateStats 更新性能统计
func (po *ParsingOptimizer) updateStats(
	duration time.Duration,
	success bool,
	programAST *sharedVO.ProgramAST,
) {
	po.statsMutex.Lock()
	defer po.statsMutex.Unlock()

	po.stats.TotalParses++
	if success {
		po.stats.SuccessfulParses++
	} else {
		po.stats.FailedParses++
	}

	po.stats.TotalDuration += duration

	// 更新平均耗时
	if po.stats.TotalParses > 0 {
		po.stats.AverageDuration = po.stats.TotalDuration / time.Duration(po.stats.TotalParses)
	}

	// 更新最大耗时
	if duration > po.stats.MaxDuration {
		po.stats.MaxDuration = duration
	}

	// 更新最小耗时
	if duration < po.stats.MinDuration {
		po.stats.MinDuration = duration
	}

	// 统计Token数量：使用AST节点数量作为Token数量的近似值
	// 注意：这是一个近似值，因为一个Token可能对应多个AST节点，或者多个Token对应一个AST节点
	// 但对于性能统计来说，这个近似值是合理的
	if programAST != nil {
		tokenCount := po.countASTNodes(programAST)
		po.stats.TotalTokens += int64(tokenCount)
		if po.stats.TotalParses > 0 {
			po.stats.AverageTokensPerParse = po.stats.TotalTokens / po.stats.TotalParses
		}
	}
}

// countASTNodes 递归统计AST节点数量（作为Token数量的近似值）
func (po *ParsingOptimizer) countASTNodes(programAST *sharedVO.ProgramAST) int {
	if programAST == nil {
		return 0
	}

	count := 1 // 程序根节点

	// 递归统计所有顶层节点及其子节点
	for _, node := range programAST.Nodes() {
		count += po.countNodeRecursive(node)
	}

	// 统计模块声明
	if programAST.ModuleDeclaration() != nil {
		count += po.countNodeRecursive(programAST.ModuleDeclaration())
	}

	// 统计导入语句
	for _, importStmt := range programAST.Imports() {
		count += po.countNodeRecursive(importStmt)
	}

	return count
}

// countNodeRecursive 递归统计单个节点及其所有子节点
func (po *ParsingOptimizer) countNodeRecursive(node sharedVO.ASTNode) int {
	if node == nil {
		return 0
	}

	count := 1 // 当前节点

	// 根据节点类型递归统计子节点
	switch n := node.(type) {
	case *sharedVO.BlockStatement:
		// 统计块语句中的所有语句
		for _, stmt := range n.Statements() {
			count += po.countNodeRecursive(stmt)
		}

	case *sharedVO.FunctionDeclaration:
		// 统计函数体的子节点
		if n.Body() != nil {
			count += po.countNodeRecursive(n.Body())
		}
		// 统计参数
		for _, param := range n.Parameters() {
			count += po.countNodeRecursive(param)
		}
		// 统计泛型参数
		for _, gp := range n.GenericParams() {
			count += po.countNodeRecursive(gp)
		}
		// 统计返回类型
		if n.ReturnType() != nil {
			count += po.countNodeRecursive(n.ReturnType())
		}

	case *sharedVO.AsyncFunctionDeclaration:
		// 统计异步函数的基础函数声明
		if n.FunctionDeclaration() != nil {
			count += po.countNodeRecursive(n.FunctionDeclaration())
		}

	case *sharedVO.StructDeclaration:
		// 统计结构体字段
		for _, field := range n.Fields() {
			count += po.countNodeRecursive(field)
		}

	case *sharedVO.EnumDeclaration:
		// 统计枚举变体
		for _, variant := range n.Variants() {
			count += po.countNodeRecursive(variant)
		}

	case *sharedVO.TraitDeclaration:
		// 统计Trait方法
		for _, method := range n.Methods() {
			count += po.countNodeRecursive(method)
		}
		// 统计泛型参数
		for _, gp := range n.GenericParams() {
			count += po.countNodeRecursive(gp)
		}

	case *sharedVO.ImplDeclaration:
		// 统计实现的方法
		for _, method := range n.Methods() {
			count += po.countNodeRecursive(method)
		}
		// 统计Trait类型和目标类型
		if n.TraitType() != nil {
			count += po.countNodeRecursive(n.TraitType())
		}
		if n.TargetType() != nil {
			count += po.countNodeRecursive(n.TargetType())
		}

	case *sharedVO.ReturnStatement:
		// 统计返回表达式
		if n.Expression() != nil {
			count += po.countNodeRecursive(n.Expression())
		}

	case *sharedVO.VariableDeclaration:
		// 统计初始化表达式
		if n.Initializer() != nil {
			count += po.countNodeRecursive(n.Initializer())
		}
		// 统计类型注解
		if n.VarType() != nil {
			count += po.countNodeRecursive(n.VarType())
		}

	case *sharedVO.BinaryExpression:
		// 统计左右操作数
		if n.Left() != nil {
			count += po.countNodeRecursive(n.Left())
		}
		if n.Right() != nil {
			count += po.countNodeRecursive(n.Right())
		}

	case *sharedVO.UnaryExpression:
		// 统计操作数
		if n.Operand() != nil {
			count += po.countNodeRecursive(n.Operand())
		}

	case *sharedVO.TypeAnnotation:
		// 统计泛型参数
		for _, arg := range n.GenericArgs() {
			count += po.countNodeRecursive(arg)
		}

	case *sharedVO.Parameter:
		// 统计参数类型注解
		if n.TypeAnnotation() != nil {
			count += po.countNodeRecursive(n.TypeAnnotation())
		}

	case *sharedVO.StructField:
		// 统计字段类型注解
		if n.FieldType() != nil {
			count += po.countNodeRecursive(n.FieldType())
		}

	case *sharedVO.TraitMethod:
		// 统计参数
		for _, param := range n.Parameters() {
			count += po.countNodeRecursive(param)
		}
		// 统计返回类型
		if n.ReturnType() != nil {
			count += po.countNodeRecursive(n.ReturnType())
		}

	case *sharedVO.ObjectProperty:
		// 统计属性值
		if n.Value() != nil {
			count += po.countNodeRecursive(n.Value())
		}
		
	case *sharedVO.ExpressionStatement:
		// 统计表达式
		if n.Expression() != nil {
			count += po.countNodeRecursive(n.Expression())
		}
		
	case *sharedVO.TypeAliasDeclaration:
		// 统计原始类型
		if n.OriginalType() != nil {
			count += po.countNodeRecursive(n.OriginalType())
		}
	}

	return count
}

// GetStats 获取性能统计
func (po *ParsingOptimizer) GetStats() *ParsingPerformanceStats {
	po.statsMutex.RLock()
	defer po.statsMutex.RUnlock()

	// 返回副本
	stats := *po.stats
	return &stats
}

// ResetStats 重置统计
func (po *ParsingOptimizer) ResetStats() {
	po.statsMutex.Lock()
	defer po.statsMutex.Unlock()

	po.stats = &ParsingPerformanceStats{
		MinDuration: time.Hour,
	}
}

// GetOptimizationRecommendations 获取优化建议
func (po *ParsingOptimizer) GetOptimizationRecommendations() []string {
	recommendations := make([]string, 0)

	stats := po.GetStats()

	// 如果平均耗时过长，建议启用并行处理
	if stats.AverageDuration > 1*time.Second && !po.config.EnableParallel {
		recommendations = append(recommendations, "考虑启用并行解析以提高性能")
	}

	// 如果失败率过高，建议检查错误恢复策略
	if stats.TotalParses > 0 {
		failureRate := float64(stats.FailedParses) / float64(stats.TotalParses)
		if failureRate > 0.1 {
			recommendations = append(recommendations, "解析失败率较高，建议检查错误恢复策略")
		}
	}

	// 如果最大耗时远大于平均耗时，建议优化慢速文件
	if stats.MaxDuration > stats.AverageDuration*3 {
		recommendations = append(recommendations, "存在解析耗时异常的文件，建议优化")
	}

	return recommendations
}
