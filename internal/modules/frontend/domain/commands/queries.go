package commands

// GetAnalysisStatusQuery 获取分析状态查询
type GetAnalysisStatusQuery struct {
	SourceFileID string `json:"source_file_id" validate:"required"`
}

// AnalysisStatusDTO 分析状态DTO
type AnalysisStatusDTO struct {
	SourceFileID    string `json:"source_file_id"`
	Status          string `json:"status"`
	LastAnalyzedAt  *string `json:"last_analyzed_at,omitempty"`
	ErrorCount      int     `json:"error_count"`
	HasAST          bool    `json:"has_ast"`
	HasSymbolTable  bool    `json:"has_symbol_table"`
}

// GetASTStructureQuery 获取AST结构查询
type GetASTStructureQuery struct {
	SourceFileID   string `json:"source_file_id" validate:"required"`
	IncludeDetails bool   `json:"include_details"`
}

// ASTStructureDTO AST结构DTO
type ASTStructureDTO struct {
	SourceFileID string                   `json:"source_file_id"`
	RootNode     *ASTNodeDTO              `json:"root_node,omitempty"`
	NodeCount    int                      `json:"node_count"`
	MaxDepth     int                      `json:"max_depth"`
	Functions    []FunctionSummaryDTO     `json:"functions"`
	Variables    []VariableSummaryDTO     `json:"variables"`
}

// ASTNodeDTO AST节点DTO
type ASTNodeDTO struct {
	Type       string                   `json:"type"`
	Children   []*ASTNodeDTO           `json:"children,omitempty"`
	Position   *PositionDTO            `json:"position,omitempty"`
	Attributes map[string]interface{}  `json:"attributes,omitempty"`
}

// PositionDTO 位置DTO
type PositionDTO struct {
	Line   int    `json:"line"`
	Column int    `json:"column"`
	File   string `json:"file"`
}

// FunctionSummaryDTO 函数摘要DTO
type FunctionSummaryDTO struct {
	Name       string      `json:"name"`
	ReturnType string      `json:"return_type"`
	Position   PositionDTO `json:"position"`
}

// VariableSummaryDTO 变量摘要DTO
type VariableSummaryDTO struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Position PositionDTO `json:"position"`
}
