// Package ast_to_entities 将 value_objects.ProgramAST 转换为 entities.Program，供 backend 代码生成使用
package ast_to_entities

import (
	"fmt"

	"echo/internal/modules/frontend/domain/entities"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// ConvertProgram 将 ProgramAST 转为 entities.Program
func ConvertProgram(programAST *sharedVO.ProgramAST) (*entities.Program, error) {
	if programAST == nil {
		return nil, fmt.Errorf("programAST is nil")
	}
	nodes := programAST.Nodes()
	statements := make([]entities.ASTNode, 0, len(nodes))
	for _, node := range nodes {
		stmt, err := convertStatement(node)
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			statements = append(statements, stmt)
		}
	}
	return &entities.Program{Statements: statements}, nil
}

func convertStatement(node sharedVO.ASTNode) (entities.ASTNode, error) {
	if node == nil {
		return nil, nil
	}
	switch n := node.(type) {
	case *sharedVO.FunctionDeclaration:
		return convertFunctionDeclaration(n)
	case *sharedVO.AsyncFunctionDeclaration:
		return convertAsyncFunctionDeclaration(n)
	case *sharedVO.VariableDeclaration:
		return convertVariableDeclaration(n)
	case *sharedVO.BlockStatement:
		return convertBlockStatement(n)
	case *sharedVO.ReturnStatement:
		return convertReturnStatement(n)
	case *sharedVO.ExpressionStatement:
		return convertExpressionStatement(n)
	case *sharedVO.StructDeclaration:
		return convertStructDeclaration(n)
	case *sharedVO.EnumDeclaration:
		return convertEnumDeclaration(n)
	case *sharedVO.TraitDeclaration:
		return convertTraitDeclaration(n)
	case *sharedVO.ImplDeclaration:
		return convertImplDeclaration(n)
	default:
		return nil, nil
	}
}

func typeAnnotationToString(ta *sharedVO.TypeAnnotation) string {
	if ta == nil {
		return ""
	}
	// 数组类型 [] 单参表示为 [T]
	if ta.TypeName() == "[]" && len(ta.GenericArgs()) == 1 {
		return "[" + typeAnnotationToString(ta.GenericArgs()[0]) + "]"
	}
	s := ta.TypeName()
	if ta.IsPointer() {
		s = "*" + s
	}
	args := ta.GenericArgs()
	if len(args) > 0 {
		s += "<"
		for i, a := range args {
			if i > 0 {
				s += ", "
			}
			s += typeAnnotationToString(a)
		}
		s += ">"
	}
	return s
}

func convertFunctionDeclaration(fd *sharedVO.FunctionDeclaration) (*entities.FuncDef, error) {
	body := make([]entities.ASTNode, 0)
	if fd.Body() != nil {
		for _, s := range fd.Body().Statements() {
			stmt, err := convertStatement(s)
			if err != nil {
				return nil, err
			}
			if stmt != nil {
				body = append(body, stmt)
			}
		}
	}
	params := make([]entities.Param, 0)
	for _, p := range fd.Parameters() {
		params = append(params, entities.Param{
			Name: p.Name(),
			Type: typeAnnotationToString(p.TypeAnnotation()),
		})
	}
	returnType := ""
	if fd.ReturnType() != nil {
		returnType = typeAnnotationToString(fd.ReturnType())
	}
	typeParams := make([]entities.GenericParam, 0)
	for _, gp := range fd.GenericParams() {
		typeParams = append(typeParams, entities.GenericParam{Name: gp.Name()})
	}
	return &entities.FuncDef{
		Name:       fd.Name(),
		TypeParams: typeParams,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
		IsAsync:    false,
	}, nil
}

