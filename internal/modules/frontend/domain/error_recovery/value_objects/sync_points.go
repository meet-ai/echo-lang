// Package value_objects 定义同步点相关值对象
package value_objects

import (
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
)

// SyncPointType 同步点类型
type SyncPointType string

const (
	// SyncPointSemicolon 分号同步点
	SyncPointSemicolon SyncPointType = "semicolon"

	// SyncPointBrace 花括号同步点
	SyncPointBrace SyncPointType = "brace"

	// SyncPointKeyword 关键字同步点
	SyncPointKeyword SyncPointType = "keyword"

	// SyncPointDeclaration 声明同步点
	SyncPointDeclaration SyncPointType = "declaration"

	// SyncPointEOF 文件结束同步点
	SyncPointEOF SyncPointType = "eof"
)

// SyncPoint 同步点值对象
// 同步点是解析器在遇到错误时可以安全跳转到的位置
type SyncPoint struct {
	pointType  SyncPointType
	tokenType  lexicalVO.EnhancedTokenType
	keywords   []string // 对于关键字类型的同步点
	strength   int      // 同步强度（数值越大越可靠）
}

// NewSyncPoint 创建新的同步点
func NewSyncPoint(pointType SyncPointType, tokenType lexicalVO.EnhancedTokenType, strength int) *SyncPoint {
	return &SyncPoint{
		pointType: pointType,
		tokenType: tokenType,
		strength:  strength,
	}
}

// NewKeywordSyncPoint 创建关键字同步点
func NewKeywordSyncPoint(keywords []string, strength int) *SyncPoint {
	return &SyncPoint{
		pointType: SyncPointKeyword,
		keywords:  keywords,
		strength:  strength,
	}
}

// PointType 获取同步点类型
func (sp *SyncPoint) PointType() SyncPointType {
	return sp.pointType
}

// TokenType 获取对应的Token类型
func (sp *SyncPoint) TokenType() lexicalVO.EnhancedTokenType {
	return sp.tokenType
}

// Keywords 获取关键字列表
func (sp *SyncPoint) Keywords() []string {
	return sp.keywords
}

// Strength 获取同步强度
func (sp *SyncPoint) Strength() int {
	return sp.strength
}

// IsMatch 检查Token是否匹配此同步点
func (sp *SyncPoint) IsMatch(token *lexicalVO.EnhancedToken) bool {
	switch sp.pointType {
	case SyncPointSemicolon:
		return token.Type() == lexicalVO.EnhancedTokenTypeSemicolon
	case SyncPointBrace:
		return token.Type() == lexicalVO.EnhancedTokenTypeLeftBrace ||
			   token.Type() == lexicalVO.EnhancedTokenTypeRightBrace
	case SyncPointKeyword:
		if token.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
			lexeme := token.Lexeme()
			for _, keyword := range sp.keywords {
				if lexeme == keyword {
					return true
				}
			}
		}
		return false
	case SyncPointDeclaration:
		// 声明关键字：fn, let, struct, enum, trait, impl
		declarationKeywords := []string{"fn", "let", "struct", "enum", "trait", "impl"}
		if token.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
			lexeme := token.Lexeme()
			for _, keyword := range declarationKeywords {
				if lexeme == keyword {
					return true
				}
			}
		}
		return false
	case SyncPointEOF:
		return token.Type() == lexicalVO.EnhancedTokenTypeEOF
	default:
		return token.Type() == sp.tokenType
	}
}

// String 返回同步点的字符串表示
func (sp *SyncPoint) String() string {
	return string(sp.pointType) + "(strength:" + string(rune(sp.strength+'0')) + ")"
}

// SyncPointRegistry 同步点注册表值对象
// 管理所有预定义的同步点
type SyncPointRegistry struct {
	syncPoints []*SyncPoint
}

// NewSyncPointRegistry 创建新的同步点注册表
func NewSyncPointRegistry() *SyncPointRegistry {
	registry := &SyncPointRegistry{
		syncPoints: make([]*SyncPoint, 0),
	}

	// 注册标准同步点
	registry.registerStandardSyncPoints()

	return registry
}

// RegisterSyncPoint 注册同步点
func (spr *SyncPointRegistry) RegisterSyncPoint(syncPoint *SyncPoint) {
	spr.syncPoints = append(spr.syncPoints, syncPoint)
}

// FindSyncPoints 查找匹配的同步点
func (spr *SyncPointRegistry) FindSyncPoints(token *lexicalVO.EnhancedToken) []*SyncPoint {
	matchingPoints := make([]*SyncPoint, 0)

	for _, syncPoint := range spr.syncPoints {
		if syncPoint.IsMatch(token) {
			matchingPoints = append(matchingPoints, syncPoint)
		}
	}

	return matchingPoints
}

// GetStrongestSyncPoint 获取最强的同步点
func (spr *SyncPointRegistry) GetStrongestSyncPoint(token *lexicalVO.EnhancedToken) *SyncPoint {
	matchingPoints := spr.FindSyncPoints(token)
	if len(matchingPoints) == 0 {
		return nil
	}

	strongest := matchingPoints[0]
	for _, point := range matchingPoints[1:] {
		if point.Strength() > strongest.Strength() {
			strongest = point
		}
	}

	return strongest
}

// AllSyncPoints 获取所有同步点
func (spr *SyncPointRegistry) AllSyncPoints() []*SyncPoint {
	return spr.syncPoints
}

// registerStandardSyncPoints 注册标准同步点
func (spr *SyncPointRegistry) registerStandardSyncPoints() {
	// 分号同步点 - 语句结束
	spr.RegisterSyncPoint(NewSyncPoint(SyncPointSemicolon, lexicalVO.EnhancedTokenTypeSemicolon, 9))

	// 花括号同步点 - 块结构边界
	spr.RegisterSyncPoint(NewSyncPoint(SyncPointBrace, lexicalVO.EnhancedTokenTypeLeftBrace, 8))
	spr.RegisterSyncPoint(NewSyncPoint(SyncPointBrace, lexicalVO.EnhancedTokenTypeRightBrace, 8))

	// 声明关键字同步点 - 新的声明开始
	declarationKeywords := []string{"fn", "let", "struct", "enum", "trait", "impl"}
	spr.RegisterSyncPoint(NewKeywordSyncPoint(declarationKeywords, 7))

	// 控制流关键字同步点
	controlKeywords := []string{"if", "else", "for", "while", "return", "break", "continue"}
	spr.RegisterSyncPoint(NewKeywordSyncPoint(controlKeywords, 6))

	// EOF同步点 - 文件结束
	spr.RegisterSyncPoint(NewSyncPoint(SyncPointEOF, lexicalVO.EnhancedTokenTypeEOF, 10))
}
