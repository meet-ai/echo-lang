package repositories

import (
	"context"
	"time"

	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

// ASTRepository AST仓储接口
type ASTRepository interface {
	// Save 保存AST到仓储
	Save(ctx context.Context, sourceFileID string, ast *entities.ASTNode) error

	// FindBySourceFileID 根据源文件ID查找AST
	FindBySourceFileID(ctx context.Context, sourceFileID string) (*entities.ASTNode, error)

	// Delete 删除AST
	Delete(ctx context.Context, sourceFileID string) error

	// Exists 检查AST是否存在
	Exists(ctx context.Context, sourceFileID string) (bool, error)

	// Update 更新AST
	Update(ctx context.Context, sourceFileID string, ast *entities.ASTNode) error

	// ListBySourceFileIDs 批量获取AST
	ListBySourceFileIDs(ctx context.Context, sourceFileIDs []string) (map[string]*entities.ASTNode, error)

	// GetASTStatistics 获取AST统计信息
	GetASTStatistics(ctx context.Context, sourceFileID string) (*ASTStatistics, error)
}

// ASTStatistics AST统计信息
type ASTStatistics struct {
	SourceFileID  string
	NodeCount     int
	MaxDepth      int
	FunctionCount int
	VariableCount int
	LiteralCount  int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewASTStatistics 创建AST统计信息
func NewASTStatistics(sourceFileID string) *ASTStatistics {
	now := time.Now()
	return &ASTStatistics{
		SourceFileID: sourceFileID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// UpdateFromAST 从AST更新统计信息
func (stats *ASTStatistics) UpdateFromAST(ast *entities.ASTNode) {
	if ast == nil {
		return
	}

	stats.NodeCount = countASTNodes(ast)
	stats.MaxDepth = calculateMaxDepth(ast)
	stats.FunctionCount = countFunctions(ast)
	stats.VariableCount = countVariables(ast)
	stats.LiteralCount = countLiterals(ast)
	stats.UpdatedAt = time.Now()
}

// countASTNodes 统计AST节点数量
func countASTNodes(ast *entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	count := 1 // 当前节点

	// 递归统计子节点
	for _, child := range (*ast).Children() {
		count += countASTNodes(&child)
	}

	return count
}

// calculateMaxDepth 计算AST最大深度
func calculateMaxDepth(ast *entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	maxChildDepth := 0
	for _, child := range (*ast).Children() {
		childDepth := calculateMaxDepth(&child)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}

	return 1 + maxChildDepth
}

// countFunctions 统计函数数量
func countFunctions(ast *entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	count := 0

	// 检查当前节点是否为函数
	if (*ast).NodeType() == entities.ASTNodeTypeFunction {
		count++
	}

	// 递归统计子节点
	for _, child := range (*ast).Children() {
		count += countFunctions(&child)
	}

	return count
}

// countVariables 统计变量数量
func countVariables(ast *entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	count := 0

	// 检查当前节点是否为变量
	if (*ast).NodeType() == entities.ASTNodeTypeVariable {
		count++
	}

	// 递归统计子节点
	for _, child := range (*ast).Children() {
		count += countVariables(&child)
	}

	return count
}

// countLiterals 统计字面量数量
func countLiterals(ast *entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	count := 0

	// 检查当前节点是否为字面量
	if (*ast).NodeType() == entities.ASTNodeTypeLiteral {
		count++
	}

	// 递归统计子节点
	for _, child := range (*ast).Children() {
		count += countLiterals(&child)
	}

	return count
}
