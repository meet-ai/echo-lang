package parser

import (
	"echo/internal/modules/frontend/domain/entities"
)

// Parser 语法解析器接口 - 保持向后兼容性
type Parser interface {
	Parse(content string) (*entities.Program, error)
}

// SimpleParser 简单解析器实现 - 兼容性包装器
type SimpleParser struct {
	parser *ParserAggregate // 聚合根
}

// NewSimpleParser 创建新的简单解析器
func NewSimpleParser() Parser {
	return &SimpleParser{
		parser: NewParserAggregate(),
	}
}

// Parse 解析源码字符串 - 保持原有接口签名
func (sp *SimpleParser) Parse(content string) (*entities.Program, error) {
	return sp.parser.ParseProgram(content)
}

// NewParserAggregateAlias 创建Parser聚合根别名 - 用于需要更细粒度控制的场景
func NewParserAggregateAlias() *ParserAggregate {
	return NewParserAggregate()
}

// GetParserAggregate 获取聚合根实例 - 用于测试或其他需要访问内部状态的场景
func (sp *SimpleParser) GetParserAggregate() *ParserAggregate {
	return sp.parser
}