func convertAsyncFunctionDeclaration(afd *sharedVO.AsyncFunctionDeclaration) (*entities.AsyncFuncDef, error) {
	fd := afd.FunctionDeclaration()
	if fd == nil {
		return nil, fmt.Errorf("AsyncFunctionDeclaration has nil FunctionDeclaration")
	}
	params := make([]entities.Param, 0)
	for _, p := range fd.Parameters() {
		params = append(params, entities.Param{
			Name: p.Name(),
			Type: typeAnnotationToString(p.TypeAnnotation()),
		})
	}
	ret := ""
	if fd.ReturnType() != nil {
		ret = typeAnnotationToString(fd.ReturnType())
	}
	body := make([]entities.ASTNode, 0)
	if fd.Body() != nil {
		for _, s := range fd.Body().Statements() {
			stmt, err := convertStatement(s)
			if err != nil {
				return nil, err
			}
			if stmt != nil {
				body = append(body, stmt)
			}
		}
	}
	return &entities.AsyncFuncDef{
		Name:       fd.Name(),
		TypeParams: nil,
		Params:     params,
		ReturnType: ret,
		Body:       body,
	}, nil
}

func convertVariableDeclaration(vd *sharedVO.VariableDeclaration) (*entities.VarDecl, error) {
	var value entities.Expr
	if vd.Initializer() != nil {
		expr, err := convertExpr(vd.Initializer())
		if err != nil {
			return nil, err
		}
		value = expr
	}
	typ := ""
	if vd.VarType() != nil {
		typ = typeAnnotationToString(vd.VarType())
	}
	return &entities.VarDecl{
		Name:     vd.Name(),
		Type:     typ,
		Value:    value,
		Inferred: typ == "",
	}, nil
}

