package generation

import (
	"echo/internal/modules/backend/domain/services/generation"
	"echo/internal/modules/backend/infrastructure/generation/impl"
)

// Container 代码生成领域服务容器
// 职责：组装和提供所有代码生成相关的领域服务实现
type Container struct {
	statementGenerator   generation.StatementGenerator
	expressionEvaluator  generation.ExpressionEvaluator
	controlFlowGenerator generation.ControlFlowGenerator
	symbolManager        generation.SymbolManager
	typeMapper           generation.TypeMapper
	irModuleManager      generation.IRModuleManager
}

// NewContainer 创建代码生成服务容器
func NewContainer() *Container {
	// 创建基础设施层实现
	typeMapperImpl := impl.NewTypeMapperImpl()
	symbolManagerImpl := impl.NewSymbolManagerImpl()
	irModuleManagerImpl := impl.NewIRModuleManagerImpl()
	expressionEvaluatorImpl := impl.NewExpressionEvaluatorImpl(symbolManagerImpl, typeMapperImpl, irModuleManagerImpl)

	// 先创建语句生成器（暂时传入nil的controlFlowGenerator）
	tempControlFlowGenerator := impl.NewControlFlowGeneratorImpl(nil, nil)
	statementGeneratorImpl := impl.NewStatementGeneratorImpl(
		expressionEvaluatorImpl,
		tempControlFlowGenerator, // 暂时传入nil
		symbolManagerImpl,
		typeMapperImpl,
		irModuleManagerImpl,
	)

	// 创建控制流生成器（传入真实的statementGenerator和expressionEvaluator）
	controlFlowGeneratorImpl := impl.NewControlFlowGeneratorImpl(statementGeneratorImpl, expressionEvaluatorImpl)

	// 更新语句生成器中的控制流生成器引用
	statementGeneratorImpl.SetControlFlowGenerator(controlFlowGeneratorImpl)

	return &Container{
		statementGenerator:   statementGeneratorImpl,
		expressionEvaluator:  expressionEvaluatorImpl,
		controlFlowGenerator: controlFlowGeneratorImpl,
		symbolManager:        symbolManagerImpl,
		typeMapper:           typeMapperImpl,
		irModuleManager:      irModuleManagerImpl,
	}
}

// StatementGenerator 获取语句生成器
func (c *Container) StatementGenerator() generation.StatementGenerator {
	return c.statementGenerator
}

// ExpressionEvaluator 获取表达式求值器
func (c *Container) ExpressionEvaluator() generation.ExpressionEvaluator {
	return c.expressionEvaluator
}

// ControlFlowGenerator 获取控制流生成器
func (c *Container) ControlFlowGenerator() generation.ControlFlowGenerator {
	return c.controlFlowGenerator
}

// SymbolManager 获取符号管理器
func (c *Container) SymbolManager() generation.SymbolManager {
	return c.symbolManager
}

// TypeMapper 获取类型映射器
func (c *Container) TypeMapper() generation.TypeMapper {
	return c.typeMapper
}

// IRModuleManager 获取IR模块管理器
func (c *Container) IRModuleManager() generation.IRModuleManager {
	return c.irModuleManager
}

// Validate 验证容器中的所有服务是否正确组装
func (c *Container) Validate() error {
	// 验证语句生成器的上下文
	if stmtGen, ok := c.statementGenerator.(*impl.StatementGeneratorImpl); ok {
		return stmtGen.ValidateGenerationContext()
	}
	return nil
}
