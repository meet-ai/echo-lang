package generation

import (
	"echo/internal/modules/frontend/domain/entities"

	"github.com/llir/llvm/ir"
)

// StatementGenerationResult 语句生成结果
type StatementGenerationResult struct {
	Success bool
	Error   error
	// 移除了GeneratedIR字段，因为现在通过IRModuleManager统一管理
}

// StatementGenerator 语句生成领域服务接口
// 职责：负责所有类型语句的代码生成逻辑
// 遵循DDD原则：单一职责、依赖倒置、接口隔离
type StatementGenerator interface {
	// 生成程序语句 - 聚合根方法
	GenerateProgram(irManager IRModuleManager, statements []entities.ASTNode) (*StatementGenerationResult, error)

	// 生成单个语句 - 核心业务方法
	GenerateStatement(irManager IRModuleManager, stmt entities.ASTNode) (*StatementGenerationResult, error)

	// 生成打印语句 - 具体语句类型处理
	GeneratePrintStatement(irManager IRModuleManager, stmt *entities.PrintStmt) (*StatementGenerationResult, error)

	// 生成变量声明 - 状态管理相关
	GenerateVarDeclaration(irManager IRModuleManager, stmt *entities.VarDecl) (*StatementGenerationResult, error)

	// 生成赋值语句 - 状态变更相关
	GenerateAssignmentStatement(irManager IRModuleManager, stmt *entities.AssignStmt) (*StatementGenerationResult, error)

	// 生成表达式语句 - 表达式包装
	GenerateExpressionStatement(irManager IRModuleManager, stmt *entities.ExprStmt) (*StatementGenerationResult, error)

	// 生成函数定义 - 函数级构造
	GenerateFuncDefinition(irManager IRModuleManager, stmt *entities.FuncDef) (*StatementGenerationResult, error)

	// 生成返回语句 - 控制流相关
	GenerateReturnStatement(irManager IRModuleManager, stmt *entities.ReturnStmt) (*StatementGenerationResult, error)

	// 生成模式匹配语句 - 高级控制流
	GenerateMatchStatement(irManager IRModuleManager, stmt *entities.MatchStmt) (*StatementGenerationResult, error)

	// 生成if语句 - 条件分支
	GenerateIfStatement(irManager IRModuleManager, stmt *entities.IfStmt) (*StatementGenerationResult, error)

	// 生成for循环 - 迭代控制
	GenerateForStatement(irManager IRModuleManager, stmt *entities.ForStmt) (*StatementGenerationResult, error)

	// 生成while循环 - 条件循环
	GenerateWhileStatement(irManager IRModuleManager, stmt *entities.WhileStmt) (*StatementGenerationResult, error)

	// 生成select语句 - 并发选择
	GenerateSelectStatement(irManager IRModuleManager, stmt *entities.SelectStmt) (*StatementGenerationResult, error)

	// 生成trait定义 - 接口契约
	GenerateTraitDefinition(irManager IRModuleManager, stmt *entities.TraitDef) (*StatementGenerationResult, error)

	// 生成枚举定义 - 整型常量
	GenerateEnumDefinition(irManager IRModuleManager, stmt *entities.EnumDef) (*StatementGenerationResult, error)

	// 生成方法定义 - 带接收者的函数
	GenerateMethodDefinition(irManager IRModuleManager, stmt *entities.MethodDef) (*StatementGenerationResult, error)

	// 编译单态化函数（延迟编译）
	CompileMonomorphizedFunction(irManager IRModuleManager, funcDef *entities.FuncDef, typeParamMap map[string]string, monoName string) error

	// 验证语句生成上下文
	ValidateGenerationContext() error

	// 获取生成统计信息
	GetGenerationStats() *StatementGenerationStats
}

// StatementGenerationStats 生成统计信息
type StatementGenerationStats struct {
	TotalStatements       int
	SuccessfulGenerations int
	FailedGenerations     int
	GenerationTime        int64 // 纳秒
}

// IRModuleManager LLVM IR模块管理领域服务接口
// 职责：统一管理LLVM IR模块的构建，确保结构正确性
type IRModuleManager interface {
	// 全局变量管理
	AddGlobalVariable(name string, value interface{}) error
	// CreateGlobalVariable 创建模块级全局变量，返回全局指针供符号表注册；任意函数内可 Load/Store。
	CreateGlobalVariable(typ interface{}, name string) (interface{}, error)
	AddStringConstant(content string) (*ir.Global, error)

	// 外部函数管理
	GetExternalFunction(name string) (interface{}, bool)
	GetFunction(name string) (interface{}, bool)
	RegisterFunction(name string, fn interface{}) error
	RegisterExternalFunction(name string, fn interface{}) error

	// 函数管理
	CreateFunction(name string, returnType interface{}, paramTypes []interface{}) (interface{}, error)
	GetCurrentFunction() interface{}
	GetCurrentFunctionName() string // 当前函数名，用于判断是否在 main 中（顶层变量用全局）
	SetCurrentFunction(fn interface{}) error
	// 类型参数管理（用于泛型函数中的trait方法调用）
	SetCurrentFunctionTypeParams(typeParams map[string]string)
	GetCurrentFunctionTypeParams() map[string]string
	// 函数定义的类型参数列表管理
	SetFunctionTypeParamNames(funcName string, typeParams []string)
	GetFunctionTypeParamNames(funcName string) []string

	// 泛型函数定义管理（用于延迟编译）
	SaveGenericFunctionDefinition(funcName string, funcDef *entities.FuncDef)
	GetGenericFunctionDefinition(funcName string) (*entities.FuncDef, bool)
	// 单态化函数编译状态管理
	IsMonomorphizedFunctionCompiled(monoName string) bool
	MarkMonomorphizedFunctionCompiled(monoName string)

	// 基本块管理
	CreateBasicBlock(name string) (interface{}, error)
	GetBlockByName(name string) (interface{}, error)
	GetCurrentBasicBlock() interface{}
	SetCurrentBasicBlock(block interface{}) error

	// 指令生成
	CreateAlloca(typ interface{}, name string) (interface{}, error)
	CreateStore(value interface{}, ptr interface{}) error
	CreateLoad(typ interface{}, ptr interface{}, name string) (interface{}, error)
	CreateRet(value interface{}) error
	CreateCall(fn interface{}, args ...interface{}) (interface{}, error)
	CreateBinaryOp(op string, left interface{}, right interface{}, name string) (interface{}, error)
	CreateBr(dest interface{}) error
	CreateCondBr(cond interface{}, trueDest interface{}, falseDest interface{}) error
	// CreatePointerIsNull 生成 (ptr == null) 的 i1 条件，用于 NULL 检查分支
	CreatePointerIsNull(ptr interface{}) (condition interface{}, err error)
	// CreateUnreachable 在当前块插入 unreachable 终止符（用于错误路径）
	CreateUnreachable() error

	// CreateGetElementPtr 创建GetElementPtr指令
	CreateGetElementPtr(elemType interface{}, ptr interface{}, indices ...interface{}) (interface{}, error)

	// CreateBitCast 创建bitcast指令
	CreateBitCast(value interface{}, targetType interface{}, name string) (interface{}, error)

	// 获取最终的IR字符串
	GetIRString() string

	// 验证IR模块完整性
	Validate() error

	// 初始化主函数
	InitializeMainFunction() error

	// 完成IR构建
	Finalize() error
}
