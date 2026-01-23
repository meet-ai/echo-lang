package dtos

import "time"

// AnalysisResultDTO 分析结果DTO
type AnalysisResultDTO struct {
	Tokens   []TokenDTO `json:"tokens"`
	AST      string     `json:"ast,omitempty"` // 简化为字符串表示
	Errors   []ErrorDTO `json:"errors"`
	Warnings []string   `json:"warnings"`
}

// TokenDTO Token DTO
type TokenDTO struct {
	Type     string `json:"type"`
	Lexeme   string `json:"lexeme"`
	Position PositionDTO `json:"position"`
}

// PositionDTO 位置DTO
type PositionDTO struct {
	Line   int    `json:"line"`
	Column int    `json:"column"`
	File   string `json:"file"`
}

// ErrorDTO 错误DTO
type ErrorDTO struct {
	Message  string       `json:"message"`
	Position PositionDTO  `json:"position"`
	Severity string       `json:"severity"`
	Time     time.Time    `json:"time"`
}

