package repositories

import (
	"context"
	"time"

	"echo/internal/modules/frontend/domain/entities"
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
func (stats *ASTStatistics) UpdateFromAST(ast entities.ASTNode) {
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
func countASTNodes(ast entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	count := 1 // 当前节点

	// 根据不同节点类型统计子节点
	switch node := ast.(type) {
	case *entities.FuncDef:
		count += len(node.Body)
		for _, stmt := range node.Body {
			count += countASTNodes(stmt)
		}
	case *entities.IfStmt:
		// node.Condition 是 Expr 接口，不是 ASTNode，暂时跳过
		for _, stmt := range node.ThenBody {
			if stmt != nil {
				count += countASTNodes(stmt)
			}
		}
		for _, stmt := range node.ElseBody {
			if stmt != nil {
				count += countASTNodes(stmt)
			}
		}
	case *entities.ForStmt:
		// 跳过 Expr 类型的字段，专注于 ASTNode 类型的子节点
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countASTNodes(stmt)
			}
		}
	case *entities.WhileStmt:
		// 跳过 Expr 类型的条件，专注于语句体
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countASTNodes(stmt)
			}
		}
	case *entities.StructDef:
		// StructDef 的字段类型信息不作为 ASTNode 计算
		// 这里可以根据需要添加其他逻辑
	case *entities.VarDecl:
		// VarDecl 的值是 Expr，不是 ASTNode，暂时跳过
	case *entities.AssignStmt:
		// 暂时跳过赋值语句中的表达式
	case *entities.ReturnStmt:
		// 暂时跳过返回语句中的表达式
	case *entities.ExprStmt:
		// 暂时跳过表达式语句
	case *entities.PrintStmt:
		// 暂时跳过打印语句中的表达式
	// 为其他节点类型添加统计逻辑...
	}

	return count
}

// calculateMaxDepth 计算AST最大深度
func calculateMaxDepth(ast entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	maxChildDepth := 0

	// 根据不同节点类型计算深度
	switch node := ast.(type) {
	case *entities.FuncDef:
		for _, stmt := range node.Body {
			childDepth := calculateMaxDepth(stmt)
			if childDepth > maxChildDepth {
				maxChildDepth = childDepth
			}
		}
	case *entities.IfStmt:
		// 跳过条件表达式，只计算语句体深度
		for _, stmt := range node.ThenBody {
			if stmt != nil {
				childDepth := calculateMaxDepth(stmt)
				if childDepth > maxChildDepth {
					maxChildDepth = childDepth
				}
			}
		}
		for _, stmt := range node.ElseBody {
			if stmt != nil {
				childDepth := calculateMaxDepth(stmt)
				if childDepth > maxChildDepth {
					maxChildDepth = childDepth
				}
			}
		}
	case *entities.ForStmt:
		// 只计算语句体深度
		for _, stmt := range node.Body {
			if stmt != nil {
				childDepth := calculateMaxDepth(stmt)
				if childDepth > maxChildDepth {
					maxChildDepth = childDepth
				}
			}
		}
	case *entities.WhileStmt:
		// 只计算语句体深度
		for _, stmt := range node.Body {
			if stmt != nil {
				childDepth := calculateMaxDepth(stmt)
				if childDepth > maxChildDepth {
					maxChildDepth = childDepth
				}
			}
		}
	// 为其他节点类型添加深度计算逻辑...
	}

	return 1 + maxChildDepth
}

// countFunctions 统计函数数量
func countFunctions(ast entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	count := 0

	// 检查当前节点是否为函数定义
	switch ast.(type) {
	case *entities.FuncDef, *entities.AsyncFuncDef:
		count++
	}

	// 递归统计子节点
	switch node := ast.(type) {
	case *entities.FuncDef:
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countFunctions(stmt)
			}
		}
	case *entities.IfStmt:
		for _, stmt := range node.ThenBody {
			if stmt != nil {
				count += countFunctions(stmt)
			}
		}
		for _, stmt := range node.ElseBody {
			if stmt != nil {
				count += countFunctions(stmt)
			}
		}
	case *entities.ForStmt:
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countFunctions(stmt)
			}
		}
	case *entities.WhileStmt:
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countFunctions(stmt)
			}
		}
	// 为其他包含语句体的节点类型添加递归逻辑...
	}

	return count
}

// countVariables 统计变量数量
func countVariables(ast entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	count := 0

	// 检查当前节点是否为变量声明
	switch ast.(type) {
	case *entities.VarDecl:
		count++
	}

	// 递归统计子节点中的变量引用
	switch node := ast.(type) {
	case *entities.FuncDef:
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countVariables(stmt)
			}
		}
	case *entities.IfStmt:
		// 跳过 Expr 类型的条件
		for _, stmt := range node.ThenBody {
			if stmt != nil {
				count += countVariables(stmt)
			}
		}
		for _, stmt := range node.ElseBody {
			if stmt != nil {
				count += countVariables(stmt)
			}
		}
	case *entities.ForStmt:
		// 跳过 Expr 类型的字段
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countVariables(stmt)
			}
		}
	case *entities.WhileStmt:
		// 跳过 Expr 类型的条件
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countVariables(stmt)
			}
		}
	case *entities.VarDecl:
		// 变量声明已在上方计数，这里不再重复
	case *entities.AssignStmt:
		// 跳过 Expr 类型的值
	case *entities.ReturnStmt:
		// 跳过 Expr 类型的值
	// 为其他节点类型添加变量统计逻辑...
	}

	return count
}

// countLiterals 统计字面量数量
func countLiterals(ast entities.ASTNode) int {
	if ast == nil {
		return 0
	}

	count := 0

	// 暂时简化字面量统计
	// TODO: 根据实际 AST 节点类型实现字面量统计

	// 递归统计子节点中的字面量
	switch node := ast.(type) {
	case *entities.FuncDef:
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countLiterals(stmt)
			}
		}
	case *entities.IfStmt:
		// 跳过 Expr 类型的条件
		for _, stmt := range node.ThenBody {
			if stmt != nil {
				count += countLiterals(stmt)
			}
		}
		for _, stmt := range node.ElseBody {
			if stmt != nil {
				count += countLiterals(stmt)
			}
		}
	case *entities.ForStmt:
		// 跳过 Expr 类型的字段
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countLiterals(stmt)
			}
		}
	case *entities.WhileStmt:
		// 跳过 Expr 类型的条件
		for _, stmt := range node.Body {
			if stmt != nil {
				count += countLiterals(stmt)
			}
		}
	case *entities.VarDecl:
		// 跳过 Expr 类型的值
	case *entities.AssignStmt:
		// 跳过 Expr 类型的值
	case *entities.ReturnStmt:
		// 跳过 Expr 类型的值
	case *entities.ArrayLiteral:
		// 跳过 Expr 类型的元素
	case *entities.StructLiteral:
		// 跳过 Expr 类型的值
	// 为其他节点类型添加字面量统计逻辑...
	}

	return count
}
