package services

import (
	"context"
	"fmt"

	"echo/internal/modules/frontend/domain/commands"
	"echo/internal/modules/frontend/domain/entities"
)

// TokenizerService 应用查询服务实现
type TokenizerService interface {
	GetAnalysisStatus(ctx context.Context, query commands.GetAnalysisStatusQuery) (*commands.AnalysisStatusDTO, error)
	GetASTStructure(ctx context.Context, query commands.GetASTStructureQuery) (*commands.ASTStructureDTO, error)
}

// tokenizerService 查询服务实现
type tokenizerService struct {
	sourceFileRepo SourceFileRepository
	astRepo        ASTRepository
}

// NewTokenizerService 创建查询服务
func NewTokenizerService(
	sourceFileRepo SourceFileRepository,
	astRepo ASTRepository,
) TokenizerService {
	return &tokenizerService{
		sourceFileRepo: sourceFileRepo,
		astRepo:        astRepo,
	}
}

// GetAnalysisStatus 查询分析状态
func (s *tokenizerService) GetAnalysisStatus(ctx context.Context, query commands.GetAnalysisStatusQuery) (*commands.AnalysisStatusDTO, error) {
	// 验证查询
	if err := s.validateGetAnalysisStatusQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	// 获取源文件
	sourceFile, err := s.sourceFileRepo.FindByID(ctx, query.SourceFileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find source file: %w", err)
	}

	// 构建DTO
	dto := &commands.AnalysisStatusDTO{
		SourceFileID:   sourceFile.ID(),
		Status:         string(sourceFile.AnalysisStatus()),
		ErrorCount:     len(sourceFile.ErrorMessages()),
		HasAST:         sourceFile.AST() != nil,
		HasSymbolTable: sourceFile.SymbolTable() != nil,
	}

	return dto, nil
}

// GetASTStructure 查询AST结构
func (s *tokenizerService) GetASTStructure(ctx context.Context, query commands.GetASTStructureQuery) (*commands.ASTStructureDTO, error) {
	// 验证查询
	if err := s.validateGetASTStructureQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	// 获取源文件
	sourceFile, err := s.sourceFileRepo.FindByID(ctx, query.SourceFileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find source file: %w", err)
	}

	// 检查是否有AST
	if sourceFile.AST() == nil {
		return nil, fmt.Errorf("AST not available for source file %s", query.SourceFileID)
	}

	// 构建AST结构DTO
	astDTO := s.buildASTStructureDTO(sourceFile.AST(), query.IncludeDetails)

	return astDTO, nil
}

// Helper methods

func (s *tokenizerService) validateGetAnalysisStatusQuery(query commands.GetAnalysisStatusQuery) error {
	if query.SourceFileID == "" {
		return fmt.Errorf("source file ID is required")
	}
	return nil
}

func (s *tokenizerService) validateGetASTStructureQuery(query commands.GetASTStructureQuery) error {
	if query.SourceFileID == "" {
		return fmt.Errorf("source file ID is required")
	}
	return nil
}

func (s *tokenizerService) buildASTStructureDTO(ast *entities.ASTNode, includeDetails bool) *commands.ASTStructureDTO {
	if ast == nil {
		return &commands.ASTStructureDTO{
			SourceFileID: "", // 暂时为空
			RootNode:     nil,
			NodeCount:    0,
			MaxDepth:     0,
			Functions:    []commands.FunctionSummaryDTO{},
			Variables:    []commands.VariableSummaryDTO{},
		}
	}

	// 暂时简化实现
	// TODO: 实现完整的 AST 结构构建
	dto := &commands.ASTStructureDTO{
		SourceFileID: "", // 暂时为空
		RootNode: &commands.ASTNodeDTO{
			Type:     "unknown", // 需要根据实际 AST 节点类型确定
			Children: []*commands.ASTNodeDTO{},
		},
		NodeCount: 1,
		MaxDepth:  1,
		Functions: []commands.FunctionSummaryDTO{},
		Variables: []commands.VariableSummaryDTO{},
	}

	return dto
}
