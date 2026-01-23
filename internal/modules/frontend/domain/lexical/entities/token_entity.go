// Package entities 定义词法分析上下文的实体
package entities

import (
	"fmt"
	"time"

	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// TokenEntity Token实体（聚合根）
// TokenEntity 是词法分析的核心实体，负责管理Token的生命周期
type TokenEntity struct {
	id       string
	token    *value_objects.Token
	isValid  bool
	validatedAt *time.Time
}

// NewTokenEntity 创建新的Token实体
func NewTokenEntity(id string, token *value_objects.Token) *TokenEntity {
	now := time.Now()
	return &TokenEntity{
		id:          id,
		token:       token,
		isValid:     false,
		validatedAt: &now,
	}
}

// ID 获取实体ID
func (te *TokenEntity) ID() string {
	return te.id
}

// Token 获取Token值对象
func (te *TokenEntity) Token() *value_objects.Token {
	return te.token
}

// IsValid 检查Token是否有效
func (te *TokenEntity) IsValid() bool {
	return te.isValid
}

// Validate 验证Token
func (te *TokenEntity) Validate() error {
	// 验证Token的基本属性
	if te.token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	if te.token.Lexeme() == "" {
		return fmt.Errorf("token lexeme cannot be empty")
	}

	if te.token.Type() == "" {
		return fmt.Errorf("token type cannot be empty")
	}

	// 检查位置信息
	location := te.token.Location()
	if !location.IsValid() {
		return fmt.Errorf("token location is invalid")
	}

	// 验证通过
	now := time.Now()
	te.isValid = true
	te.validatedAt = &now

	return nil
}

// ValidatedAt 获取验证时间
func (te *TokenEntity) ValidatedAt() *time.Time {
	return te.validatedAt
}

// String 返回实体的字符串表示
func (te *TokenEntity) String() string {
	return fmt.Sprintf("TokenEntity{ID: %s, Token: %s, Valid: %t}",
		te.id, te.token.String(), te.isValid)
}

// TokenStreamEntity Token流实体（聚合根）
// TokenStreamEntity 管理Token流的整体状态和操作
type TokenStreamEntity struct {
	id          string
	sourceFile  *value_objects.SourceFile
	tokens      []*TokenEntity
	currentPos  int
	isComplete  bool
	createdAt   time.Time
}

// NewTokenStreamEntity 创建新的Token流实体
func NewTokenStreamEntity(id string, sourceFile *value_objects.SourceFile) *TokenStreamEntity {
	return &TokenStreamEntity{
		id:         id,
		sourceFile: sourceFile,
		tokens:     make([]*TokenEntity, 0),
		currentPos: 0,
		isComplete: false,
		createdAt:  time.Now(),
	}
}

// ID 获取实体ID
func (tse *TokenStreamEntity) ID() string {
	return tse.id
}

// SourceFile 获取源文件
func (tse *TokenStreamEntity) SourceFile() *value_objects.SourceFile {
	return tse.sourceFile
}

// AddToken 添加Token到流中
func (tse *TokenStreamEntity) AddToken(tokenEntity *TokenEntity) {
	tse.tokens = append(tse.tokens, tokenEntity)
}

// Tokens 获取所有Token实体
func (tse *TokenStreamEntity) Tokens() []*TokenEntity {
	return tse.tokens
}

// Count 获取Token数量
func (tse *TokenStreamEntity) Count() int {
	return len(tse.tokens)
}

// CurrentToken 获取当前位置的Token
func (tse *TokenStreamEntity) CurrentToken() *TokenEntity {
	if tse.currentPos >= len(tse.tokens) {
		return nil
	}
	return tse.tokens[tse.currentPos]
}

// NextToken 移动到下一个Token
func (tse *TokenStreamEntity) NextToken() *TokenEntity {
	if tse.currentPos >= len(tse.tokens) {
		return nil
	}
	token := tse.tokens[tse.currentPos]
	tse.currentPos++
	return token
}

// PeekToken 查看下一个Token但不移动位置
func (tse *TokenStreamEntity) PeekToken() *TokenEntity {
	if tse.currentPos >= len(tse.tokens) {
		return nil
	}
	return tse.tokens[tse.currentPos]
}

// ResetPosition 重置位置到开头
func (tse *TokenStreamEntity) ResetPosition() {
	tse.currentPos = 0
}

// CurrentPosition 获取当前位置
func (tse *TokenStreamEntity) CurrentPosition() int {
	return tse.currentPos
}

// IsAtEnd 检查是否到达流末尾
func (tse *TokenStreamEntity) IsAtEnd() bool {
	return tse.currentPos >= len(tse.tokens)
}

// Complete 标记Token流完成
func (tse *TokenStreamEntity) Complete() {
	tse.isComplete = true
}

// IsComplete 检查Token流是否完成
func (tse *TokenStreamEntity) IsComplete() bool {
	return tse.isComplete
}

// ValidateAllTokens 验证所有Token
func (tse *TokenStreamEntity) ValidateAllTokens() []*value_objects.ParseError {
	errors := make([]*value_objects.ParseError, 0)

	for _, tokenEntity := range tse.tokens {
		if err := tokenEntity.Validate(); err != nil {
			parseError := value_objects.NewParseError(
				fmt.Sprintf("invalid token %s: %v", tokenEntity.ID(), err),
				tokenEntity.Token().Location(),
				value_objects.ErrorTypeLexical,
				value_objects.SeverityError,
			)
			errors = append(errors, parseError)
		}
	}

	return errors
}

// CreatedAt 获取创建时间
func (tse *TokenStreamEntity) CreatedAt() time.Time {
	return tse.createdAt
}

// String 返回实体的字符串表示
func (tse *TokenStreamEntity) String() string {
	return fmt.Sprintf("TokenStreamEntity{ID: %s, TokenCount: %d, Complete: %t}",
		tse.id, tse.Count(), tse.isComplete)
}
