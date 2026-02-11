package semantic

import (
	"context"

	"echo/internal/modules/frontend/domain/semantic/services"
	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// SemanticAnalysisApplicationService 语义分析应用服务
// 职责：编排“构建符号表 -> 类型检查 -> 符号解析”的完整语义分析流程
type SemanticAnalysisApplicationService struct {
	analyzer *services.SemanticAnalyzer
}

// NewSemanticAnalysisApplicationService 创建语义分析应用服务
func NewSemanticAnalysisApplicationService(
	analyzer *services.SemanticAnalyzer,
) *SemanticAnalysisApplicationService {
	return &SemanticAnalysisApplicationService{
		analyzer: analyzer,
	}
}

// AnalyzeProgram 使用三阶段流水线分析程序的语义
func (s *SemanticAnalysisApplicationService) AnalyzeProgram(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
) (*value_objects.SemanticAnalysisResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	result := value_objects.NewSemanticAnalysisResult(s.analyzer.SymbolTable())

	if err := s.analyzer.BuildSymbolTable(ctx, programAST, result); err != nil {
		return result, err
	}
	if err := s.analyzer.PerformTypeChecking(ctx, programAST, result); err != nil {
		return result, err
	}
	if err := s.analyzer.PerformSymbolResolution(ctx, programAST, result); err != nil {
		return result, err
	}

	return result, nil
}

