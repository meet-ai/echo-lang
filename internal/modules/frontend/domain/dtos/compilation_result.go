package dtos

// CompilationResult 编译结果DTO
type CompilationResult struct {
	SourceFile    string `json:"source_file"`
	AST           string `json:"ast"`
	GeneratedCode string `json:"generated_code"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
}

