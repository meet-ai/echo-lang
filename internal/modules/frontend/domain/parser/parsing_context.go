// Package parser 定义解析上下文管理
package parser

import (
	"fmt"
	"time"

	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// ParsingContext 解析上下文
// 管理解析过程中的状态、位置、错误等信息
type ParsingContext struct {
	// 上下文ID
	contextID string

	// 源文件信息
	sourceFile *lexicalVO.SourceFile

	// Token流
	tokenStream *lexicalVO.EnhancedTokenStream

	// 当前解析位置
	currentPosition int

	// 解析状态栈（支持嵌套状态）
	stateStack []ParserState

	// 当前解析器类型
	currentParserType ParserType

	// 错误收集
	errors []*sharedVO.ParseError

	// 恢复上下文
	recoveryContext *RecoveryContext

	// 上下文元数据
	metadata map[string]interface{}
}

// ParserType 解析器类型
type ParserType string

const (
	ParserTypeRecursiveDescent ParserType = "recursive_descent" // 递归下降解析器
	ParserTypePratt            ParserType = "pratt"             // Pratt表达式解析器
	ParserTypeLR               ParserType = "lr"                // LR歧义解析器
)

// ParserState 解析器状态
type ParserState struct {
	StateType   ParserType
	StateName   string
	Position    int
	Metadata    map[string]interface{}
	ParentState *ParserState // 父状态（用于嵌套）
}

// RecoveryContext 恢复上下文
type RecoveryContext struct {
	InRecoveryMode bool
	RecoveryCount  int
	LastError      *sharedVO.ParseError
}

// NewParsingContext 创建新的解析上下文
func NewParsingContext(
	sourceFile *lexicalVO.SourceFile,
	tokenStream *lexicalVO.EnhancedTokenStream,
) *ParsingContext {
	return &ParsingContext{
		contextID:         generateContextID(),
		sourceFile:        sourceFile,
		tokenStream:       tokenStream,
		currentPosition:   0,
		stateStack:        make([]ParserState, 0),
		currentParserType: ParserTypeRecursiveDescent,
		errors:            make([]*sharedVO.ParseError, 0),
		recoveryContext:   &RecoveryContext{InRecoveryMode: false, RecoveryCount: 0},
		metadata:          make(map[string]interface{}),
	}
}

// ContextID 获取上下文ID
func (pc *ParsingContext) ContextID() string {
	return pc.contextID
}

// SourceFile 获取源文件
func (pc *ParsingContext) SourceFile() *lexicalVO.SourceFile {
	return pc.sourceFile
}

// TokenStream 获取Token流
func (pc *ParsingContext) TokenStream() *lexicalVO.EnhancedTokenStream {
	return pc.tokenStream
}

// CurrentPosition 获取当前解析位置
func (pc *ParsingContext) CurrentPosition() int {
	return pc.currentPosition
}

// SetCurrentPosition 设置当前解析位置
func (pc *ParsingContext) SetCurrentPosition(position int) {
	pc.currentPosition = position
}

// AdvancePosition 前进解析位置
func (pc *ParsingContext) AdvancePosition(offset int) {
	pc.currentPosition += offset
}

// CurrentParserType 获取当前解析器类型
func (pc *ParsingContext) CurrentParserType() ParserType {
	return pc.currentParserType
}

// SetCurrentParserType 设置当前解析器类型
func (pc *ParsingContext) SetCurrentParserType(parserType ParserType) {
	pc.currentParserType = parserType
}

// PushState 推入新状态到状态栈
func (pc *ParsingContext) PushState(stateType ParserType, stateName string) {
	var parentState *ParserState
	if len(pc.stateStack) > 0 {
		parentState = &pc.stateStack[len(pc.stateStack)-1]
	}

	newState := ParserState{
		StateType:   stateType,
		StateName:   stateName,
		Position:    pc.currentPosition,
		Metadata:    make(map[string]interface{}),
		ParentState: parentState,
	}

	pc.stateStack = append(pc.stateStack, newState)
}

// PopState 弹出状态栈顶状态
func (pc *ParsingContext) PopState() *ParserState {
	if len(pc.stateStack) == 0 {
		return nil
	}

	topState := pc.stateStack[len(pc.stateStack)-1]
	pc.stateStack = pc.stateStack[:len(pc.stateStack)-1]

	return &topState
}

// CurrentState 获取当前状态
func (pc *ParsingContext) CurrentState() *ParserState {
	if len(pc.stateStack) == 0 {
		return nil
	}
	return &pc.stateStack[len(pc.stateStack)-1]
}

// StateStackDepth 获取状态栈深度
func (pc *ParsingContext) StateStackDepth() int {
	return len(pc.stateStack)
}

// AddError 添加错误
func (pc *ParsingContext) AddError(err *sharedVO.ParseError) {
	pc.errors = append(pc.errors, err)
}

// Errors 获取所有错误
func (pc *ParsingContext) Errors() []*sharedVO.ParseError {
	return pc.errors
}

// HasErrors 检查是否有错误
func (pc *ParsingContext) HasErrors() bool {
	return len(pc.errors) > 0
}

// ClearErrors 清除所有错误
func (pc *ParsingContext) ClearErrors() {
	pc.errors = make([]*sharedVO.ParseError, 0)
}

// RecoveryContext 获取恢复上下文
func (pc *ParsingContext) RecoveryContext() *RecoveryContext {
	return pc.recoveryContext
}

// EnterRecoveryMode 进入恢复模式
func (pc *ParsingContext) EnterRecoveryMode(err *sharedVO.ParseError) {
	pc.recoveryContext.InRecoveryMode = true
	pc.recoveryContext.RecoveryCount++
	pc.recoveryContext.LastError = err
}

// ExitRecoveryMode 退出恢复模式
func (pc *ParsingContext) ExitRecoveryMode() {
	pc.recoveryContext.InRecoveryMode = false
}

// IsInRecoveryMode 检查是否在恢复模式
func (pc *ParsingContext) IsInRecoveryMode() bool {
	return pc.recoveryContext.InRecoveryMode
}

// SetMetadata 设置元数据
func (pc *ParsingContext) SetMetadata(key string, value interface{}) {
	pc.metadata[key] = value
}

// GetMetadata 获取元数据
func (pc *ParsingContext) GetMetadata(key string) (interface{}, bool) {
	value, exists := pc.metadata[key]
	return value, exists
}

// Clone 克隆解析上下文（用于状态保存）
func (pc *ParsingContext) Clone() *ParsingContext {
	cloned := &ParsingContext{
		contextID:         pc.contextID,
		sourceFile:        pc.sourceFile,
		tokenStream:       pc.tokenStream,
		currentPosition:   pc.currentPosition,
		stateStack:        make([]ParserState, len(pc.stateStack)),
		currentParserType: pc.currentParserType,
		errors:            make([]*sharedVO.ParseError, len(pc.errors)),
		recoveryContext: &RecoveryContext{
			InRecoveryMode: pc.recoveryContext.InRecoveryMode,
			RecoveryCount:  pc.recoveryContext.RecoveryCount,
			LastError:      pc.recoveryContext.LastError,
		},
		metadata: make(map[string]interface{}),
	}

	// 深拷贝状态栈
	copy(cloned.stateStack, pc.stateStack)

	// 深拷贝错误列表
	copy(cloned.errors, pc.errors)

	// 深拷贝元数据
	for k, v := range pc.metadata {
		cloned.metadata[k] = v
	}

	return cloned
}

// SaveCheckpoint 保存检查点（用于回退）
func (pc *ParsingContext) SaveCheckpoint() *ParsingCheckpoint {
	return &ParsingCheckpoint{
		ContextID:       pc.contextID,
		Position:        pc.currentPosition,
		StateStack:      pc.Clone().stateStack,
		ParserType:      pc.currentParserType,
		RecoveryContext: *pc.recoveryContext,
		Metadata:        pc.metadata,
	}
}

// RestoreCheckpoint 恢复检查点
func (pc *ParsingContext) RestoreCheckpoint(checkpoint *ParsingCheckpoint) error {
	if checkpoint.ContextID != pc.contextID {
		return fmt.Errorf("checkpoint context ID mismatch: expected %s, got %s", pc.contextID, checkpoint.ContextID)
	}

	pc.currentPosition = checkpoint.Position
	pc.stateStack = checkpoint.StateStack
	pc.currentParserType = checkpoint.ParserType
	pc.recoveryContext = &checkpoint.RecoveryContext

	// 恢复元数据
	pc.metadata = make(map[string]interface{})
	for k, v := range checkpoint.Metadata {
		pc.metadata[k] = v
	}

	return nil
}

// ParsingCheckpoint 解析检查点
// 用于保存和恢复解析状态
type ParsingCheckpoint struct {
	ContextID       string
	Position        int
	StateStack      []ParserState
	ParserType      ParserType
	RecoveryContext RecoveryContext
	Metadata        map[string]interface{}
}

// String 返回上下文的字符串表示
func (pc *ParsingContext) String() string {
	return fmt.Sprintf("ParsingContext[ID:%s, Position:%d, Parser:%s, States:%d]",
		pc.contextID, pc.currentPosition, pc.currentParserType, len(pc.stateStack))
}

// generateContextID 生成上下文ID
func generateContextID() string {
	return fmt.Sprintf("ctx_%d", time.Now().UnixNano())
}
