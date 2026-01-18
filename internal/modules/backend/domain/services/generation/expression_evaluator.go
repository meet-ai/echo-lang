package generation

import (
	"echo/internal/modules/frontend/domain/entities"
)

// ExpressionEvaluator 表达式求值领域服务接口
// 职责：负责所有类型表达式的求值和代码生成
type ExpressionEvaluator interface {
	// 求值表达式，通过IRModuleManager生成指令，返回生成的IR值
	Evaluate(irManager IRModuleManager, expr entities.Expr) (interface{}, error)

	// 求值整数字面量
	EvaluateIntLiteral(irManager IRModuleManager, literal *entities.IntLiteral) (interface{}, error)

	// 求值字符串字面量
	EvaluateStringLiteral(irManager IRModuleManager, literal *entities.StringLiteral) (interface{}, error)

	// 求值标识符（变量引用）
	EvaluateIdentifier(irManager IRModuleManager, identifier *entities.Identifier) (interface{}, error)

	// 求值二元表达式
	EvaluateBinaryExpr(irManager IRModuleManager, expr *entities.BinaryExpr) (interface{}, error)

	// 求值函数调用
	EvaluateFuncCall(irManager IRModuleManager, call *entities.FuncCall) (interface{}, error)

	// 求值数组字面量
	EvaluateArrayLiteral(irManager IRModuleManager, literal *entities.ArrayLiteral) (interface{}, error)

	// 求值模式匹配表达式
	EvaluateMatchExpr(irManager IRModuleManager, expr *entities.MatchExpr) (interface{}, error)
}