func convertBlockStatement(bs *sharedVO.BlockStatement) (*entities.BlockStmt, error) {
	stmts := make([]entities.ASTNode, 0)
	for _, s := range bs.Statements() {
		stmt, err := convertStatement(s)
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	return &entities.BlockStmt{Statements: stmts}, nil
}

func convertReturnStatement(rs *sharedVO.ReturnStatement) (*entities.ReturnStmt, error) {
	var value entities.Expr
	if rs.Expression() != nil {
		expr, err := convertExpr(rs.Expression())
		if err != nil {
			return nil, err
		}
		if e, ok := expr.(entities.Expr); ok {
			value = e
		}
	}
	return &entities.ReturnStmt{Value: value}, nil
}

func convertExpressionStatement(es *sharedVO.ExpressionStatement) (*entities.ExprStmt, error) {
	if es.Expression() == nil {
		return nil, nil
	}
	expr, err := convertExpr(es.Expression())
	if err != nil {
		return nil, err
	}
	if expr == nil {
		return nil, nil
	}
	return &entities.ExprStmt{Expression: expr}, nil
}

func convertStructDeclaration(sd *sharedVO.StructDeclaration) (*entities.StructDef, error) {
	fields := make([]entities.StructField, 0)
	for _, f := range sd.Fields() {
		typ := ""
		if f.FieldType() != nil {
			typ = typeAnnotationToString(f.FieldType())
		}
		fields = append(fields, entities.StructField{Name: f.Name(), Type: typ})
	}
	return &entities.StructDef{
		Name:   sd.Name(),
		Fields: fields,
	}, nil
}

func convertEnumDeclaration(ed *sharedVO.EnumDeclaration) (*entities.EnumDef, error) {
	variants := make([]entities.EnumVariant, 0)
	for _, v := range ed.Variants() {
		variants = append(variants, entities.EnumVariant{Name: v.Name()})
	}
	return &entities.EnumDef{
		Name:       ed.Name(),
		ImplTraits: nil,
		Variants:   variants,
	}, nil
}

func convertTraitDeclaration(td *sharedVO.TraitDeclaration) (*entities.TraitDef, error) {
	methods := make([]entities.TraitMethod, 0)
	for _, m := range td.Methods() {
		params := make([]entities.Param, 0)
		for _, p := range m.Parameters() {
			params = append(params, entities.Param{
				Name: p.Name(),
				Type: typeAnnotationToString(p.TypeAnnotation()),
			})
		}
		ret := ""
		if m.ReturnType() != nil {
			ret = typeAnnotationToString(m.ReturnType())
		}
		methods = append(methods, entities.TraitMethod{
			Name:       m.Name(),
			Params:     params,
			ReturnType: ret,
		})
	}
	typeParams := make([]entities.GenericParam, 0)
	for _, gp := range td.GenericParams() {
		typeParams = append(typeParams, entities.GenericParam{Name: gp.Name()})
	}
	return &entities.TraitDef{
		Name:       td.Name(),
		TypeParams: typeParams,
		Methods:    methods,
	}, nil
}

func convertImplDeclaration(id *sharedVO.ImplDeclaration) (*entities.ImplDef, error) {
	traitName := ""
	if id.TraitType() != nil {
		traitName = typeAnnotationToString(id.TraitType())
	}
	targetType := ""
	if id.TargetType() != nil {
		targetType = typeAnnotationToString(id.TargetType())
	}
	methods := make([]entities.MethodDef, 0)
	for _, m := range id.Methods() {
		allParams := m.Parameters()
		receiverVar := ""
		params := make([]entities.Param, 0)
		for i, p := range allParams {
			param := entities.Param{Name: p.Name(), Type: typeAnnotationToString(p.TypeAnnotation())}
			if i == 0 {
				receiverVar = p.Name()
			} else {
				params = append(params, param)
			}
		}
		ret := ""
		if m.ReturnType() != nil {
			ret = typeAnnotationToString(m.ReturnType())
		}
		body := make([]entities.ASTNode, 0)
		if m.Body() != nil {
			for _, s := range m.Body().Statements() {
				stmt, err := convertStatement(s)
				if err != nil {
					return nil, err
				}
				if stmt != nil {
					body = append(body, stmt)
				}
			}
		}
		methods = append(methods, entities.MethodDef{
			Receiver:    targetType,
			ReceiverVar: receiverVar,
			Name:        m.Name(),
			Params:      params,
			ReturnType:  ret,
			Body:        body,
		})
	}
	return &entities.ImplDef{
		TraitName: traitName,
		Methods:   methods,
	}, nil
}

func convertExpr(node sharedVO.ASTNode) (entities.Expr, error) {
	if node == nil {
		return nil, nil
	}
	switch n := node.(type) {
	case *sharedVO.Identifier:
		return &entities.Identifier{Name: n.Name()}, nil
	case *sharedVO.IntegerLiteral:
		return &entities.IntLiteral{Value: int(n.Value())}, nil
	case *sharedVO.FloatLiteral:
		return &entities.FloatLiteral{Value: n.Value()}, nil
	case *sharedVO.StringLiteral:
		return &entities.StringLiteral{Value: n.Value()}, nil
	case *sharedVO.BooleanLiteral:
		return &entities.BoolLiteral{Value: n.Value()}, nil
	case *sharedVO.BinaryExpression:
		left, err := convertExpr(n.Left())
		if err != nil {
			return nil, err
		}
		right, err := convertExpr(n.Right())
		if err != nil {
			return nil, err
		}
		if left == nil || right == nil {
			return nil, fmt.Errorf("binary expression missing operand")
		}
		return &entities.BinaryExpr{Left: left, Op: n.Operator(), Right: right}, nil
	case *sharedVO.UnaryExpression:
		operand, err := convertExpr(n.Operand())
		if err != nil {
			return nil, err
		}
		if operand == nil {
			return nil, fmt.Errorf("unary expression missing operand")
		}
		op := n.Operator()
		switch op {
		case "-":
			return &entities.BinaryExpr{Left: &entities.IntLiteral{Value: 0}, Op: "-", Right: operand}, nil
		case "+":
			return operand, nil
		case "spawn":
			return &entities.SpawnExpr{Function: operand}, nil
		case "await":
			return &entities.AwaitExpr{Expression: operand}, nil
		case "<-":
			return &entities.ReceiveExpr{Channel: operand}, nil
		default:
			return nil, fmt.Errorf("unary op %q not supported in converter", op)
		}
	case *sharedVO.FunctionCallExpression:
		callee := n.Callee()
		name := ""
		if ident, ok := callee.(*sharedVO.Identifier); ok {
			name = ident.Name()
		} else {
			return nil, fmt.Errorf("function call callee must be identifier")
		}
		args := make([]entities.Expr, 0, len(n.Args()))
		for _, a := range n.Args() {
			ea, err := convertExpr(a)
			if err != nil {
				return nil, err
			}
			args = append(args, ea)
		}
		return &entities.FuncCall{Name: name, Args: args}, nil
	default:
		return nil, nil
	}
}
