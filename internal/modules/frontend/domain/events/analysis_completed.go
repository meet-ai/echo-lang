package events

import (
	"time"

	"github.com/meetai/echo-lang/internal/modules/frontend"
)

// AnalysisCompleted 分析完成事件
type AnalysisCompleted struct {
	// 事件标识
	eventID    string
	eventType  string
	occurredAt time.Time

	// 分析上下文
	sourceFileID string
	analysisType string

	// 分析结果
	success  bool
	errorMsg string
	duration time.Duration

	// 分析产物统计
	tokenCount   int
	astNodeCount int
	symbolCount  int

	// 源文件信息
	filePath string
	fileSize int64
}

// NewAnalysisCompleted 创建分析完成事件
func NewAnalysisCompleted(
	sourceFileID, analysisType, filePath string,
	success bool, errorMsg string, duration time.Duration,
	tokenCount, astNodeCount, symbolCount int, fileSize int64,
) *AnalysisCompleted {
	return &AnalysisCompleted{
		eventID:      generateEventID("analysis_completed"),
		eventType:    "frontend.analysis.completed",
		occurredAt:   time.Now(),
		sourceFileID: sourceFileID,
		analysisType: analysisType,
		success:      success,
		errorMsg:     errorMsg,
		duration:     duration,
		tokenCount:   tokenCount,
		astNodeCount: astNodeCount,
		symbolCount:  symbolCount,
		filePath:     filePath,
		fileSize:     fileSize,
	}
}

// EventID 返回事件ID
func (e *AnalysisCompleted) EventID() string {
	return e.eventID
}

// EventType 返回事件类型
func (e *AnalysisCompleted) EventType() string {
	return e.eventType
}

// OccurredAt 返回事件发生时间
func (e *AnalysisCompleted) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID 返回聚合根ID
func (e *AnalysisCompleted) AggregateID() string {
	return e.sourceFileID
}

// SourceFileID 返回源文件ID
func (e *AnalysisCompleted) SourceFileID() string {
	return e.sourceFileID
}

// AnalysisType 返回分析类型
func (e *AnalysisCompleted) AnalysisType() string {
	return e.analysisType
}

// Success 返回分析是否成功
func (e *AnalysisCompleted) Success() bool {
	return e.success
}

// ErrorMsg 返回错误消息
func (e *AnalysisCompleted) ErrorMsg() string {
	return e.errorMsg
}

// Duration 返回分析耗时
func (e *AnalysisCompleted) Duration() time.Duration {
	return e.duration
}

// TokenCount 返回Token数量
func (e *AnalysisCompleted) TokenCount() int {
	return e.tokenCount
}

// ASTNodeCount 返回AST节点数量
func (e *AnalysisCompleted) ASTNodeCount() int {
	return e.astNodeCount
}

// SymbolCount 返回符号数量
func (e *AnalysisCompleted) SymbolCount() int {
	return e.symbolCount
}

// FilePath 返回文件路径
func (e *AnalysisCompleted) FilePath() string {
	return e.filePath
}

// FileSize 返回文件大小
func (e *AnalysisCompleted) FileSize() int64 {
	return e.fileSize
}

// IsErrorEvent 判断是否为错误事件
func (e *AnalysisCompleted) IsErrorEvent() bool {
	return !e.success
}

// AnalysisPhase 返回分析阶段
func (e *AnalysisCompleted) AnalysisPhase() string {
	switch e.analysisType {
	case "lexical":
		return "词法分析"
	case "syntax":
		return "语法分析"
	case "semantic":
		return "语义分析"
	default:
		return "未知分析"
	}
}

// PerformanceMetrics 返回性能指标
func (e *AnalysisCompleted) PerformanceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"duration_ms":     e.duration.Milliseconds(),
		"tokens_per_sec":  float64(e.tokenCount) / e.duration.Seconds(),
		"nodes_per_sec":   float64(e.astNodeCount) / e.duration.Seconds(),
		"symbols_per_sec": float64(e.symbolCount) / e.duration.Seconds(),
		"file_size_kb":    float64(e.fileSize) / 1024.0,
	}
}

// generateEventID 生成事件ID
func generateEventID(eventType string) string {
	return eventType + "-" + time.Now().Format("20060102-150405-000000000")
}
