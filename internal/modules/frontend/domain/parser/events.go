package parser

import (
	"time"

	"github.com/google/uuid"
)

// Event 领域事件接口
type Event interface {
	GetEventType() string
	GetEventVersion() string
	GetTimestamp() time.Time
}

// ProgramParsed 程序解析完成事件
type ProgramParsed struct {
	EventID         string
	ProgramID       string
	StatementsCount int
	FunctionsCount  int
	ParseTime       time.Duration
	SourceLines     int
	Timestamp       time.Time
}

func (e ProgramParsed) GetEventType() string    { return "parser.program.parsed" }
func (e ProgramParsed) GetEventVersion() string { return "1.0" }
func (e ProgramParsed) GetTimestamp() time.Time { return e.Timestamp }

// StatementParsed 语句解析完成事件
type StatementParsed struct {
	EventID        string
	StatementID    string
	StatementType  string // "if", "while", "let", "print"等
	LineNumber     int
	HasExpressions bool
	Complexity     int // 语句复杂度评分
	Timestamp      time.Time
}

func (e StatementParsed) GetEventType() string    { return "parser.statement.parsed" }
func (e StatementParsed) GetEventVersion() string { return "1.0" }
func (e StatementParsed) GetTimestamp() time.Time { return e.Timestamp }

// ExpressionParsed 表达式解析完成事件
type ExpressionParsed struct {
	EventID        string
	ExpressionID   string
	ExpressionType string // "binary", "call", "literal"等
	OperatorCount  int    // 运算符数量
	NestingLevel   int    // 嵌套层级
	Timestamp      time.Time
}

func (e ExpressionParsed) GetEventType() string    { return "parser.expression.parsed" }
func (e ExpressionParsed) GetEventVersion() string { return "1.0" }
func (e ExpressionParsed) GetTimestamp() time.Time { return e.Timestamp }

// BlockExtracted 代码块提取完成事件
type BlockExtracted struct {
	EventID   string
	BlockType string // "if", "while", "function"等
	StartLine int
	EndLine   int
	LineCount int
	HasNested bool // 是否包含嵌套块
	Timestamp time.Time
}

func (e BlockExtracted) GetEventType() string    { return "parser.block.extracted" }
func (e BlockExtracted) GetEventVersion() string { return "1.0" }
func (e BlockExtracted) GetTimestamp() time.Time { return e.Timestamp }

// SyntaxErrorOccurred 语法错误发生事件
type SyntaxErrorOccurred struct {
	EventID      string
	ErrorMessage string
	ErrorCode    string // "UNEXPECTED_TOKEN", "BRACE_MISMATCH"等
	LineNumber   int
	ColumnNumber int
	ContextLine  string // 出错行的内容
	Severity     string // "error", "warning"
	Timestamp    time.Time
}

func (e SyntaxErrorOccurred) GetEventType() string    { return "parser.syntax.error" }
func (e SyntaxErrorOccurred) GetEventVersion() string { return "1.0" }
func (e SyntaxErrorOccurred) GetTimestamp() time.Time { return e.Timestamp }

// ParseCompleted 解析过程完成事件
type ParseCompleted struct {
	EventID      string
	ProgramID    string
	Success      bool
	TotalLines   int
	ErrorCount   int
	WarningCount int
	Duration     time.Duration
	Timestamp    time.Time
}

func (e ParseCompleted) GetEventType() string    { return "parser.parse.completed" }
func (e ParseCompleted) GetEventVersion() string { return "1.0" }
func (e ParseCompleted) GetTimestamp() time.Time { return e.Timestamp }

// EventBus 事件总线接口
type EventBus interface {
	Publish(event Event) error
	Subscribe(eventType string, handler func(Event)) error
}

// EventPublisher 事件发布器
type EventPublisher struct {
	bus EventBus
}

func NewEventPublisher(bus EventBus) *EventPublisher {
	return &EventPublisher{bus: bus}
}

func (ep *EventPublisher) PublishProgramParsed(programID string, statementsCount int, parseTime time.Duration, sourceLines int) error {
	event := ProgramParsed{
		EventID:         uuid.New().String(),
		ProgramID:       programID,
		StatementsCount: statementsCount,
		ParseTime:       parseTime,
		SourceLines:     sourceLines,
		Timestamp:       time.Now(),
	}
	return ep.bus.Publish(event)
}

func (ep *EventPublisher) PublishStatementParsed(statementID string, statementType string, lineNumber int, hasExpressions bool, complexity int) error {
	event := StatementParsed{
		EventID:        uuid.New().String(),
		StatementID:    statementID,
		StatementType:  statementType,
		LineNumber:     lineNumber,
		HasExpressions: hasExpressions,
		Complexity:     complexity,
		Timestamp:      time.Now(),
	}
	return ep.bus.Publish(event)
}

func (ep *EventPublisher) PublishSyntaxError(errorMessage string, errorCode string, lineNumber int, columnNumber int, contextLine string, severity string) error {
	event := SyntaxErrorOccurred{
		EventID:      uuid.New().String(),
		ErrorMessage: errorMessage,
		ErrorCode:    errorCode,
		LineNumber:   lineNumber,
		ColumnNumber: columnNumber,
		ContextLine:  contextLine,
		Severity:     severity,
		Timestamp:    time.Now(),
	}
	return ep.bus.Publish(event)
}
