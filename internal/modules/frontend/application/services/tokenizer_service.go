package services

import (
	"context"
	"fmt"

	"github.com/meetai/echo-lang/internal/modules/frontend"
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

// TokenizerService 应用查询服务实现
type TokenizerService interface {
	GetAnalysisStatus(ctx context.Context, query frontend.GetAnalysisStatusQuery) (*frontend.AnalysisStatusDTO, error)
	GetASTStructure(ctx context.Context, query frontend.GetASTStructureQuery) (*frontend.ASTStructureDTO, error)
}

// tokenizerService 查询服务实现
type tokenizerService struct {
	sourceFileRepo frontend.SourceFileRepository
	astRepo        frontend.ASTRepository
}

// NewTokenizerService 创建查询服务
func NewTokenizerService(
	sourceFileRepo frontend.SourceFileRepository,
	astRepo frontend.ASTRepository,
) TokenizerService {
	return &tokenizerService{
		sourceFileRepo: sourceFileRepo,
		astRepo:        astRepo,
	}
}

// GetAnalysisStatus 查询分析状态
func (s *tokenizerService) GetAnalysisStatus(ctx context.Context, query frontend.GetAnalysisStatusQuery) (*frontend.AnalysisStatusDTO, error) {
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
	dto := &frontend.AnalysisStatusDTO{
		SourceFileID:   sourceFile.ID(),
		FilePath:       sourceFile.FilePath(),
		AnalysisStatus: string(sourceFile.AnalysisStatus()),
		TokenCount:     len(sourceFile.Tokens()),
		HasAST:         sourceFile.AST() != nil,
		HasSymbolTable: sourceFile.SymbolTable() != nil,
		ErrorCount:     len(sourceFile.ErrorMessages()),
		Errors:         sourceFile.ErrorMessages(),
		LastAnalyzedAt: sourceFile.LastAnalyzedAt(),
	}

	return dto, nil
}

// GetASTStructure 查询AST结构
func (s *tokenizerService) GetASTStructure(ctx context.Context, query frontend.GetASTStructureQuery) (*frontend.ASTStructureDTO, error) {
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
	astDTO, err := s.buildASTStructureDTO(sourceFile.AST(), query.IncludeDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to build AST structure: %w", err)
	}

	return astDTO, nil
}

// Helper methods

func (s *tokenizerService) validateGetAnalysisStatusQuery(query frontend.GetAnalysisStatusQuery) error {
	if query.SourceFileID == "" {
		return fmt.Errorf("source file ID is required")
	}
	return nil
}

func (s *tokenizerService) validateGetASTStructureQuery(query frontend.GetASTStructureQuery) error {
	if query.SourceFileID == "" {
		return fmt.Errorf("source file ID is required")
	}
	return nil
}

func (s *tokenizerService) buildASTStructureDTO(ast *entities.ASTNode, includeDetails bool) (*frontend.ASTStructureDTO, error) {
	if ast == nil {
		return &frontend.ASTStructureDTO{
			NodeType: "null",
			Children: []frontend.ASTStructureDTO{},
		}, nil
	}

	dto := &frontend.ASTStructureDTO{
		NodeType: string(ast.NodeType()),
		Children: make([]frontend.ASTStructureDTO, 0, len(ast.Children())),
	}

	if includeDetails {
		// Include additional details based on node type
		switch ast.NodeType() {
		case entities.ASTNodeTypeFunction:
			if fn, ok := ast.(*entities.FunctionNode); ok {
				dto.Details = map[string]interface{}{
					"name":       fn.Name(),
					"parameters": fn.Parameters(),
					"returnType": fn.ReturnType(),
				}
			}
		case entities.ASTNodeTypeVariable:
			if vn, ok := ast.(*entities.VariableNode); ok {
				dto.Details = map[string]interface{}{
					"name":  vn.Name(),
					"type":  vn.VariableType(),
					"value": vn.Value(),
				}
			}
		}
	}

	// Recursively build children
	for _, child := range ast.Children() {
		childDTO, err := s.buildASTStructureDTO(child, includeDetails)
		if err != nil {
			return nil, err
		}
		dto.Children = append(dto.Children, *childDTO)
	}

	return dto, nil
}
