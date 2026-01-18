package services

import (
	"context"

	"echo/internal/modules/frontend/domain/entities"
)

// Monomorphization 单态化服务
// 职责：将泛型函数和类型转换为具体类型的版本
type Monomorphization struct {
	// 记录已单态化的函数，避免重复处理
	monomorphized map[string]*MonomorphizedFunction
}

// MonomorphizedFunction 单态化函数
type MonomorphizedFunction struct {
	OriginalName string
	TypeParams   []string // 类型参数，如 ["T", "U"]
	ConcreteTypes []string // 具体类型，如 ["int", "string"]
	MonomorphedName string // 单态化后的名称，如 "add_int_string"
	Function       *entities.FuncDef
}

// NewMonomorphization 创建单态化服务
func NewMonomorphization() *Monomorphization {
	return &Monomorphization{
		monomorphized: make(map[string]*MonomorphizedFunction),
	}
}

// MonomorphizeFunction 单态化泛型函数
func (m *Monomorphization) MonomorphizeFunction(
	ctx context.Context,
	funcDef *entities.FuncDef,
	concreteTypes map[string]string,
) (*MonomorphizedFunction, error) {

	// 生成单态化后的函数名
	monoName := m.generateMonomorphizedName(funcDef.Name, concreteTypes)

	// 检查是否已存在
	if existing, exists := m.monomorphized[monoName]; exists {
		return existing, nil
	}

	// 复制函数定义
	monomorphizedFunc := &entities.FuncDef{
		Name:       monoName,
		TypeParams: funcDef.TypeParams, // 保持类型参数（如果有）
		Params:     make([]entities.Param, len(funcDef.Params)),
		ReturnType: funcDef.ReturnType,
		Body:       make([]entities.ASTNode, len(funcDef.Body)),
	}

	// 替换类型参数为具体类型
	for i, param := range funcDef.Params {
		substitutedType := m.substituteType(param.Type, concreteTypes)
		// 类型断言为string（简化处理）
		typeStr, ok := substitutedType.(string)
		if !ok {
			typeStr = param.Type // 如果转换失败，使用原始类型
		}
		monomorphizedFunc.Params[i] = entities.Param{
			Name: param.Name,
			Type: typeStr,
		}
	}

	// 复制函数体（这里简化处理，实际需要递归替换所有类型引用）
	copy(monomorphizedFunc.Body, funcDef.Body)

	// 创建单态化结果
	result := &MonomorphizedFunction{
		OriginalName:    funcDef.Name,
		TypeParams:      m.extractTypeParams(funcDef),
		ConcreteTypes:   m.mapToSlice(concreteTypes),
		MonomorphedName: monoName,
		Function:        monomorphizedFunc,
	}

	// 缓存结果
	m.monomorphized[monoName] = result

	return result, nil
}

// generateMonomorphizedName 生成单态化函数名
func (m *Monomorphization) generateMonomorphizedName(originalName string, concreteTypes map[string]string) string {
	name := originalName + "_"
	for _, concreteType := range concreteTypes {
		name += concreteType + "_"
	}
	// 移除末尾的下划线
	if len(name) > 0 && name[len(name)-1] == '_' {
		name = name[:len(name)-1]
	}
	return name
}

// substituteType 替换类型参数为具体类型
func (m *Monomorphization) substituteType(typeExpr interface{}, concreteTypes map[string]string) interface{} {
	// 这里简化处理，实际需要递归处理复杂类型
	if typeName, ok := typeExpr.(string); ok {
		if concrete, exists := concreteTypes[typeName]; exists {
			return concrete
		}
	}
	return typeExpr
}

// extractTypeParams 提取类型参数（这里简化实现）
func (m *Monomorphization) extractTypeParams(funcDef *entities.FuncDef) []string {
	// 实际实现需要从函数定义中解析泛型参数
	// 这里返回空切片作为占位符
	return []string{}
}

// mapToSlice 将map转换为有序切片
func (m *Monomorphization) mapToSlice(mymap map[string]string) []string {
	result := make([]string, 0, len(mymap))
	for _, value := range mymap {
		result = append(result, value)
	}
	return result
}

// GetMonomorphizedFunctions 获取所有单态化函数
func (m *Monomorphization) GetMonomorphizedFunctions() []*MonomorphizedFunction {
	result := make([]*MonomorphizedFunction, 0, len(m.monomorphized))
	for _, fn := range m.monomorphized {
		result = append(result, fn)
	}
	return result
}

// MonomorphizeProgram 对整个程序进行单态化处理
func (m *Monomorphization) MonomorphizeProgram(program *entities.Program) (*entities.Program, error) {
	// 创建程序副本
	result := &entities.Program{
		Statements: make([]entities.ASTNode, 0, len(program.Statements)),
	}

	// 遍历所有语句，查找泛型函数调用并进行单态化
	for _, stmt := range program.Statements {
		processedStmt, err := m.processStatement(stmt)
		if err != nil {
			return nil, err
		}
		result.Statements = append(result.Statements, processedStmt)
	}

	// 添加所有单态化的函数定义
	for _, monoFunc := range m.GetMonomorphizedFunctions() {
		result.Statements = append(result.Statements, monoFunc.Function)
	}

	return result, nil
}

// processStatement 处理单个语句
func (m *Monomorphization) processStatement(stmt entities.ASTNode) (entities.ASTNode, error) {
	// 这里简化实现，实际需要处理函数调用语句
	// 检查是否是泛型函数调用，如果是则进行单态化
	return stmt, nil
}