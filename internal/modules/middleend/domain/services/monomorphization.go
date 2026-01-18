package services

import (
	"fmt"
	"strings"

	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

// MonomorphizedFunction 单态化函数定义
type MonomorphizedFunction struct {
	OriginalName string            // 原始泛型函数名
	TypeArgs     []string          // 具体类型参数
	MonoName     string            // 单态化后的函数名
	FuncDef      *entities.FuncDef // 单态化后的函数定义
}

// Monomorphization 单态化引擎
type Monomorphization struct {
	monomorphizedFuncs map[string]*MonomorphizedFunction // key: "funcName[T1,T2]"
	typeInference      *TypeInference
}

// NewMonomorphization 创建单态化引擎
func NewMonomorphization() *Monomorphization {
	return &Monomorphization{
		monomorphizedFuncs: make(map[string]*MonomorphizedFunction),
		typeInference:      NewTypeInference(),
	}
}

// MonomorphizeProgram 对整个程序进行单态化处理
func (m *Monomorphization) MonomorphizeProgram(program *entities.Program) (*entities.Program, error) {
	// 第一遍：收集所有泛型函数调用
	genericCalls := m.collectGenericCalls(program)

	// 第二遍：为每个泛型调用生成单态化版本
	for _, call := range genericCalls {
		if err := m.monomorphizeCall(program, call); err != nil {
			return nil, fmt.Errorf("failed to monomorphize call to %s: %v", call.Name, err)
		}
	}

	// 第三遍：更新所有函数调用为单态化版本
	if err := m.updateCallsToMonomorphized(program); err != nil {
		return nil, err
	}

	return program, nil
}

// collectGenericCalls 收集程序中的所有泛型函数调用
func (m *Monomorphization) collectGenericCalls(program *entities.Program) []*entities.FuncCall {
	var calls []*entities.FuncCall

	for _, stmt := range program.Statements {
		m.collectCallsFromNode(stmt, &calls)
	}

	return calls
}

// collectCallsFromNode 从AST节点中收集函数调用
func (m *Monomorphization) collectCallsFromNode(node entities.ASTNode, calls *[]*entities.FuncCall) {
	switch n := node.(type) {
	case *entities.FuncCall:
		*calls = append(*calls, n)
		// 递归处理参数中的函数调用
		for _, arg := range n.Args {
			if argExpr, ok := arg.(entities.ASTNode); ok {
				m.collectCallsFromNode(argExpr, calls)
			}
		}
	case *entities.FuncDef:
		// 递归处理函数体
		for _, stmt := range n.Body {
			m.collectCallsFromNode(stmt, calls)
		}
	case *entities.IfStmt:
		for _, stmt := range n.ThenBody {
			m.collectCallsFromNode(stmt, calls)
		}
		if n.ElseBody != nil {
			for _, stmt := range n.ElseBody {
				m.collectCallsFromNode(stmt, calls)
			}
		}
	case *entities.ForStmt:
		for _, stmt := range n.Body {
			m.collectCallsFromNode(stmt, calls)
		}
	case *entities.WhileStmt:
		for _, stmt := range n.Body {
			m.collectCallsFromNode(stmt, calls)
		}
	case *entities.VarDecl:
		if n.Value != nil {
			if valExpr, ok := n.Value.(entities.ASTNode); ok {
				m.collectCallsFromNode(valExpr, calls)
			}
		}
	case *entities.ReturnStmt:
		if n.Value != nil {
			if valExpr, ok := n.Value.(entities.ASTNode); ok {
				m.collectCallsFromNode(valExpr, calls)
			}
		}
	}
}

// monomorphizeCall 为单个函数调用生成单态化版本
func (m *Monomorphization) monomorphizeCall(program *entities.Program, call *entities.FuncCall) error {
	// 查找对应的泛型函数定义
	funcDef := m.findGenericFuncDef(program, call.Name)
	if funcDef == nil || len(funcDef.TypeParams) == 0 {
		// 不是泛型函数，跳过
		return nil
	}

	// 推断类型参数
	inferredTypes, err := m.typeInference.InferTypes(funcDef, call)
	if err != nil {
		return fmt.Errorf("type inference failed: %v", err)
	}

	// 检查是否已经生成过单态化版本
	monoKey := m.makeMonomorphizationKey(call.Name, inferredTypes)
	if _, exists := m.monomorphizedFuncs[monoKey]; exists {
		return nil // 已经生成过
	}

	// 生成单态化版本
	monoFunc, err := m.createMonomorphizedFunction(funcDef, inferredTypes)
	if err != nil {
		return fmt.Errorf("failed to create monomorphized function: %v", err)
	}

	// 添加到程序中
	program.Statements = append(program.Statements, monoFunc.FuncDef)
	m.monomorphizedFuncs[monoKey] = monoFunc

	return nil
}

// findGenericFuncDef 查找泛型函数定义
func (m *Monomorphization) findGenericFuncDef(program *entities.Program, funcName string) *entities.FuncDef {
	for _, stmt := range program.Statements {
		if funcDef, ok := stmt.(*entities.FuncDef); ok {
			if funcDef.Name == funcName && len(funcDef.TypeParams) > 0 {
				return funcDef
			}
		}
	}
	return nil
}

// makeMonomorphizationKey 生成单态化函数的唯一键
func (m *Monomorphization) makeMonomorphizationKey(funcName string, typeArgs map[string]string) string {
	var parts []string
	parts = append(parts, funcName)

	// 按类型参数名称排序以确保一致性
	for _, typeParam := range []string{"T", "U", "V", "W"} { // 扩展支持更多类型参数
		if typ, exists := typeArgs[typeParam]; exists {
			parts = append(parts, typ)
		}
	}

	return funcName + "[" + strings.Join(parts[1:], ",") + "]"
}

// createMonomorphizedFunction 创建单态化函数
func (m *Monomorphization) createMonomorphizedFunction(originalFunc *entities.FuncDef, typeArgs map[string]string) (*MonomorphizedFunction, error) {
	// 生成单态化函数名
	monoName := m.makeMonomorphizationKey(originalFunc.Name, typeArgs)

	// 复制函数定义
	monoFunc := &entities.FuncDef{
		Name:       monoName,
		TypeParams: []entities.GenericParam{}, // 单态化后没有类型参数
		Params:     make([]entities.Param, len(originalFunc.Params)),
		ReturnType: originalFunc.ReturnType,
		Body:       make([]entities.ASTNode, len(originalFunc.Body)),
	}

	// 复制参数，替换类型变量
	for i, param := range originalFunc.Params {
		monoFunc.Params[i] = entities.Param{
			Name: param.Name,
			Type: m.substituteTypeVars(param.Type, typeArgs),
		}
	}

	// 替换返回类型
	monoFunc.ReturnType = m.substituteTypeVars(originalFunc.ReturnType, typeArgs)

	// 复制并替换函数体
	for i, stmt := range originalFunc.Body {
		monoFunc.Body[i] = m.substituteTypeVarsInNode(stmt, typeArgs)
	}

	return &MonomorphizedFunction{
		OriginalName: originalFunc.Name,
		TypeArgs:     m.typeMapToSlice(typeArgs),
		MonoName:     monoName,
		FuncDef:      monoFunc,
	}, nil
}

// substituteTypeVars 替换类型变量
func (m *Monomorphization) substituteTypeVars(typeStr string, typeArgs map[string]string) string {
	result := typeStr

	// 替换简单类型变量
	for varName, concreteType := range typeArgs {
		result = strings.ReplaceAll(result, varName, concreteType)
	}

	// 处理泛型类型，如 Container[T] -> Container[int]
	if strings.Contains(result, "[") && strings.Contains(result, "]") {
		// 这里简化处理，更复杂的泛型类型需要递归处理
		for varName, concreteType := range typeArgs {
			result = strings.ReplaceAll(result, "["+varName+"]", "["+concreteType+"]")
		}
	}

	return result
}

// substituteTypeVarsInNode 在AST节点中替换类型变量
func (m *Monomorphization) substituteTypeVarsInNode(node entities.ASTNode, typeArgs map[string]string) entities.ASTNode {
	switch n := node.(type) {
	case *entities.VarDecl:
		return &entities.VarDecl{
			Name:  n.Name,
			Type:  m.substituteTypeVars(n.Type, typeArgs),
			Value: n.Value, // 表达式中的类型变量替换需要更复杂的处理
		}
	case *entities.ReturnStmt:
		return &entities.ReturnStmt{
			Value: n.Value, // 类似地需要处理表达式
		}
	case *entities.FuncCall:
		// 更新函数调用为单态化版本
		monoKey := m.makeMonomorphizationKey(n.Name, typeArgs)
		if monoFunc, exists := m.monomorphizedFuncs[monoKey]; exists {
			return &entities.FuncCall{
				Name:     monoFunc.MonoName,
				TypeArgs: []string{}, // 单态化后没有类型参数
				Args:     n.Args,
			}
		}
		return n
	default:
		return node // 其他节点暂时保持不变
	}
}

// typeMapToSlice 将类型映射转换为有序切片
func (m *Monomorphization) typeMapToSlice(typeMap map[string]string) []string {
	var result []string
	// 按标准顺序返回：T, U, V, W...
	for _, key := range []string{"T", "U", "V", "W"} {
		if val, exists := typeMap[key]; exists {
			result = append(result, val)
		}
	}
	return result
}

// updateCallsToMonomorphized 更新所有函数调用为单态化版本
func (m *Monomorphization) updateCallsToMonomorphized(program *entities.Program) error {
	for _, stmt := range program.Statements {
		if err := m.updateCallsInNode(stmt); err != nil {
			return err
		}
	}
	return nil
}

// updateCallsInNode 更新节点中的函数调用
func (m *Monomorphization) updateCallsInNode(node entities.ASTNode) error {
	switch n := node.(type) {
	case *entities.FuncCall:
		// TODO: 实现完整的函数调用更新逻辑
		// 这里需要访问程序上下文来查找函数定义并进行类型推断

		// 递归处理参数
		for _, arg := range n.Args {
			if argNode, ok := arg.(entities.ASTNode); ok {
				if err := m.updateCallsInNode(argNode); err != nil {
					return err
				}
			}
		}

	case *entities.FuncDef:
		for _, stmt := range n.Body {
			if err := m.updateCallsInNode(stmt); err != nil {
				return err
			}
		}
		// 处理其他复合语句...
	}

	return nil
}

// GetMonomorphizedFunctions 获取所有单态化函数
func (m *Monomorphization) GetMonomorphizedFunctions() map[string]*MonomorphizedFunction {
	result := make(map[string]*MonomorphizedFunction)
	for k, v := range m.monomorphizedFuncs {
		result[k] = v
	}
	return result
}
