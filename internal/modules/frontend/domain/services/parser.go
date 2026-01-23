package services

import (
	"echo/internal/modules/frontend/domain/entities"
	"echo/internal/modules/frontend/domain/parser"
)

// SimpleParser 简单解析器实现 - DDD门面层
// 职责：作为应用服务的门面，委托给领域层的ParserAggregate处理
type SimpleParser struct {
	domainParser *parser.ParserAggregate // 领域层聚合根
}

// NewSimpleParser 创建新的简单解析器
func NewSimpleParser() Parser {
	return &SimpleParser{
		domainParser: parser.NewParserAggregate(),
	}
}

// Parse 解析源码字符串 - 门面方法，委托给领域层
func (sp *SimpleParser) Parse(content string) (*entities.Program, error) {
	// 直接委托给领域层的ParserAggregate
	return sp.domainParser.ParseProgram(content)
}
