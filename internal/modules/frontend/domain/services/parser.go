package services

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"echo/internal/modules/frontend/domain/entities"
)

// Parser 接口已在 interfaces.go 中定义

// SimpleParser 简单语法分析器实现
type SimpleParser struct{}

// NewSimpleParser 创建新的简单解析器
func NewSimpleParser() Parser {
	return &SimpleParser{}
}

// Parse 解析源代码内容为AST
func (p *SimpleParser) Parse(content string) (*entities.Program, error) {
	program := &entities.Program{Statements: []entities.ASTNode{}}

	lines := strings.Split(content, "\n")
	i := 0
	inFunctionBody := false
	inIfBody := false
	inWhileBody := false
	inForBody := false
	inStructBody := false
	inMethodBody := false
	inEnumBody := false
	inImplBody := false
	inMatchBody := false
	inAsyncFunctionBody := false
	var currentFunction *entities.FuncDef
	var currentIfStmt *entities.IfStmt
	var currentWhileStmt *entities.WhileStmt
	var currentForStmt *entities.ForStmt
	var currentStructDef *entities.StructDef
	var currentMethodDef *entities.MethodDef
	var currentEnumDef *entities.EnumDef
	var currentMatchStmt *entities.MatchStmt
	var currentAsyncFunc *entities.AsyncFuncDef
	var pendingImplTraits []entities.ImplAnnotation              // 等待应用的impl注解
	var pendingAssociatedTypeImpls []entities.AssociatedTypeImpl // 等待应用的关联类型实现
	var functionBodyLines []string
	var ifBodyLines []string
	var whileBodyLines []string
	var forBodyLines []string
	var structBodyLines []string
	var methodBodyLines []string
	var enumBodyLines []string
	var implBodyLines []string
	var matchBodyLines []string
	var asyncFunctionBodyLines []string
	var parsingElse bool     // 标记是否正在解析else分支
	var thenBranchEnded bool // 标记then分支是否已结束

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "//") {
			i++
			continue
		}

		// 检查then分支结束后是否有else或else if
		if thenBranchEnded {
			if strings.TrimSpace(line) == "else" {
				// 进入else分支
				thenBranchEnded = false
				parsingElse = true
				inIfBody = true
				ifBodyLines = []string{}
				i++ // 跳过这一行，继续下一行
				continue
			} else if strings.Contains(line, "} else if ") {
				// 处理 } else if condition { 语法
				// 解析 else if condition 部分
				elseIfLine := strings.TrimSpace(line[1:]) // 移除开头的 }
				elseIfStmt, err := p.parseElseIfStmt(elseIfLine, i)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid else if statement: %v", i, err)
				}

				// 将 else if 作为嵌套的 if 语句放入当前if的 else 分支
				currentIfStmt.ElseBody = []entities.ASTNode{elseIfStmt}

				// 设置新的 currentIfStmt 为 else if 语句，继续处理其then分支
				currentIfStmt = elseIfStmt.(*entities.IfStmt)
				thenBranchEnded = false
				inIfBody = true
				ifBodyLines = []string{}
				i++ // 跳过这一行，继续下一行
				continue
			}
		}

		if inFunctionBody {
			// 在函数体内
			if line == "}" {
				// 函数体结束
				inFunctionBody = false

				// 从函数体行中移除最后的"}"（如果存在）
				bodyLines := functionBodyLines
				if len(bodyLines) > 0 && strings.TrimSpace(bodyLines[len(bodyLines)-1]) == "}" {
					bodyLines = bodyLines[:len(bodyLines)-1]
				}

				// 使用parseBlock解析函数体
				body, _, err := p.parseBlock(bodyLines, i-len(functionBodyLines)+1)
				if err != nil {
					return nil, fmt.Errorf("function %s body error: %v", currentFunction.Name, err)
				}
				currentFunction.Body = body

				// 添加函数定义到程序
				program.Statements = append(program.Statements, currentFunction)
				currentFunction = nil
				functionBodyLines = nil
			} else {
				// 收集函数体内的行
				functionBodyLines = append(functionBodyLines, line)
			}
		} else if inWhileBody {
			// 在while语句体内
			if line == "}" {
				// while语句体结束
				inWhileBody = false

				// 解析while循环体内的语句
				for j, bodyLine := range whileBodyLines {
					bodyLine = strings.TrimSpace(bodyLine)
					if bodyLine != "" {
						stmt, err := p.parseStatement(bodyLine, i-len(whileBodyLines)+j+1)
						if err != nil {
							return nil, fmt.Errorf("while statement body error: %v", err)
						}
						if stmt != nil {
							currentWhileStmt.Body = append(currentWhileStmt.Body, stmt)
						}
					}
				}

				// 添加完整的while语句到程序
				program.Statements = append(program.Statements, currentWhileStmt)
				currentWhileStmt = nil
				whileBodyLines = nil
			} else {
				// 收集while语句体内的行
				whileBodyLines = append(whileBodyLines, line)
			}
		} else if inForBody {
			// 在for语句体内
			if line == "}" {
				// for语句体结束
				inForBody = false

				// 解析for循环体内的语句
				for j, bodyLine := range forBodyLines {
					bodyLine = strings.TrimSpace(bodyLine)
					if bodyLine != "" {
						stmt, err := p.parseStatement(bodyLine, i-len(forBodyLines)+j+1)
						if err != nil {
							return nil, fmt.Errorf("for statement body error: %v", err)
						}
						if stmt != nil {
							currentForStmt.Body = append(currentForStmt.Body, stmt)
						}
					}
				}

				// 添加完整的for语句到程序
				program.Statements = append(program.Statements, currentForStmt)
				currentForStmt = nil
				forBodyLines = nil
			} else {
				// 收集for语句体内的行
				forBodyLines = append(forBodyLines, line)
			}
		} else if inStructBody {
			// 在结构体体内
			if line == "}" {
				// 结构体定义结束
				inStructBody = false

				// 解析结构体字段
				var fields []entities.StructField
				for _, fieldLine := range structBodyLines {
					fieldLine = strings.TrimSpace(fieldLine)
					fieldLine = strings.TrimSuffix(fieldLine, ",") // 移除可选的逗号
					if strings.Contains(fieldLine, ":") {
						parts := strings.Split(fieldLine, ":")
						if len(parts) == 2 {
							fieldName := strings.TrimSpace(parts[0])
							fieldType := strings.TrimSpace(parts[1])
							fields = append(fields, entities.StructField{
								Name: fieldName,
								Type: fieldType,
							})
						}
					}
				}

				currentStructDef.Fields = fields

				// 添加结构体定义到程序
				program.Statements = append(program.Statements, currentStructDef)
				currentStructDef = nil
				structBodyLines = nil
			} else {
				// 收集结构体字段行
				structBodyLines = append(structBodyLines, line)
			}
		} else if inMethodBody {
			// 在方法体内
			if line == "}" {
				// 方法定义结束
				inMethodBody = false

				// 解析方法体内的语句
				for j, bodyLine := range methodBodyLines {
					bodyLine = strings.TrimSpace(bodyLine)
					if bodyLine != "" {
						stmt, err := p.parseStatement(bodyLine, i-len(methodBodyLines)+j+1)
						if err != nil {
							return nil, fmt.Errorf("method %s body error: %v", currentMethodDef.Name, err)
						}
						if stmt != nil {
							currentMethodDef.Body = append(currentMethodDef.Body, stmt)
						}
					}
				}

				// 添加方法定义到程序
				program.Statements = append(program.Statements, currentMethodDef)
				currentMethodDef = nil
				methodBodyLines = nil
			} else {
				// 收集方法体内的行
				methodBodyLines = append(methodBodyLines, line)
			}
		} else if inMatchBody {
			// 在match体内
			if line == "}" {
				// match语句结束
				inMatchBody = false

				// 解析match的case分支
				matchCases, _, err := p.parseMatchCases(matchBodyLines, i-len(matchBodyLines)+1)
				if err != nil {
					return nil, fmt.Errorf("match cases error: %v", err)
				}
				currentMatchStmt.Cases = matchCases

				// 添加match语句到程序
				program.Statements = append(program.Statements, currentMatchStmt)
				currentMatchStmt = nil
				matchBodyLines = nil
			} else {
				// 收集match体内的行（case行）
				matchBodyLines = append(matchBodyLines, line)
			}
		} else if inAsyncFunctionBody {
			// 在async函数体内
			if line == "}" {
				// async函数定义结束
				inAsyncFunctionBody = false

				// 解析async函数体内的语句
				for j, bodyLine := range asyncFunctionBodyLines {
					bodyLine = strings.TrimSpace(bodyLine)
					if bodyLine != "" {
						stmt, err := p.parseStatement(bodyLine, i-len(asyncFunctionBodyLines)+j+1)
						if err != nil {
							return nil, fmt.Errorf("async function %s body error: %v", currentAsyncFunc.Name, err)
						}
						if stmt != nil {
							currentAsyncFunc.Body = append(currentAsyncFunc.Body, stmt)
						}
					}
				}

				// 添加async函数定义到程序
				program.Statements = append(program.Statements, currentAsyncFunc)
				currentAsyncFunc = nil
				asyncFunctionBodyLines = nil
			} else {
				// 收集async函数体内的行
				asyncFunctionBodyLines = append(asyncFunctionBodyLines, line)
			}
		} else if inEnumBody {
			// 在枚举体内
			if line == "}" {
				// 枚举定义结束
				inEnumBody = false

				// 解析枚举值
				var variants []entities.EnumVariant
				for _, enumLine := range enumBodyLines {
					enumLine = strings.TrimSpace(enumLine)
					enumLine = strings.TrimSuffix(enumLine, ",") // 移除可选的逗号
					if enumLine != "" {
						variants = append(variants, entities.EnumVariant{Name: enumLine})
					}
				}

				currentEnumDef.Variants = variants

				// 添加枚举定义到程序
				program.Statements = append(program.Statements, currentEnumDef)
				currentEnumDef = nil
				enumBodyLines = nil
			} else {
				// 收集枚举值行
				enumBodyLines = append(enumBodyLines, line)
			}
		} else if inImplBody {
			// 在impl体内
			if line == "}" {
				// impl块结束
				inImplBody = false

				// 暂时不支持方法实现，未来扩展
				// 这里可以解析方法实现

				// 添加impl块到程序
				implBodyLines = nil
			} else {
				// 收集impl体内的行
				implBodyLines = append(implBodyLines, line)
			}
		} else if inIfBody {
			// 在if语句体内
			if line == "}" {
				// if语句体结束
				if parsingElse {
					// else分支结束
					inIfBody = false
					parsingElse = false

					// 解析else分支内的语句
					for j, bodyLine := range ifBodyLines {
						bodyLine = strings.TrimSpace(bodyLine)
						if bodyLine != "" {
							stmt, err := p.parseStatement(bodyLine, i-len(ifBodyLines)+j+1)
							if err != nil {
								return nil, fmt.Errorf("if statement else body error: %v", err)
							}
							if stmt != nil {
								currentIfStmt.ElseBody = append(currentIfStmt.ElseBody, stmt)
							}
						}
					}

					// 添加完整的if语句到程序
					program.Statements = append(program.Statements, currentIfStmt)
					currentIfStmt = nil
					ifBodyLines = nil
				} else {
					// then分支结束，检查是否有else
					// 解析then分支内的语句
					for j, bodyLine := range ifBodyLines {
						bodyLine = strings.TrimSpace(bodyLine)
						if bodyLine != "" {
							stmt, err := p.parseStatement(bodyLine, i-len(ifBodyLines)+j+1)
							if err != nil {
								return nil, fmt.Errorf("if statement then body error: %v", err)
							}
							if stmt != nil {
								currentIfStmt.ThenBody = append(currentIfStmt.ThenBody, stmt)
							}
						}
					}
					ifBodyLines = nil

					// then分支已结束，等待可能的else
					thenBranchEnded = true
					// 注意：这里不立即添加到program.Statements，等待检查是否有else
				}
			} else if strings.Contains(line, "} else if ") {
				// 处理 } else if condition { 语法
				// 解析 else if condition 部分
				elseIfLine := strings.TrimSpace(line[1:]) // 移除开头的 }
				elseIfStmt, err := p.parseElseIfStmt(elseIfLine, i)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid else if statement: %v", i, err)
				}

				// 将 else if 作为嵌套的 if 语句放入 else 分支
				currentIfStmt.ElseBody = []entities.ASTNode{elseIfStmt}

				// 设置新的 currentIfStmt 为 else if 语句
				currentIfStmt = elseIfStmt.(*entities.IfStmt)
				inIfBody = true
				parsingElse = false
				ifBodyLines = []string{}
			} else if strings.Contains(line, "} else {") {
				// 先处理then分支（} else { 表示then分支结束）

				// 解析then分支内的语句
				for j, bodyLine := range ifBodyLines {
					bodyLine = strings.TrimSpace(bodyLine)
					if bodyLine != "" {
						stmt, err := p.parseStatement(bodyLine, i-len(ifBodyLines)+j+1)
						if err != nil {
							return nil, fmt.Errorf("if statement then body error: %v", err)
						}
						if stmt != nil {
							currentIfStmt.ThenBody = append(currentIfStmt.ThenBody, stmt)
						}
					}
				}

				// 然后进入else分支
				parsingElse = true
				ifBodyLines = []string{}
			} else if strings.TrimSpace(line) == "else" && !parsingElse {
				// 进入else分支（else在单独一行）
				parsingElse = true
				ifBodyLines = []string{}
			} else {
				// 收集if语句体内的行（then分支或else分支）
				ifBodyLines = append(ifBodyLines, line)
			}
		} else if currentFunction != nil && line == "{" {
			// 函数体的开始
			inFunctionBody = true
			functionBodyLines = []string{}
		} else {
			// 不在函数体内或if体内，正常解析
			if strings.HasPrefix(line, "func (") && strings.Contains(line, "{") {
				// 方法定义开始（多行）
				methodDef, err := p.parseMethodDef(line, i+1)
				if err != nil {
					return nil, err
				}
				currentMethodDef = methodDef.(*entities.MethodDef)
				inMethodBody = true
				methodBodyLines = []string{}
			} else if strings.HasPrefix(line, "func ") {
				// 函数定义开始
				funcDef, err := p.parseFuncDef(line, i+1)
				if err != nil {
					return nil, err
				}
				currentFunction = funcDef.(*entities.FuncDef)

				// 检查是否在同一行有{
				if strings.Contains(line, "{") {
					inFunctionBody = true
					functionBodyLines = []string{}
				} else {
					// 等待下一行出现{
					// 这里可以设置一个标志，表示正在等待函数体的开始
				}
			} else if strings.HasPrefix(line, "if ") && strings.Contains(line, "{") {
				// if语句开始（多行）
				ifStmt, err := p.parseIfStmt(line, i+1)
				if err != nil {
					return nil, err
				}
				currentIfStmt = ifStmt.(*entities.IfStmt)
				inIfBody = true
				parsingElse = false
				ifBodyLines = []string{}
			} else if strings.HasPrefix(line, "while ") && strings.Contains(line, "{") {
				// while语句开始（多行）
				whileStmt, err := p.parseWhileStmt(line, i+1)
				if err != nil {
					return nil, err
				}
				currentWhileStmt = whileStmt.(*entities.WhileStmt)
				inWhileBody = true
				whileBodyLines = []string{}
			} else if strings.HasPrefix(line, "for ") && strings.Contains(line, "{") {
				// for语句开始（多行）
				forStmt, err := p.parseForStmt(line, i+1)
				if err != nil {
					return nil, err
				}
				currentForStmt = forStmt.(*entities.ForStmt)
				inForBody = true
				forBodyLines = []string{}
			} else if strings.HasPrefix(line, "@impl ") {
				// @impl注解，支持泛型参数
				implLine := strings.TrimSpace(line[6:]) // 移除"@impl "

				// 解析trait名称和泛型参数
				var traitName string
				var typeArgs []string

				if strings.Contains(implLine, "[") {
					// 有泛型参数，如 Printable[int]
					bracketStart := strings.Index(implLine, "[")
					traitName = strings.TrimSpace(implLine[:bracketStart])

					bracketEnd := strings.LastIndex(implLine, "]")
					if bracketEnd == -1 {
						return nil, fmt.Errorf("line %d: invalid @impl syntax, missing closing bracket", i+1)
					}

					typeArgsStr := strings.TrimSpace(implLine[bracketStart+1 : bracketEnd])
					if typeArgsStr != "" {
						// 解析逗号分隔的类型参数
						args := strings.Split(typeArgsStr, ",")
						for _, arg := range args {
							typeArgs = append(typeArgs, strings.TrimSpace(arg))
						}
					}
				} else {
					// 无泛型参数
					traitName = strings.TrimSpace(implLine)
				}

				pendingImplTraits = append(pendingImplTraits, entities.ImplAnnotation{
					TraitName: traitName,
					TypeArgs:  typeArgs,
				})
			} else if strings.HasPrefix(line, "type ") && strings.Contains(line, " = ") {
				// 关联类型实现：type Item = int;
				// 可以跟在@impl注解之后，也可以跟在struct定义之后
				if len(pendingImplTraits) > 0 || currentStructDef != nil {
					// 解析关联类型实现
					typeLine := strings.TrimSpace(line[5:]) // 移除"type "
					typeLine = strings.TrimSuffix(typeLine, ";")

					equalIndex := strings.Index(typeLine, " = ")
					if equalIndex == -1 {
						return nil, fmt.Errorf("line %d: invalid associated type implementation", i+1)
					}

					typeName := strings.TrimSpace(typeLine[:equalIndex])
					targetType := strings.TrimSpace(typeLine[equalIndex+3:])

					// 如果有当前struct，立即应用到它
					if currentStructDef != nil {
						currentStructDef.AssociatedTypeImpls = append(currentStructDef.AssociatedTypeImpls, entities.AssociatedTypeImpl{
							Name:       typeName,
							TargetType: targetType,
						})
					} else {
						// 否则加入pending列表
						pendingAssociatedTypeImpls = append(pendingAssociatedTypeImpls, entities.AssociatedTypeImpl{
							Name:       typeName,
							TargetType: targetType,
						})
					}
				} else {
					return nil, fmt.Errorf("line %d: associated type implementation must follow @impl annotation or struct definition", i+1)
				}
			} else if strings.HasPrefix(line, "struct ") && strings.Contains(line, "{") {
				// struct定义开始（多行），应用pending的impl traits
				structDef, err := p.parseStructDef(line, i+1)
				if err != nil {
					return nil, err
				}
				currentStructDef = structDef.(*entities.StructDef)
				// 应用pending的impl traits和关联类型实现
				currentStructDef.ImplTraits = append(currentStructDef.ImplTraits, pendingImplTraits...)
				currentStructDef.AssociatedTypeImpls = append(currentStructDef.AssociatedTypeImpls, pendingAssociatedTypeImpls...)
				pendingImplTraits = nil          // 清空pending traits
				pendingAssociatedTypeImpls = nil // 清空pending associated types
				inStructBody = true
				structBodyLines = []string{}
			} else if strings.HasPrefix(line, "enum ") && strings.Contains(line, "{") {
				// enum定义开始（多行），应用pending的impl traits
				enumDef, err := p.parseEnumDef(line, i+1)
				if err != nil {
					return nil, err
				}
				currentEnumDef = enumDef.(*entities.EnumDef)
				// 应用pending的impl traits
				currentEnumDef.ImplTraits = append(currentEnumDef.ImplTraits, pendingImplTraits...)
				pendingImplTraits = nil // 清空pending traits
				inEnumBody = true
				enumBodyLines = []string{}
			} else if strings.HasPrefix(line, "trait ") && strings.Contains(line, "{") {
				// 解析完整的trait定义
				traitDef, consumedLines, err := p.parseTraitDef(line, i+1, lines)
				if err != nil {
					return nil, err
				}
				program.Statements = append(program.Statements, traitDef)
				i += consumedLines - 1 // 跳过已处理的行

				// 对于多行trait，跳过结束的}
				if consumedLines > 1 && i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "}" {
					i++ // 跳过}
				}
				continue
			} else if strings.HasPrefix(line, "func (") && strings.Contains(line, "{") {
				// 方法定义开始（多行）
				methodDef, err := p.parseMethodDef(line, i+1)
				if err != nil {
					return nil, err
				}
				currentMethodDef = methodDef.(*entities.MethodDef)
				inMethodBody = true
				methodBodyLines = []string{}
			} else if strings.HasPrefix(line, "match ") && strings.Contains(line, "{") {
				// match语句开始（多行）
				matchStmt, err := p.parseMatchStmt(line, i+1)
				if err != nil {
					return nil, err
				}
				currentMatchStmt = matchStmt.(*entities.MatchStmt)
				inMatchBody = true
				matchBodyLines = []string{}
			} else if strings.HasPrefix(line, "async func ") && strings.Contains(line, "{") {
				// async函数定义开始（多行）
				asyncFuncDef, err := p.parseAsyncFuncDef(line, i+1)
				if err != nil {
					return nil, err
				}
				currentAsyncFunc = asyncFuncDef.(*entities.AsyncFuncDef)
				inAsyncFunctionBody = true
				asyncFunctionBodyLines = []string{}
			} else {
				// 普通语句
				stmt, err := p.parseStatement(line, i+1)
				if err != nil {
					return nil, err
				}
				if stmt != nil {
					program.Statements = append(program.Statements, stmt)
				}
			}
		}
		i++
	}

	// 检查是否有未结束的函数
	if inFunctionBody {
		return nil, fmt.Errorf("function %s body not closed with }", currentFunction.Name)
	}

	return program, nil
}

// parseStatement 解析单个语句
func (p *SimpleParser) parseStatement(line string, lineNum int) (entities.ASTNode, error) {
	line = strings.TrimSpace(line)

	// 忽略块结束符
	if line == "}" {
		return nil, nil
	}

	// 解析 print 语句
	if strings.HasPrefix(line, "print ") {
		value := strings.TrimSpace(line[6:])
		if value == "" {
			return nil, fmt.Errorf("line %d: print statement requires an expression", lineNum)
		}

		// 移除注释部分
		if commentIndex := strings.Index(value, "//"); commentIndex != -1 {
			value = strings.TrimSpace(value[:commentIndex])
		}

		// 解析print后的表达式
		expr, err := p.parseExpr(value)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid print expression: %v", lineNum, err)
		}

		return &entities.PrintStmt{
			Value: expr,
		}, nil
	}

	// 解析变量声明: let name: type = value
	if strings.HasPrefix(line, "let ") {
		return p.parseVarDecl(line, lineNum)
	}

	// 解析赋值语句: variable = expression
	if strings.Contains(line, " = ") && !strings.Contains(line, ": ") {
		return p.parseAssignStmt(line, lineNum)
	}

	// 解析方法定义: func (r ReceiverType) methodName(params) -> returnType {
	if strings.HasPrefix(line, "func (") {
		return p.parseMethodDef(line, lineNum)
	}

	// 函数定义由Parse函数处理，这里不处理

	// 解析match语句: match value { case1 => stmt1, case2 => stmt2 }
	if strings.HasPrefix(line, "match ") {
		return p.parseMatchStmt(line, lineNum)
	}

	// 解析 if 语句: if condition {
	if strings.HasPrefix(line, "if ") {
		return p.parseIfStmt(line, lineNum)
	}

	// 解析 while 循环: while condition {
	if strings.HasPrefix(line, "while ") {
		return p.parseWhileStmt(line, lineNum)
	}

	// 解析 for 循环: for init; condition; increment {
	if strings.HasPrefix(line, "for ") {
		return p.parseForStmt(line, lineNum)
	}

	// 解析结构体定义: struct Name { field1: type1, field2: type2 }
	if strings.HasPrefix(line, "struct ") {
		return p.parseStructDef(line, lineNum)
	}

	// 解析枚举定义: enum Name { Variant1, Variant2, Variant3 }
	if strings.HasPrefix(line, "enum ") {
		return p.parseEnumDef(line, lineNum)
	}

	// trait定义由Parse函数处理，这里不处理

	// 解析方法调用: receiver.method(args)
	if p.isMethodCall(line) {
		return p.parseMethodCall(line, lineNum)
	}

	// 解析函数调用: funcName(arg1, arg2, ...)
	if p.isFunctionCall(line) {
		return p.parseFuncCall(line, lineNum)
	}

	// 解析异步表达式: await expr, spawn expr, chan type, channel operations
	if p.isAsyncExpression(line) {
		return p.parseAsyncStatement(line, lineNum)
	}

	// 解析 select 语句: select { case ... }
	if strings.HasPrefix(line, "select ") {
		return p.parseSelectStmt(line, lineNum)
	}

	// 解析 return 语句: return expr
	if strings.HasPrefix(line, "return ") {
		return p.parseReturnStmt(line, lineNum)
	}

	// 解析 break 语句
	if line == "break" {
		return &entities.BreakStmt{}, nil
	}

	// 解析 continue 语句
	if line == "continue" {
		return &entities.ContinueStmt{}, nil
	}

	// 解析异步表达式语句 (spawn, await等)
	if p.isAsyncExpression(line) {
		return p.parseAsyncStatement(line, lineNum)
	}

	return nil, fmt.Errorf("line %d: unknown statement: %s", lineNum, line)
}

// parseSelectStmt 解析select语句
func (p *SimpleParser) parseSelectStmt(line string, lineNum int) (entities.ASTNode, error) {
	// select语句必须以"select {"开始
	if !strings.HasPrefix(line, "select {") {
		return nil, fmt.Errorf("line %d: select statement must start with 'select {'", lineNum)
	}

	selectStmt := entities.NewSelectStmt()

	// 简化实现：select语句需要多行解析
	// 这里返回基础结构，实际case分支解析需要在Parse函数中实现
	return selectStmt, nil
}

// parseFuncCall 解析函数调用
func (p *SimpleParser) parseFuncCall(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：funcName(args) 或 funcName[T](args) 或 funcName[T, U](args)

	line = strings.TrimSpace(line)

	// 查找函数名
	nameEnd := strings.IndexAny(line, "[(")
	if nameEnd == -1 {
		return nil, fmt.Errorf("line %d: invalid function call", lineNum)
	}

	funcName := strings.TrimSpace(line[:nameEnd])

	// 解析类型参数
	var typeArgs []string
	remaining := line[nameEnd:]

	if strings.HasPrefix(remaining, "[") {
		// 有类型参数
		closeIndex := strings.Index(remaining, "]")
		if closeIndex == -1 {
			return nil, fmt.Errorf("line %d: unclosed type parameter brackets", lineNum)
		}

		typeArgStr := remaining[1:closeIndex]
		if typeArgStr != "" {
			typeArgs = strings.Split(typeArgStr, ",")
			for i, arg := range typeArgs {
				typeArgs[i] = strings.TrimSpace(arg)
			}
		}

		remaining = remaining[closeIndex+1:]
	}

	// 解析参数
	if !strings.HasPrefix(remaining, "(") {
		return nil, fmt.Errorf("line %d: function call missing parentheses", lineNum)
	}

	closeParen := strings.Index(remaining, ")")
	if closeParen == -1 {
		return nil, fmt.Errorf("line %d: unclosed function call parentheses", lineNum)
	}

	argStr := strings.TrimSpace(remaining[1:closeParen])

	var args []entities.Expr
	if argStr != "" {
		// 简单参数解析（目前只支持字面量和标识符）
		argParts := strings.Split(argStr, ",")
		for _, argPart := range argParts {
			argPart = strings.TrimSpace(argPart)
			if argPart != "" {
				expr, err := p.parseExpr(argPart)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid argument %s: %v", lineNum, argPart, err)
				}
				args = append(args, expr)
			}
		}
	}

	return &entities.FuncCall{
		Name:     funcName,
		TypeArgs: typeArgs,
		Args:     args,
	}, nil
}

// parseVarDecl 解析变量声明
func (p *SimpleParser) parseVarDecl(line string, lineNum int) (entities.ASTNode, error) {
	// let name: type = value
	parts := strings.Split(line[4:], "=")
	if len(parts) != 2 {
		return nil, fmt.Errorf("line %d: invalid variable declaration", lineNum)
	}

	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	// 解析变量名和类型（支持类型推断）
	var name, varType string
	var inferred bool

	nameType := strings.Split(left, ":")
	if len(nameType) == 2 {
		// 显式类型注解：let name: type
		name = strings.TrimSpace(nameType[0])
		varType = strings.TrimSpace(nameType[1])
		inferred = false
	} else if len(nameType) == 1 {
		// 类型推断：let name（没有冒号）
		name = strings.TrimSpace(nameType[0])
		varType = "" // 类型将在后续推断阶段确定
		inferred = true
	} else {
		return nil, fmt.Errorf("line %d: invalid variable declaration syntax", lineNum)
	}
	value, err := p.parseExpr(right)
	if err != nil {
		return nil, fmt.Errorf("line %d: %v", lineNum, err)
	}

	return &entities.VarDecl{
		Name:     name,
		Type:     varType,
		Value:    value,
		Inferred: inferred,
	}, nil
}

// parseAssignStmt 解析赋值语句: variable = expression
func (p *SimpleParser) parseAssignStmt(line string, lineNum int) (entities.ASTNode, error) {
	// 查找第一个" = "
	equalIndex := strings.Index(line, " = ")
	if equalIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid assignment statement", lineNum)
	}

	// 获取变量名和表达式
	varName := strings.TrimSpace(line[:equalIndex])
	valueStr := strings.TrimSpace(line[equalIndex+3:])

	// 验证变量名
	if varName == "" {
		return nil, fmt.Errorf("line %d: missing variable name in assignment", lineNum)
	}

	// 解析赋值表达式
	value, err := p.parseExpr(valueStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid assignment value: %v", lineNum, err)
	}

	return &entities.AssignStmt{
		Name:  varName,
		Value: value,
	}, nil
}

// parseFuncDef 解析函数定义
// parseGenericParams 解析泛型参数，如 [T], [T: Trait], [T, U: Trait]
func (p *SimpleParser) parseGenericParams(typeParamStr string) ([]entities.GenericParam, error) {
	var typeParams []entities.GenericParam

	if typeParamStr == "" {
		return typeParams, nil
	}

	// 移除方括号
	inner := strings.TrimSpace(typeParamStr[1 : len(typeParamStr)-1])
	if inner == "" {
		return typeParams, nil
	}

	// 按逗号分割参数
	paramStrs := strings.Split(inner, ",")
	for _, paramStr := range paramStrs {
		paramStr = strings.TrimSpace(paramStr)
		if paramStr == "" {
			continue
		}

		// 检查是否有约束 (T: Trait)
		var name, constraintsStr string
		if strings.Contains(paramStr, ":") {
			parts := strings.SplitN(paramStr, ":", 2)
			name = strings.TrimSpace(parts[0])
			constraintsStr = strings.TrimSpace(parts[1])
		} else {
			name = paramStr
		}

		// 解析约束
		var constraints []string
		if constraintsStr != "" {
			// 支持多个约束，用 + 分隔
			constraintList := strings.Split(constraintsStr, "+")
			for _, c := range constraintList {
				constraints = append(constraints, strings.TrimSpace(c))
			}
		}

		typeParams = append(typeParams, entities.GenericParam{
			Name:        name,
			Constraints: constraints,
		})
	}

	return typeParams, nil
}

func (p *SimpleParser) parseFuncDef(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：func name[T](param1: type1, param2: type2) -> returnType { body }
	// 支持泛型：func name[T: Trait](param1: type1) -> returnType
	// 简化版本：func name(param: type) -> returnType

	funcLine := strings.TrimSpace(line[5:])

	// 查找函数名和泛型参数
	nameEnd := strings.IndexAny(funcLine, "( [")
	if nameEnd == -1 {
		return nil, fmt.Errorf("line %d: invalid function definition", lineNum)
	}

	name := strings.TrimSpace(funcLine[:nameEnd])

	// 解析泛型参数
	var typeParams []entities.GenericParam
	remainingLine := funcLine[nameEnd:]

	if strings.HasPrefix(remainingLine, "[") {
		// 有泛型参数
		gtIndex := strings.Index(remainingLine, "]")
		if gtIndex == -1 {
			return nil, fmt.Errorf("line %d: unclosed generic parameter brackets", lineNum)
		}

		typeParamStr := remainingLine[:gtIndex+1]
		var err error
		typeParams, err = p.parseGenericParams(typeParamStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid generic parameters: %v", lineNum, err)
		}

		remainingLine = remainingLine[gtIndex+1:]
	}

	// 解析参数
	var params []entities.Param
	if strings.Contains(funcLine, "(") && strings.Contains(funcLine, ")") {
		paramsStart := strings.Index(funcLine, "(")
		paramsEnd := strings.Index(funcLine, ")")
		if paramsStart != -1 && paramsEnd != -1 && paramsEnd > paramsStart {
			paramsStr := strings.TrimSpace(funcLine[paramsStart+1 : paramsEnd])
			if paramsStr != "" {
				// 解析多个参数: param1: type1, param2: type2
				paramList := strings.Split(paramsStr, ",")
				for _, paramStr := range paramList {
					paramStr = strings.TrimSpace(paramStr)
					if strings.Contains(paramStr, ":") {
						parts := strings.Split(paramStr, ":")
						if len(parts) == 2 {
							paramName := strings.TrimSpace(parts[0])
							paramType := strings.TrimSpace(parts[1])
							params = append(params, entities.Param{Name: paramName, Type: paramType})
						}
					}
				}
			}
		}
	}

	// 解析返回类型
	returnType := "void"
	if strings.Contains(funcLine, "->") {
		arrowIndex := strings.Index(funcLine, "->")
		returnPart := strings.TrimSpace(funcLine[arrowIndex+2:])
		// 移除 { 如果存在（多行函数体）
		returnType = strings.TrimSpace(strings.TrimSuffix(returnPart, "{"))

		// 如果没有{，说明是多行函数定义
		if !strings.Contains(funcLine, "{") {
			// 对于多行函数定义，返回类型就是整个returnPart
			returnType = strings.TrimSpace(returnPart)
		}

		// 解析类型（支持Result[int]和Option[string]）
		parsedType, err := p.parseType(returnType)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid return type: %v", lineNum, err)
		}
		returnType = parsedType
	}

	// 对于多行函数体，函数体将在Parse函数中单独解析
	return &entities.FuncDef{
		Name:       name,
		TypeParams: typeParams, // 泛型类型参数
		Params:     params,
		ReturnType: returnType,
		Body:       []entities.ASTNode{}, // 函数体将在多行解析中填充
	}, nil
}

// parseIfStmt 解析if语句
func (p *SimpleParser) parseIfStmt(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：if condition { ... } else { ... }
	// 多行版本：只解析条件部分，语句块在Parse函数中处理

	ifLine := strings.TrimSpace(line[3:]) // 移除"if "

	// 找到条件和 { 的分界点
	braceIndex := strings.Index(ifLine, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: if statement must have opening brace", lineNum)
	}

	conditionStr := strings.TrimSpace(ifLine[:braceIndex])

	condition, err := p.parseExpr(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid condition: %v", lineNum, err)
	}

	return &entities.IfStmt{
		Condition: condition,
		ThenBody:  []entities.ASTNode{}, // 语句块将在多行解析中填充
		ElseBody:  []entities.ASTNode{}, // else分支将在多行解析中填充
	}, nil
}

// parseElseIfStmt 解析else if语句
func (p *SimpleParser) parseElseIfStmt(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：else if condition { ... }
	// 多行版本：只解析条件部分，语句块在Parse函数中处理

	elseIfLine := strings.TrimSpace(line[7:]) // 移除"else if "

	// 找到条件和 { 的分界点
	braceIndex := strings.Index(elseIfLine, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: else if statement must have opening brace", lineNum)
	}

	conditionStr := strings.TrimSpace(elseIfLine[:braceIndex])

	condition, err := p.parseExpr(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid condition: %v", lineNum, err)
	}

	return &entities.IfStmt{
		Condition: condition,
		ThenBody:  []entities.ASTNode{}, // 语句块将在多行解析中填充
		ElseBody:  []entities.ASTNode{}, // else分支将在多行解析中填充
	}, nil
}

// parseForStmt 解析for循环
func (p *SimpleParser) parseForStmt(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：for condition { ... }
	// 多行版本：只解析条件部分，语句块在Parse函数中处理

	forLine := strings.TrimSpace(line[4:]) // 移除"for "

	// 找到条件和 { 的分界点
	braceIndex := strings.Index(forLine, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: for statement must have opening brace", lineNum)
	}

	conditionStr := strings.TrimSpace(forLine[:braceIndex])

	condition, err := p.parseExpr(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid condition: %v", lineNum, err)
	}

	return &entities.ForStmt{
		Init:      nil, // 不支持初始化
		Condition: condition,
		Increment: nil,                  // 不支持递增
		Body:      []entities.ASTNode{}, // 语句块将在多行解析中填充
	}, nil
}

// parseWhileStmt 解析while循环
func (p *SimpleParser) parseWhileStmt(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：while condition { ... }
	// 多行版本：只解析条件部分，语句块在Parse函数中处理

	whileLine := strings.TrimSpace(line[6:]) // 移除"while "

	// 找到条件和 { 的分界点
	braceIndex := strings.Index(whileLine, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: while statement must have opening brace", lineNum)
	}

	conditionStr := strings.TrimSpace(whileLine[:braceIndex])

	condition, err := p.parseExpr(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid condition: %v", lineNum, err)
	}

	return &entities.WhileStmt{
		Condition: condition,
		Body:      []entities.ASTNode{}, // 语句块将在多行解析中填充
	}, nil
}

// parseStructDef 解析结构体定义
func (p *SimpleParser) parseStructDef(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：struct Name[T] { ... } 或 struct Name { ... }
	// 多行版本：只解析结构体名称和泛型参数，字段在Parse函数中处理

	structLine := strings.TrimSpace(line[7:]) // 移除"struct "

	// 解析结构体名和泛型参数
	nameEnd := strings.IndexAny(structLine, "{ [")
	if nameEnd == -1 {
		return nil, fmt.Errorf("line %d: invalid struct definition", lineNum)
	}

	structName := strings.TrimSpace(structLine[:nameEnd])

	// 解析泛型参数
	var typeParams []entities.GenericParam
	remainingLine := structLine[nameEnd:]

	if strings.HasPrefix(remainingLine, "[") {
		// 有泛型参数
		bracketEnd := strings.Index(remainingLine, "]")
		if bracketEnd == -1 {
			return nil, fmt.Errorf("line %d: unclosed generic parameter brackets", lineNum)
		}

		typeParamStr := remainingLine[:bracketEnd+1] // 包含右括号
		var err error
		typeParams, err = p.parseGenericParams(typeParamStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid generic parameters: %v", lineNum, err)
		}
	}

	return &entities.StructDef{
		Name:       structName,
		TypeParams: typeParams,                  // 泛型类型参数
		ImplTraits: []entities.ImplAnnotation{}, // impl traits将在Parse函数中设置
		Fields:     []entities.StructField{},    // 字段将在多行解析中填充
	}, nil
}

// parseEnumDef 解析枚举定义
func (p *SimpleParser) parseEnumDef(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：enum Name { Variant1, Variant2, Variant3 }
	// 单行版本：解析枚举名称和所有枚举值

	enumLine := strings.TrimSpace(line[5:]) // 移除"enum "

	// 找到枚举名和 { 的分界点
	braceIndex := strings.Index(enumLine, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: enum definition must have opening brace", lineNum)
	}

	enumName := strings.TrimSpace(enumLine[:braceIndex])

	// 解析枚举值
	variantStr := strings.TrimSpace(enumLine[braceIndex+1:])
	variantStr = strings.TrimSuffix(variantStr, "}")

	var variants []entities.EnumVariant
	if variantStr != "" {
		variantParts := strings.Split(variantStr, ",")
		for _, variantPart := range variantParts {
			variantName := strings.TrimSpace(variantPart)
			if variantName != "" {
				variants = append(variants, entities.EnumVariant{Name: variantName})
			}
		}
	}

	return &entities.EnumDef{
		Name:       enumName,
		ImplTraits: []entities.ImplAnnotation{}, // impl traits将在Parse函数中设置
		Variants:   variants,
	}, nil
}

// parseMethodDef 解析方法定义
func (p *SimpleParser) parseMethodDef(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：
	// func (r ReceiverType) methodName(params) -> returnType {
	// func (r Receiver[T]) methodName[U](params) -> returnType {  // 泛型方法
	// 多行版本：只解析方法头，方法体在Parse函数中处理

	methodLine := strings.TrimSpace(line[5:]) // 移除"func "

	// 解析接收者: (r ReceiverType) 或 (r Receiver[T])
	if !strings.HasPrefix(methodLine, "(") {
		return nil, fmt.Errorf("line %d: method definition must start with receiver", lineNum)
	}

	receiverEnd := strings.Index(methodLine, ")")
	if receiverEnd == -1 {
		return nil, fmt.Errorf("line %d: invalid receiver syntax", lineNum)
	}

	receiverStr := strings.TrimSpace(methodLine[1:receiverEnd])
	receiverParts := strings.Fields(receiverStr)
	if len(receiverParts) < 2 {
		return nil, fmt.Errorf("line %d: invalid receiver format", lineNum)
	}

	receiverVar := receiverParts[0]
	receiverTypeStr := strings.Join(receiverParts[1:], " ")

	// 解析接收者类型参数（如 Receiver[T]）
	var receiverTypeParams []entities.GenericParam
	if strings.Contains(receiverTypeStr, "[") && strings.HasSuffix(receiverTypeStr, "]") {
		bracketStart := strings.Index(receiverTypeStr, "[")
		baseType := strings.TrimSpace(receiverTypeStr[:bracketStart])
		typeParamStr := receiverTypeStr[bracketStart:]

		var err error
		receiverTypeParams, err = p.parseGenericParams(typeParamStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid receiver type parameters: %v", lineNum, err)
		}
		receiverTypeStr = baseType
	}

	// 解析方法名和可能的泛型参数
	remaining := strings.TrimSpace(methodLine[receiverEnd+1:])

	// 查找方法名结束位置（遇到(、[或空格）
	methodNameEnd := strings.IndexAny(remaining, "([ ")
	if methodNameEnd == -1 {
		return nil, fmt.Errorf("line %d: invalid method name", lineNum)
	}

	methodName := strings.TrimSpace(remaining[:methodNameEnd])

	// 解析方法类型参数（如 methodName[T, U]）
	var methodTypeParams []entities.GenericParam
	remainingAfterName := strings.TrimSpace(remaining[methodNameEnd:])
	if strings.HasPrefix(remainingAfterName, "[") {
		bracketEnd := strings.Index(remainingAfterName, "]")
		if bracketEnd == -1 {
			return nil, fmt.Errorf("line %d: unclosed method type parameter brackets", lineNum)
		}
		typeParamStr := remainingAfterName[:bracketEnd+1]
		remainingAfterName = strings.TrimSpace(remainingAfterName[bracketEnd+1:])

		var err error
		methodTypeParams, err = p.parseGenericParams(typeParamStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid method type parameters: %v", lineNum, err)
		}
	}

	// 解析参数
	var params []entities.Param
	if strings.HasPrefix(remainingAfterName, "(") {
		paramsEnd := strings.Index(remainingAfterName, ")")
		if paramsEnd == -1 {
			return nil, fmt.Errorf("line %d: unclosed parameter list", lineNum)
		}
		paramsStr := strings.TrimSpace(remainingAfterName[1:paramsEnd])
		remainingAfterName = strings.TrimSpace(remainingAfterName[paramsEnd+1:])

		if paramsStr != "" {
			paramList := strings.Split(paramsStr, ",")
			for _, paramStr := range paramList {
				paramStr = strings.TrimSpace(paramStr)
				if strings.Contains(paramStr, ":") {
					parts := strings.Split(paramStr, ":")
					if len(parts) == 2 {
						paramName := strings.TrimSpace(parts[0])
						paramType := strings.TrimSpace(parts[1])
						params = append(params, entities.Param{Name: paramName, Type: paramType})
					}
				}
			}
		}
	}

	// 解析返回类型
	returnType := "void"
	if strings.HasPrefix(remainingAfterName, "->") {
		returnPart := strings.TrimSpace(remainingAfterName[2:])
		returnType = strings.TrimSpace(strings.TrimSuffix(returnPart, "{"))

		// 解析类型（支持Result[int]和Option[string]）
		parsedType, err := p.parseType(returnType)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid return type: %v", lineNum, err)
		}
		returnType = parsedType
	}

	return &entities.MethodDef{
		Receiver:       receiverTypeStr,
		ReceiverVar:    receiverVar,
		ReceiverParams: receiverTypeParams, // 接收者类型参数
		TypeParams:     methodTypeParams,   // 方法类型参数
		Name:           methodName,
		Params:         params,
		ReturnType:     returnType,
		Body:           []entities.ASTNode{}, // 方法体将在多行解析中填充
	}, nil
}

// parseExpr 解析表达式
func (p *SimpleParser) parseExpr(expr string) (entities.Expr, error) {
	expr = strings.TrimSpace(expr)

	// 移除行内注释
	if commentIndex := strings.Index(expr, "//"); commentIndex >= 0 {
		expr = strings.TrimSpace(expr[:commentIndex])
	}

	// 处理括号表达式
	if len(expr) >= 2 && expr[0] == '(' && expr[len(expr)-1] == ')' {
		innerExpr := strings.TrimSpace(expr[1 : len(expr)-1])
		return p.parseExpr(innerExpr)
	}

	// 字符串字面量
	if len(expr) >= 2 && expr[0] == '"' && expr[len(expr)-1] == '"' {
		return &entities.StringLiteral{
			Value: expr[1 : len(expr)-1],
		}, nil
	}

	// 整数字面量
	if intVal, err := strconv.Atoi(expr); err == nil {
		return &entities.IntLiteral{
			Value: intVal,
		}, nil
	}

	// 浮点数字面量
	if floatVal, err := strconv.ParseFloat(expr, 64); err == nil {
		return &entities.FloatLiteral{
			Value: floatVal,
		}, nil
	}

	// match表达式 (如 match value { Case => action })
	if strings.HasPrefix(expr, "match ") {
		return p.parseMatchExpr(expr)
	}

	// await表达式 (如 await asyncCall())
	if strings.HasPrefix(expr, "await ") {
		inner := strings.TrimSpace(expr[6:])
		asyncExpr, err := p.parseExpr(inner)
		if err != nil {
			return nil, fmt.Errorf("invalid await expression: %v", err)
		}
		result := entities.NewAwaitExpr(asyncExpr)
		return result, nil
	}

	// spawn表达式 (如 spawn funcName(args))
	if strings.HasPrefix(expr, "spawn ") {
		inner := strings.TrimSpace(expr[6:])
		if !strings.Contains(inner, "(") {
			return nil, fmt.Errorf("invalid spawn expression: missing parentheses")
		}
		parenIndex := strings.Index(inner, "(")
		funcName := strings.TrimSpace(inner[:parenIndex])
		argsStr := inner[parenIndex+1:]
		if !strings.HasSuffix(argsStr, ")") {
			return nil, fmt.Errorf("invalid spawn expression: missing closing parenthesis")
		}
		argsStr = strings.TrimSpace(argsStr[:len(argsStr)-1])

		// 解析函数名（可以是标识符或更复杂的表达式）
		funcExpr, err := p.parseExpr(funcName)
		if err != nil {
			return nil, err
		}

		// 解析参数
		var args []entities.Expr
		if argsStr != "" {
			argStrs := strings.Split(argsStr, ",")
			for _, argStr := range argStrs {
				argStr = strings.TrimSpace(argStr)
				if argStr != "" {
					arg, err := p.parseExpr(argStr)
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
				}
			}
		}

		return entities.NewSpawnExpr(funcExpr, args), nil
	}

	// 通道字面量 (如 chan int)
	if strings.HasPrefix(expr, "chan ") {
		typeStr := strings.TrimSpace(expr[5:])
		return entities.NewChanLiteral(typeStr), nil
	}

	// 发送表达式 (如 channel <- value)
	if strings.Contains(expr, " <- ") {
		parts := strings.SplitN(expr, " <- ", 2)
		if len(parts) == 2 {
			channelExpr, err := p.parseExpr(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, err
			}
			valueExpr, err := p.parseExpr(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, err
			}
			return entities.NewSendExpr(channelExpr, valueExpr), nil
		}
	}

	// 接收表达式 (如 <- channel)
	if strings.HasPrefix(expr, "<- ") {
		channelStr := strings.TrimSpace(expr[3:])
		channelExpr, err := p.parseExpr(channelStr)
		if err != nil {
			return nil, err
		}
		return entities.NewReceiveExpr(channelExpr), nil
	}

	// 数组字面量 (如 [1, 2, 3])
	if strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]") {
		return p.parseArrayLiteral(expr)
	}

	// len() 函数调用 (如 len(array)) - 必须在结构体字段访问之前处理
	if strings.HasPrefix(expr, "len(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[4 : len(expr)-1])
		arrayExpr, err := p.parseExpr(inner)
		if err != nil {
			return nil, fmt.Errorf("invalid len expression: %v", err)
		}
		return &entities.LenExpr{
			Array: arrayExpr,
		}, nil
	}

	// 结构体字段访问 (如 point.x) - 在len()之后处理
	if strings.Contains(expr, ".") {
		parts := strings.Split(expr, ".")
		if len(parts) == 2 {
			object, err := p.parseExpr(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, err
			}
			field := strings.TrimSpace(parts[1])
			return &entities.StructAccess{
				Object: object,
				Field:  field,
			}, nil
		}
	}

	// Result字面量 (如 Ok(value) 或 Err(error))
	if strings.HasPrefix(expr, "Ok(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[3 : len(expr)-1])
		valueExpr, err := p.parseExpr(inner)
		if err != nil {
			return nil, fmt.Errorf("invalid Ok literal: %v", err)
		}
		return &entities.OkLiteral{Value: valueExpr}, nil
	}
	if strings.HasPrefix(expr, "Err(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[4 : len(expr)-1])
		errorExpr, err := p.parseExpr(inner)
		if err != nil {
			return nil, fmt.Errorf("invalid Err literal: %v", err)
		}
		return &entities.ErrLiteral{Error: errorExpr}, nil
	}

	// Option字面量 (如 Some(value) 或 None)
	if strings.HasPrefix(expr, "Some(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[5 : len(expr)-1])
		valueExpr, err := p.parseExpr(inner)
		if err != nil {
			return nil, fmt.Errorf("invalid Some literal: %v", err)
		}
		return &entities.SomeLiteral{Value: valueExpr}, nil
	}
	if expr == "None" {
		return &entities.NoneLiteral{}, nil
	}

	// 错误传播操作符 (如 expr?)
	if strings.HasSuffix(expr, "?") {
		inner := strings.TrimSpace(expr[:len(expr)-1])
		innerExpr, err := p.parseExpr(inner)
		if err != nil {
			return nil, fmt.Errorf("invalid error propagation: %v", err)
		}
		return &entities.ErrorPropagation{Expr: innerExpr}, nil
	}

	// await表达式 (如 await asyncCall()) - 必须在函数调用之前检查
	if strings.HasPrefix(expr, "await ") {
		inner := strings.TrimSpace(expr[6:])
		asyncExpr, err := p.parseExpr(inner)
		if err != nil {
			return nil, fmt.Errorf("invalid await expression: %v", err)
		}
		return entities.NewAwaitExpr(asyncExpr), nil
	}

	// spawn表达式 (如 spawn funcName(args)) - 必须在函数调用之前检查
	if strings.HasPrefix(expr, "spawn ") {
		inner := strings.TrimSpace(expr[6:])
		if !strings.Contains(inner, "(") {
			return nil, fmt.Errorf("invalid spawn expression: missing parentheses")
		}
		parenIndex := strings.Index(inner, "(")
		funcName := strings.TrimSpace(inner[:parenIndex])
		argsStr := inner[parenIndex+1:]
		if !strings.HasSuffix(argsStr, ")") {
			return nil, fmt.Errorf("invalid spawn expression: missing closing parenthesis")
		}
		argsStr = strings.TrimSpace(argsStr[:len(argsStr)-1])

		funcExpr, err := p.parseExpr(funcName)
		if err != nil {
			return nil, fmt.Errorf("invalid spawn function: %v", err)
		}

		var args []entities.Expr
		if argsStr != "" {
			argStrs := strings.Split(argsStr, ",")
			for _, argStr := range argStrs {
				argStr = strings.TrimSpace(argStr)
				if argStr != "" {
					arg, err := p.parseExpr(argStr)
					if err != nil {
						return nil, fmt.Errorf("invalid spawn argument: %v", err)
					}
					args = append(args, arg)
				}
			}
		}

		return entities.NewSpawnExpr(funcExpr, args), nil
	}

	// 结构体字面量 (如 User{name: "Alice", age: 30})
	if strings.Contains(expr, "{") && strings.Contains(expr, "}") {
		return p.parseStructLiteral(expr)
	}

	// len() 函数调用 (如 len(array)) - 必须在函数调用之前处理
	if strings.HasPrefix(expr, "len(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[4 : len(expr)-1])
		arrayExpr, err := p.parseExpr(inner)
		if err != nil {
			return nil, fmt.Errorf("invalid len expression: %v", err)
		}
		return &entities.LenExpr{
			Array: arrayExpr,
		}, nil
	}

	// 函数调用 (如 funcName(args) 或 funcName[T](args))
	if strings.Contains(expr, "(") && strings.Contains(expr, ")") {
		return p.parseFuncCallExpr(expr)
	}

	// 方法调用 (如 receiver.method(args))
	if strings.Contains(expr, ".") && strings.Contains(expr, "(") && strings.HasSuffix(expr, ")") {
		return p.parseMethodCallExpr(expr)
	}

	// 索引访问 (如 array[index])
	if strings.Contains(expr, "[") && strings.Contains(expr, "]") && !strings.Contains(expr, ":") {
		return p.parseIndexExpr(expr)
	}

	// 切片操作 (如 array[start:end])
	if strings.Contains(expr, "[") && strings.Contains(expr, ":") && strings.Contains(expr, "]") {
		return p.parseSliceExpr(expr)
	}

	// 布尔字面量
	if expr == "true" {
		return &entities.BoolLiteral{Value: true}, nil
	}
	if expr == "false" {
		return &entities.BoolLiteral{Value: false}, nil
	}

	// 标识符（必须在布尔字面量之后）
	if p.isIdentifier(expr) {
		return &entities.Identifier{
			Name: expr,
		}, nil
	}

	// 二元表达式 (算术运算符)
	if strings.Contains(expr, " + ") || strings.Contains(expr, " - ") ||
		strings.Contains(expr, " * ") || strings.Contains(expr, " / ") ||
		strings.Contains(expr, " % ") {
		return p.parseBinaryExpr(expr)
	}

	// 比较运算符
	if strings.Contains(expr, " > ") || strings.Contains(expr, " < ") ||
		strings.Contains(expr, " >= ") || strings.Contains(expr, " <= ") ||
		strings.Contains(expr, " == ") || strings.Contains(expr, " != ") {
		return p.parseComparisonExpr(expr)
	}
	fmt.Fprintf(os.Stderr, "DEBUG: '%s' is not boolean\n", expr)

	// 标识符（变量名）
	// 必须放在最后，因为其他表达式可能包含字母
	if matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, expr); matched {
		fmt.Fprintf(os.Stderr, "DEBUG: Parsed '%s' as Identifier\n", expr)
		return &entities.Identifier{
			Name: expr,
		}, nil
	}

	return nil, fmt.Errorf("unsupported expression: %s", expr)
}

// parseIndexExpr 解析索引访问表达式 (如 array[index])
func (p *SimpleParser) parseIndexExpr(expr string) (entities.Expr, error) {
	// 找到数组表达式和索引表达式
	bracketStart := strings.Index(expr, "[")
	bracketEnd := strings.LastIndex(expr, "]")

	if bracketStart == -1 || bracketEnd == -1 || bracketEnd <= bracketStart {
		return nil, fmt.Errorf("invalid index expression: %s", expr)
	}

	arrayStr := strings.TrimSpace(expr[:bracketStart])
	indexStr := strings.TrimSpace(expr[bracketStart+1 : bracketEnd])

	// 解析数组表达式和索引表达式
	arrayExpr, err := p.parseExpr(arrayStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array expression in index: %v", err)
	}

	indexExpr, err := p.parseExpr(indexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid index expression: %v", err)
	}

	return &entities.IndexExpr{
		Array: arrayExpr,
		Index: indexExpr,
	}, nil
}

// parseSliceExpr 解析切片表达式 (如 array[start:end])
func (p *SimpleParser) parseSliceExpr(expr string) (entities.Expr, error) {
	// 找到数组表达式和切片参数
	bracketStart := strings.Index(expr, "[")
	bracketEnd := strings.LastIndex(expr, "]")

	if bracketStart == -1 || bracketEnd == -1 || bracketEnd <= bracketStart {
		return nil, fmt.Errorf("invalid slice expression: %s", expr)
	}

	arrayStr := strings.TrimSpace(expr[:bracketStart])
	sliceStr := strings.TrimSpace(expr[bracketStart+1 : bracketEnd])

	// 分割start:end
	colonIndex := strings.Index(sliceStr, ":")
	if colonIndex == -1 {
		return nil, fmt.Errorf("invalid slice expression: missing colon")
	}

	startStr := strings.TrimSpace(sliceStr[:colonIndex])
	endStr := strings.TrimSpace(sliceStr[colonIndex+1:])

	// 解析数组表达式
	arrayExpr, err := p.parseExpr(arrayStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array expression in slice: %v", err)
	}

	// 解析start表达式（可选）
	var startExpr entities.Expr
	if startStr != "" {
		startExpr, err = p.parseExpr(startStr)
		if err != nil {
			return nil, fmt.Errorf("invalid start expression in slice: %v", err)
		}
	}

	// 解析end表达式（可选）
	var endExpr entities.Expr
	if endStr != "" {
		endExpr, err = p.parseExpr(endStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end expression in slice: %v", err)
		}
	}

	return &entities.SliceExpr{
		Array: arrayExpr,
		Start: startExpr,
		End:   endExpr,
	}, nil
}

// parseMethodCallExpr 解析方法调用表达式 (如 receiver.method(args) 或 receiver.method[T](args))
func (p *SimpleParser) parseMethodCallExpr(expr string) (entities.Expr, error) {
	// 找到点和括号的位置
	dotIndex := strings.LastIndex(expr, ".")
	if dotIndex == -1 {
		return nil, fmt.Errorf("invalid method call expression: missing dot")
	}

	parenIndex := strings.Index(expr[dotIndex:], "(")
	if parenIndex == -1 {
		return nil, fmt.Errorf("invalid method call expression: missing opening parenthesis")
	}
	parenIndex += dotIndex

	closeParenIndex := strings.LastIndex(expr, ")")
	if closeParenIndex == -1 {
		return nil, fmt.Errorf("invalid method call expression: missing closing parenthesis")
	}

	// 解析接收者
	receiverStr := strings.TrimSpace(expr[:dotIndex])
	receiver, err := p.parseExpr(receiverStr)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver in method call: %v", err)
	}

	// 解析方法名和类型参数
	methodPart := expr[dotIndex+1 : parenIndex]
	var methodName string
	var typeArgs []string

	// 检测是否是数组方法调用
	isArrayMethod := false
	arrayMethods := []string{"push", "pop", "insert", "remove", "clear", "sort", "reverse"}
	for _, m := range arrayMethods {
		if strings.TrimSpace(methodPart) == m || strings.HasPrefix(strings.TrimSpace(methodPart), m+"[") {
			isArrayMethod = true
			break
		}
	}

	if isArrayMethod {
		// 解析数组方法调用
		methodName = strings.TrimSpace(methodPart)
		if strings.Contains(methodName, "[") {
			// 处理可能的类型参数，如 push[int]
			bracketStart := strings.Index(methodName, "[")
			methodName = strings.TrimSpace(methodName[:bracketStart])
		}

		// 解析参数
		argsStr := expr[parenIndex+1 : closeParenIndex]
		var args []entities.Expr

		if strings.TrimSpace(argsStr) != "" {
			// 解析逗号分隔的参数
			argStrs := strings.Split(argsStr, ",")
			for _, argStr := range argStrs {
				argStr = strings.TrimSpace(argStr)
				if argStr != "" {
					arg, err := p.parseExpr(argStr)
					if err != nil {
						return nil, fmt.Errorf("invalid argument in array method call: %v", err)
					}
					args = append(args, arg)
				}
			}
		}

		return &entities.ArrayMethodCallExpr{
			Array:  receiver,
			Method: methodName,
			Args:   args,
		}, nil
	}

	if strings.Contains(methodPart, "[") {
		// 有类型参数，如 method[T]
		bracketStart := strings.Index(methodPart, "[")
		methodName = strings.TrimSpace(methodPart[:bracketStart])

		bracketEnd := strings.LastIndex(methodPart, "]")
		if bracketEnd == -1 {
			return nil, fmt.Errorf("invalid method call: unclosed type parameter brackets")
		}

		typeArgsStr := strings.TrimSpace(methodPart[bracketStart+1 : bracketEnd])
		if typeArgsStr != "" {
			typeArgs = strings.Split(typeArgsStr, ",")
			for i, arg := range typeArgs {
				typeArgs[i] = strings.TrimSpace(arg)
			}
		}
	} else {
		methodName = strings.TrimSpace(methodPart)
	}

	// 解析参数
	argsStr := expr[parenIndex+1 : closeParenIndex]
	var args []entities.Expr

	if strings.TrimSpace(argsStr) != "" {
		// 解析逗号分隔的参数
		argStrs := strings.Split(argsStr, ",")
		for _, argStr := range argStrs {
			argStr = strings.TrimSpace(argStr)
			if argStr != "" {
				arg, err := p.parseExpr(argStr)
				if err != nil {
					return nil, fmt.Errorf("invalid argument in method call: %v", err)
				}
				args = append(args, arg)
			}
		}
	}

	return &entities.MethodCallExpr{
		Receiver:   receiver,
		MethodName: methodName,
		TypeArgs:   typeArgs,
		Args:       args,
	}, nil
}

// parseMatchPattern 解析模式匹配中的模式（支持高级模式）
func (p *SimpleParser) parseMatchPattern(patternStr string) (entities.Expr, error) {
	patternStr = strings.TrimSpace(patternStr)

	// 通配符模式：_
	if patternStr == "_" {
		return &entities.WildcardPattern{}, nil
	}

	// Try模式：Ok(variable)
	if strings.HasPrefix(patternStr, "Ok(") && strings.HasSuffix(patternStr, ")") {
		inner := strings.TrimSpace(patternStr[3 : len(patternStr)-1])
		if inner == "" {
			return &entities.OkPattern{Variable: "_"}, nil // 通配符
		}
		return &entities.OkPattern{Variable: inner}, nil
	}

	// Try模式：Err(variable)
	if strings.HasPrefix(patternStr, "Err(") && strings.HasSuffix(patternStr, ")") {
		inner := strings.TrimSpace(patternStr[4 : len(patternStr)-1])
		if inner == "" {
			return &entities.ErrPattern{Variable: "_"}, nil // 通配符
		}
		return &entities.ErrPattern{Variable: inner}, nil
	}

	// Option模式：Some(variable)
	if strings.HasPrefix(patternStr, "Some(") && strings.HasSuffix(patternStr, ")") {
		inner := strings.TrimSpace(patternStr[5 : len(patternStr)-1])
		if inner == "" {
			return &entities.SomePattern{Variable: "_"}, nil // 通配符
		}
		return &entities.SomePattern{Variable: inner}, nil
	}

	// Option模式：None
	if patternStr == "None" {
		return &entities.NonePattern{}, nil
	}

	// 数组模式：[elem1, elem2, ...rest]
	if strings.HasPrefix(patternStr, "[") && strings.HasSuffix(patternStr, "]") {
		return p.parseArrayPattern(patternStr)
	}

	// 结构体模式：TypeName{field1: pattern1, field2: pattern2}
	if strings.Contains(patternStr, "{") && strings.Contains(patternStr, "}") {
		return p.parseStructPattern(patternStr)
	}

	// 元组模式：(elem1, elem2, elem3)
	if strings.HasPrefix(patternStr, "(") && strings.HasSuffix(patternStr, ")") && !strings.Contains(patternStr, "Ok(") && !strings.Contains(patternStr, "Err(") && !strings.Contains(patternStr, "Some(") {
		return p.parseTuplePattern(patternStr)
	}

	// 字面量模式：数字、字符串、布尔值
	if p.isLiteralPattern(patternStr) {
		return p.parseLiteralPattern(patternStr)
	}

	// 标识符模式（变量绑定）
	if p.isValidIdentifier(patternStr) {
		return entities.NewIdentifierPattern(patternStr), nil
	}

	// 其他情况，尝试作为普通表达式解析
	return p.parseExpr(patternStr)
}

// parseArrayPattern 解析数组模式，如 [x, y, ...rest]
func (p *SimpleParser) parseArrayPattern(patternStr string) (entities.Expr, error) {
	inner := strings.TrimSpace(patternStr[1 : len(patternStr)-1]) // 移除 []
	if inner == "" {
		return entities.NewArrayPattern([]entities.Expr{}, nil), nil
	}

	var elements []entities.Expr
	var rest entities.Expr

	// 检查是否有剩余元素模式 (...rest)
	if strings.Contains(inner, "...") {
		parts := strings.SplitN(inner, "...", 2)
		inner = strings.TrimSpace(parts[0])
		restPart := strings.TrimSpace(parts[1])
		if restPart != "" {
			var err error
			rest, err = p.parseMatchPattern(restPart)
			if err != nil {
				return nil, err
			}
		}
	}

	if inner != "" {
		elemStrs := strings.Split(inner, ",")
		for _, elemStr := range elemStrs {
			elemStr = strings.TrimSpace(elemStr)
			if elemStr == "" {
				continue
			}
			if elemStr == "_" {
				elements = append(elements, nil) // 通配符
			} else {
				elem, err := p.parseMatchPattern(elemStr)
				if err != nil {
					return nil, err
				}
				elements = append(elements, elem)
			}
		}
	}

	return entities.NewArrayPattern(elements, rest), nil
}

// parseStructPattern 解析结构体模式，如 Point{x: 0, y: 0}
func (p *SimpleParser) parseStructPattern(patternStr string) (entities.Expr, error) {
	// 查找类型名和{的位置
	braceIndex := strings.Index(patternStr, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("invalid struct pattern: %s", patternStr)
	}

	typeName := strings.TrimSpace(patternStr[:braceIndex])

	// 解析字段
	fieldsStr := strings.TrimSpace(patternStr[braceIndex+1:])
	if !strings.HasSuffix(fieldsStr, "}") {
		return nil, fmt.Errorf("invalid struct pattern: missing closing brace in %s", patternStr)
	}

	fieldsStr = strings.TrimSpace(fieldsStr[:len(fieldsStr)-1]) // 移除}

	var fields []entities.StructPatternField
	if fieldsStr != "" {
		fieldStrs := strings.Split(fieldsStr, ",")
		for _, fieldStr := range fieldStrs {
			fieldStr = strings.TrimSpace(fieldStr)
			if fieldStr == "" {
				continue
			}

			if !strings.Contains(fieldStr, ":") {
				return nil, fmt.Errorf("invalid struct field pattern: %s", fieldStr)
			}

			parts := strings.SplitN(fieldStr, ":", 2)
			fieldName := strings.TrimSpace(parts[0])
			patternStr := strings.TrimSpace(parts[1])

			pattern, err := p.parseMatchPattern(patternStr)
			if err != nil {
				return nil, err
			}

			fields = append(fields, entities.NewStructPatternField(fieldName, pattern))
		}
	}

	return entities.NewStructPattern(typeName, fields), nil
}

// parseTuplePattern 解析元组模式，如 (x, y, z)
func (p *SimpleParser) parseTuplePattern(patternStr string) (entities.Expr, error) {
	inner := strings.TrimSpace(patternStr[1 : len(patternStr)-1]) // 移除 ()
	if inner == "" {
		return entities.NewTuplePattern([]entities.Expr{}), nil
	}

	var elements []entities.Expr
	elemStrs := strings.Split(inner, ",")
	for _, elemStr := range elemStrs {
		elemStr = strings.TrimSpace(elemStr)
		if elemStr == "" {
			continue
		}

		elem, err := p.parseMatchPattern(elemStr)
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
	}

	return entities.NewTuplePattern(elements), nil
}

// parseLiteralPattern 解析字面量模式
func (p *SimpleParser) parseLiteralPattern(patternStr string) (entities.Expr, error) {
	// 尝试解析为整数
	if intVal, err := strconv.Atoi(patternStr); err == nil {
		return entities.NewLiteralPattern(intVal, "int"), nil
	}

	// 尝试解析为布尔值
	if patternStr == "true" {
		return entities.NewLiteralPattern(true, "bool"), nil
	}
	if patternStr == "false" {
		return entities.NewLiteralPattern(false, "bool"), nil
	}

	// 字符串字面量
	if strings.HasPrefix(patternStr, "\"") && strings.HasSuffix(patternStr, "\"") {
		strVal := patternStr[1 : len(patternStr)-1]
		return entities.NewLiteralPattern(strVal, "string"), nil
	}

	return nil, fmt.Errorf("unsupported literal pattern: %s", patternStr)
}

// isLiteralPattern 检查是否为字面量模式
func (p *SimpleParser) isLiteralPattern(s string) bool {
	// 检查是否为数字
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}

	// 检查是否为布尔值
	if s == "true" || s == "false" {
		return true
	}

	// 检查是否为字符串
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return true
	}

	return false
}

// parseAsyncFuncDef 解析异步函数定义
func (p *SimpleParser) parseAsyncFuncDef(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：async func name[T](params) -> returnType {
	// 多行版本：只解析函数头，函数体在Parse函数中填充

	funcLine := strings.TrimSpace(line[11:]) // 移除"async func "

	// 查找函数名和泛型参数
	nameEnd := strings.IndexAny(funcLine, "( [")
	if nameEnd == -1 {
		return nil, fmt.Errorf("line %d: invalid async function definition", lineNum)
	}

	funcName := strings.TrimSpace(funcLine[:nameEnd])

	// 解析泛型参数
	var typeParams []entities.GenericParam
	remaining := funcLine[nameEnd:]
	if strings.HasPrefix(remaining, "[") {
		bracketEnd := strings.Index(remaining, "]")
		if bracketEnd == -1 {
			return nil, fmt.Errorf("line %d: unclosed generic parameter brackets", lineNum)
		}
		typeParamStr := remaining[:bracketEnd+1]
		var err error
		typeParams, err = p.parseGenericParams(typeParamStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid generic parameters: %v", lineNum, err)
		}
		remaining = remaining[bracketEnd+1:]
	}

	// 解析参数
	var params []entities.Param
	if strings.HasPrefix(remaining, "(") {
		paramsEnd := strings.Index(remaining, ")")
		if paramsEnd == -1 {
			return nil, fmt.Errorf("line %d: unclosed parameter list", lineNum)
		}
		paramsStr := strings.TrimSpace(remaining[1:paramsEnd])
		remaining = remaining[paramsEnd+1:]

		if paramsStr != "" {
			paramList := strings.Split(paramsStr, ",")
			for _, paramStr := range paramList {
				paramStr = strings.TrimSpace(paramStr)
				if strings.Contains(paramStr, ":") {
					parts := strings.Split(paramStr, ":")
					if len(parts) == 2 {
						paramName := strings.TrimSpace(parts[0])
						paramType := strings.TrimSpace(parts[1])
						params = append(params, entities.Param{Name: paramName, Type: paramType})
					}
				}
			}
		}
	}

	// 解析返回类型
	returnType := "void"
	remaining = strings.TrimSpace(remaining)
	if strings.HasPrefix(remaining, "->") {
		returnPart := strings.TrimSpace(remaining[2:])
		// 移除可能的 { 符号
		returnType = strings.TrimSpace(strings.TrimSuffix(returnPart, "{"))

		// 解析类型（支持Result[int]和Option[string]）
		parsedType, err := p.parseType(returnType)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid return type: %v", lineNum, err)
		}
		returnType = parsedType
	}

	return &entities.AsyncFuncDef{
		Name:       funcName,
		TypeParams: typeParams,
		Params:     params,
		ReturnType: returnType,
		Body:       []entities.ASTNode{}, // 函数体将在多行解析中填充
	}, nil
}

// isValidIdentifier 检查是否为有效的标识符
func (p *SimpleParser) isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}

	runes := []rune(s)
	if !unicode.IsLetter(runes[0]) && runes[0] != '_' {
		return false
	}

	for _, r := range runes[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}

	// 检查是否为关键字
	keywords := []string{"func", "if", "else", "for", "while", "return", "struct", "enum", "trait", "impl", "print", "let", "match", "async", "await", "chan", "spawn", "future", "true", "false"}
	for _, keyword := range keywords {
		if s == keyword {
			return false
		}
	}

	return true
}

// parseType 解析类型注解，支持Result[int]、Option[string]、Result[T]等
func (p *SimpleParser) parseType(typeStr string) (string, error) {
	typeStr = strings.TrimSpace(typeStr)

	// Result类型: Result[int] 或 Result[T]
	if strings.HasPrefix(typeStr, "Result[") && strings.HasSuffix(typeStr, "]") {
		innerType := strings.TrimSpace(typeStr[7 : len(typeStr)-1])
		return fmt.Sprintf("Result[%s]", innerType), nil
	}

	// Option类型: Option[string] 或 Option[T]
	if strings.HasPrefix(typeStr, "Option[") && strings.HasSuffix(typeStr, "]") {
		innerType := strings.TrimSpace(typeStr[7 : len(typeStr)-1])
		return fmt.Sprintf("Option[%s]", innerType), nil
	}

	// Future类型: Future[int] 或 Future[T]
	if strings.HasPrefix(typeStr, "Future[") && strings.HasSuffix(typeStr, "]") {
		innerType := strings.TrimSpace(typeStr[7 : len(typeStr)-1])
		return fmt.Sprintf("Future[%s]", innerType), nil
	}

	// Chan类型: chan int
	if strings.HasPrefix(typeStr, "chan ") {
		elementType := strings.TrimSpace(typeStr[5:])
		return fmt.Sprintf("chan %s", elementType), nil
	}

	// 动态Trait引用: &dyn Trait
	if strings.HasPrefix(typeStr, "&dyn ") {
		traitName := strings.TrimSpace(typeStr[5:]) // 移除"&dyn "
		// 验证Trait名称格式（简化实现）
		if traitName != "" {
			return fmt.Sprintf("&dyn %s", traitName), nil
		}
	}

	// Trait泛型类型: Printable[int] 或 Printable[T]
	// 检查是否包含[和]，且前缀不是内置类型
	if strings.Contains(typeStr, "[") && strings.Contains(typeStr, "]") {
		bracketStart := strings.Index(typeStr, "[")
		bracketEnd := strings.LastIndex(typeStr, "]")
		if bracketStart > 0 && bracketEnd > bracketStart {
			typeName := strings.TrimSpace(typeStr[:bracketStart])
			typeArgsStr := strings.TrimSpace(typeStr[bracketStart+1 : bracketEnd])

			// 检查是否是已知的Trait名称（简化实现，实际应该从符号表查询）
			knownTraits := []string{"Printable", "Serializable", "Comparable"}
			isTrait := false
			for _, trait := range knownTraits {
				if typeName == trait {
					isTrait = true
					break
				}
			}

			if isTrait {
				return fmt.Sprintf("%s[%s]", typeName, typeArgsStr), nil
			}
		}
	}

	// 基本类型
	return typeStr, nil
}

// parseStructLiteral 解析结构体字面量，如 User{name: "Alice", age: 30}
func (p *SimpleParser) parseStructLiteral(expr string) (entities.Expr, error) {
	expr = strings.TrimSpace(expr)

	// 查找类型名和{的位置
	braceIndex := strings.Index(expr, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("invalid struct literal: %s", expr)
	}

	typeName := strings.TrimSpace(expr[:braceIndex])

	// 解析字段
	fieldsStr := strings.TrimSpace(expr[braceIndex+1:])
	if !strings.HasSuffix(fieldsStr, "}") {
		return nil, fmt.Errorf("invalid struct literal: missing closing brace in %s", expr)
	}

	fieldsStr = strings.TrimSpace(fieldsStr[:len(fieldsStr)-1]) // 移除}

	fields := make(map[string]entities.Expr)

	if fieldsStr != "" {
		fieldPairs := strings.Split(fieldsStr, ",")
		for _, pair := range fieldPairs {
			pair = strings.TrimSpace(pair)
			if strings.Contains(pair, ":") {
				parts := strings.SplitN(pair, ":", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid field syntax in struct literal: %s", pair)
				}

				fieldName := strings.TrimSpace(parts[0])
				fieldValueStr := strings.TrimSpace(parts[1])

				fieldValue, err := p.parseExpr(fieldValueStr)
				if err != nil {
					return nil, fmt.Errorf("invalid field value in struct literal: %v", err)
				}

				fields[fieldName] = fieldValue
			}
		}
	}

	return &entities.StructLiteral{
		Type:   typeName,
		Fields: fields,
	}, nil
}

// parseFuncCallExpr 解析函数调用表达式
func (p *SimpleParser) parseFuncCallExpr(expr string) (entities.Expr, error) {
	// 支持语法：funcName(args) 或 funcName[T](args) 或 funcName[T, U](args)

	expr = strings.TrimSpace(expr)

	// 查找函数名
	nameEnd := strings.IndexAny(expr, "[(")
	if nameEnd == -1 {
		return nil, fmt.Errorf("invalid function call expression: %s", expr)
	}

	funcName := strings.TrimSpace(expr[:nameEnd])

	// 解析类型参数
	var typeArgs []string
	remaining := expr[nameEnd:]

	if strings.HasPrefix(remaining, "[") {
		// 有类型参数
		closeIndex := strings.Index(remaining, "]")
		if closeIndex == -1 {
			return nil, fmt.Errorf("unclosed type parameter brackets in: %s", expr)
		}

		typeArgStr := remaining[1:closeIndex]
		if typeArgStr != "" {
			typeArgs = strings.Split(typeArgStr, ",")
			for i, arg := range typeArgs {
				typeArgs[i] = strings.TrimSpace(arg)
			}
		}

		remaining = remaining[closeIndex+1:]
	}

	// 解析参数
	if !strings.HasPrefix(remaining, "(") {
		return nil, fmt.Errorf("function call missing parentheses in: %s", expr)
	}

	closeParen := strings.Index(remaining, ")")
	if closeParen == -1 {
		return nil, fmt.Errorf("unclosed function call parentheses in: %s", expr)
	}

	argStr := strings.TrimSpace(remaining[1:closeParen])

	var args []entities.Expr
	if argStr != "" {
		// 简单参数解析（目前只支持字面量和标识符）
		argParts := strings.Split(argStr, ",")
		for _, argPart := range argParts {
			argPart = strings.TrimSpace(argPart)
			if argPart != "" {
				argExpr, err := p.parseExpr(argPart)
				if err != nil {
					return nil, fmt.Errorf("invalid argument %s: %v", argPart, err)
				}
				args = append(args, argExpr)
			}
		}
	}

	return &entities.FuncCall{
		Name:     funcName,
		TypeArgs: typeArgs,
		Args:     args,
	}, nil
}

// parseBinaryExpr 解析二元表达式

// parseTraitDef 解析完整的trait定义
func (p *SimpleParser) parseTraitDef(line string, lineNum int, lines []string) (entities.ASTNode, int, error) {
	// 支持语法：trait Name { methods }
	// 解析整个trait定义，包括方法

	traitLine := strings.TrimSpace(line[6:]) // 移除"trait "

	// 解析trait名、泛型参数和继承关系
	nameEnd := strings.IndexAny(traitLine, "{ [:")
	if nameEnd == -1 {
		return nil, 0, fmt.Errorf("line %d: invalid trait definition", lineNum)
	}

	name := strings.TrimSpace(traitLine[:nameEnd])

	// 解析泛型参数和继承关系
	var typeParams []entities.GenericParam
	var superTraits []string
	remainingLine := traitLine[nameEnd:]

	var braceIndex int
	if strings.HasPrefix(remainingLine, "[") {
		// 有泛型参数
		braceIndex = strings.Index(remainingLine, "{")
		if braceIndex == -1 {
			return nil, 0, fmt.Errorf("line %d: trait definition must have opening brace", lineNum)
		}

		typeParamStr := remainingLine[:braceIndex]
		var err error
		typeParams, err = p.parseGenericParams(typeParamStr)
		if err != nil {
			return nil, 0, fmt.Errorf("line %d: invalid generic parameters: %v", lineNum, err)
		}

		remainingLine = remainingLine[braceIndex:]
		braceIndex = 0 // 重置，因为remainingLine已经改变了
	} else if strings.HasPrefix(remainingLine, ":") {
		// 有继承关系
		braceIndex = strings.Index(remainingLine, "{")
		if braceIndex == -1 {
			return nil, 0, fmt.Errorf("line %d: trait definition must have opening brace", lineNum)
		}

		inheritanceStr := strings.TrimSpace(remainingLine[1:braceIndex]) // 移除":"
		if inheritanceStr != "" {
			// 解析逗号分隔的继承trait
			superTraitNames := strings.Split(inheritanceStr, ",")
			for _, traitName := range superTraitNames {
				traitName = strings.TrimSpace(traitName)
				if traitName != "" {
					superTraits = append(superTraits, traitName)
				}
			}
		}

		remainingLine = remainingLine[braceIndex:]
		braceIndex = 0 // 重置，因为remainingLine已经改变了
	} else {
		remainingLine = traitLine[nameEnd:]
		braceIndex = strings.Index(remainingLine, "{")
		if braceIndex == -1 {
			return nil, 0, fmt.Errorf("line %d: trait definition must have opening brace", lineNum)
		}
	}

	// 检查是否是单行trait
	remaining := strings.TrimSpace(remainingLine[braceIndex+1:])
	if strings.HasPrefix(remaining, "{") && strings.HasSuffix(remaining, "}") && strings.Count(remaining, "{") == 1 {
		// 单行trait - 解析body内容
		bodyContent := strings.TrimSpace(remaining[1 : len(remaining)-1]) // 移除{和}

		// 解析关联类型和方法
		var associatedTypes []entities.AssociatedType
		var methods []entities.TraitMethod
		if bodyContent != "" {
			// 先按分号分割声明
			declarations := strings.Split(bodyContent, ";")
			for _, decl := range declarations {
				decl = strings.TrimSpace(decl)
				if decl == "" {
					continue
				}

				if strings.HasPrefix(decl, "type ") && !strings.Contains(decl, "(") {
					// 关联类型声明：type Item
					typeName := strings.TrimSpace(decl[5:]) // 移除"type "
					associatedTypes = append(associatedTypes, entities.AssociatedType{Name: typeName})
				} else if strings.HasPrefix(decl, "func ") {
					// 方法声明
					method, err := p.parseTraitMethod(decl, lineNum)
					if err != nil {
						return nil, 0, fmt.Errorf("trait %s method error: %v", name, err)
					}
					methods = append(methods, *method)
				} else {
					// 调试：打印无法识别的内容
				}
			}
		}

		return &entities.TraitDef{
			Name:            name,
			TypeParams:      typeParams,
			AssociatedTypes: associatedTypes,
			Methods:         methods,
		}, 1, nil // 只消耗了一行
	}

	// 多行trait，收集所有行直到}
	bodyLines := []string{}
	consumedLines := 1
	currentIndex := lineNum // 从下一行开始

	for consumedLines < 1000 {
		if currentIndex >= len(lines) {
			return nil, consumedLines, fmt.Errorf("trait %s definition not closed", name)
		}

		currentLine := strings.TrimSpace(lines[currentIndex])
		currentIndex++
		consumedLines++

		if currentLine == "}" {
			break // 找到结束
		}

		if currentLine != "" {
			bodyLines = append(bodyLines, currentLine)
		}
	}

	// 解析关联类型和方法
	var associatedTypes []entities.AssociatedType
	var methods []entities.TraitMethod

	// 单行trait的情况
	if remainingLine == "{}" || remainingLine == "{ }" {
		return &entities.TraitDef{
			Name:            name,
			TypeParams:      typeParams,
			AssociatedTypes: associatedTypes,
			Methods:         methods,
		}, 1, nil // 只消耗了一行
	}
	// 处理每一行，允许一行包含多个声明（用分号分隔）
	for _, traitLine := range bodyLines {
		traitLine = strings.TrimSpace(traitLine)

		if traitLine == "" {
			continue
		}

		// 如果一行包含分号，分割成多个声明
		declarations := strings.Split(traitLine, ";")
		for _, decl := range declarations {
			decl = strings.TrimSpace(decl)
			if decl == "" {
				continue
			}

			if strings.HasPrefix(decl, "type ") && !strings.Contains(decl, "(") {
				// 关联类型声明：type Item
				typeName := strings.TrimSpace(decl[5:]) // 移除"type "
				associatedTypes = append(associatedTypes, entities.AssociatedType{Name: typeName})
			} else if strings.HasPrefix(decl, "func ") {
				// 方法声明
				method, err := p.parseTraitMethod(decl, lineNum)
				if err != nil {
					return nil, consumedLines, fmt.Errorf("trait %s method error: %v", name, err)
				}
				methods = append(methods, *method)
			}
		}
	}

	return &entities.TraitDef{
		Name:            name,
		TypeParams:      typeParams,
		SuperTraits:     superTraits,
		AssociatedTypes: associatedTypes,
		Methods:         methods,
	}, consumedLines, nil
}

// parseTraitMethod 解析trait方法声明
func (p *SimpleParser) parseTraitMethod(methodStr string, lineNum int) (*entities.TraitMethod, error) {
	// 支持语法：
	// func methodName(params) -> ReturnType
	// func methodName[T](params) -> ReturnType  // 方法级泛型参数
	// func methodName[T, U](params) -> ReturnType

	methodStr = strings.TrimSpace(methodStr)
	if !strings.HasPrefix(methodStr, "func ") {
		return nil, fmt.Errorf("trait method must start with 'func'")
	}

	methodStr = methodStr[5:] // 移除"func "

	// 解析方法名和可能的泛型参数
	// 查找方法名结束位置（遇到(、[或空格）
	methodNameEnd := strings.IndexAny(methodStr, "([ ")
	if methodNameEnd == -1 {
		return nil, fmt.Errorf("invalid method name")
	}

	methodName := strings.TrimSpace(methodStr[:methodNameEnd])

	// 解析方法类型参数（如 methodName[T, U]）
	var methodTypeParams []entities.GenericParam
	remainingAfterName := strings.TrimSpace(methodStr[methodNameEnd:])
	if strings.HasPrefix(remainingAfterName, "[") {
		bracketEnd := strings.Index(remainingAfterName, "]")
		if bracketEnd == -1 {
			return nil, fmt.Errorf("unclosed method type parameter brackets")
		}
		typeParamStr := remainingAfterName[:bracketEnd+1]
		remainingAfterName = strings.TrimSpace(remainingAfterName[bracketEnd+1:])

		var err error
		methodTypeParams, err = p.parseGenericParams(typeParamStr)
		if err != nil {
			return nil, fmt.Errorf("invalid method type parameters: %v", err)
		}
	}

	// 解析参数
	parenIndex := strings.Index(remainingAfterName, "(")
	if parenIndex == -1 {
		return nil, fmt.Errorf("invalid method signature")
	}

	// 解析参数
	paramsEnd := strings.Index(methodStr, ")")
	if paramsEnd == -1 || paramsEnd < parenIndex {
		return nil, fmt.Errorf("invalid method parameters")
	}

	paramsStr := strings.TrimSpace(methodStr[parenIndex+1 : paramsEnd])
	var params []entities.Param
	if paramsStr != "" {
		paramParts := strings.Split(paramsStr, ",")
		for _, paramPart := range paramParts {
			paramPart = strings.TrimSpace(paramPart)
			if strings.Contains(paramPart, ":") {
				parts := strings.Split(paramPart, ":")
				if len(parts) == 2 {
					params = append(params, entities.Param{
						Name: strings.TrimSpace(parts[0]),
						Type: strings.TrimSpace(parts[1]),
					})
				}
			}
		}
	}

	// 检查是否有默认实现（方法体）
	remaining := strings.TrimSpace(methodStr[paramsEnd+1:])

	// 解析返回类型
	returnType := "void"
	body := []entities.ASTNode{}
	isDefault := false

	if strings.Contains(remaining, "{") {
		// 有默认实现
		isDefault = true

		// 解析返回类型（如果有->）
		braceIndex := strings.Index(remaining, "{")
		beforeBrace := strings.TrimSpace(remaining[:braceIndex])
		if strings.HasPrefix(beforeBrace, "->") {
			returnType = strings.TrimSpace(beforeBrace[2:])
		}

		// 解析方法体
		bodyStr := strings.TrimSpace(remaining[braceIndex:])
		if strings.HasPrefix(bodyStr, "{") && strings.HasSuffix(bodyStr, "}") {
			bodyContent := strings.TrimSpace(bodyStr[1 : len(bodyStr)-1])
			if bodyContent != "" {
				// 简单解析body（暂时只支持单行语句）
				lines := strings.Split(bodyContent, ";")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					// 简单处理：只支持return语句
					if strings.HasPrefix(line, "return ") {
						valueStr := strings.TrimSpace(line[7:])
						if valueStr != "" {
							expr, err := p.parseExpr(valueStr)
							if err != nil {
								return nil, fmt.Errorf("invalid return expression in trait method: %v", err)
							}
							body = append(body, &entities.ReturnStmt{Value: expr})
						} else {
							body = append(body, &entities.ReturnStmt{})
						}
					}
				}
			}
		}
	} else {
		// 没有默认实现，只解析返回类型
		if strings.HasPrefix(remaining, "->") {
			returnType = strings.TrimSpace(remaining[2:])
		}
	}

	return &entities.TraitMethod{
		Name:       methodName,
		TypeParams: methodTypeParams,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
		IsDefault:  isDefault,
	}, nil
}

// parseArrayLiteral 解析数组字面量
func (p *SimpleParser) parseArrayLiteral(expr string) (entities.Expr, error) {
	// 移除 [ 和 ]
	content := strings.TrimSpace(expr[1 : len(expr)-1])

	var elements []entities.Expr
	if content != "" {
		// 按逗号分割元素
		elementStrs := strings.Split(content, ",")
		for _, elementStr := range elementStrs {
			elementStr = strings.TrimSpace(elementStr)
			if elementStr != "" {
				element, err := p.parseExpr(elementStr)
				if err != nil {
					return nil, fmt.Errorf("invalid array element: %v", err)
				}
				elements = append(elements, element)
			}
		}
	}

	return &entities.ArrayLiteral{Elements: elements}, nil
}

// parseMatchExpr 解析match表达式（单行版本，用于表达式上下文）
func (p *SimpleParser) parseMatchExpr(expr string) (entities.Expr, error) {
	// 支持语法：match value { pattern1 => expr1, pattern2 => expr2 }
	// 单行版本：只支持简单表达式

	matchLine := strings.TrimSpace(expr[6:]) // 移除"match "

	// 找到值和 { 的分界点
	braceIndex := strings.Index(matchLine, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("match expression must have opening brace")
	}

	valueStr := strings.TrimSpace(matchLine[:braceIndex])
	value, err := p.parseExpr(valueStr)
	if err != nil {
		return nil, fmt.Errorf("invalid match value: %v", err)
	}

	// 解析case分支（单行版本）
	casesStr := strings.TrimSpace(matchLine[braceIndex+1:])
	casesStr = strings.TrimSuffix(casesStr, "}")

	var cases []entities.MatchCase
	if casesStr != "" {
		caseParts := strings.Split(casesStr, ",")
		for _, casePart := range caseParts {
			casePart = strings.TrimSpace(casePart)
			if strings.Contains(casePart, "=>") {
				parts := strings.SplitN(casePart, "=>", 2)
				if len(parts) == 2 {
					pattern, err := p.parseExpr(strings.TrimSpace(parts[0]))
					if err != nil {
						return nil, fmt.Errorf("invalid match pattern: %v", err)
					}

					action, err := p.parseExpr(strings.TrimSpace(parts[1]))
					if err != nil {
						return nil, fmt.Errorf("invalid match action: %v", err)
					}

					cases = append(cases, entities.MatchCase{
						Pattern: pattern,
						Body:    []entities.ASTNode{&entities.ReturnStmt{Value: action}}, // 简单版本，直接返回action
					})
				}
			}
		}
	}

	return &entities.MatchExpr{
		Value: value,
		Cases: cases,
	}, nil
}

// parseMatchStmt 解析match语句（用于语句上下文）
func (p *SimpleParser) parseMatchStmt(line string, lineNum int) (entities.ASTNode, error) {
	// 支持语法：match value { ... }
	// 多行版本：在parseBlock中填充case分支

	matchLine := strings.TrimSpace(line[6:]) // 移除"match "

	// 找到值和 { 的分界点
	braceIndex := strings.Index(matchLine, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: match statement must have opening brace", lineNum)
	}

	valueStr := strings.TrimSpace(matchLine[:braceIndex])
	value, err := p.parseExpr(valueStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid match value: %v", lineNum, err)
	}

	return &entities.MatchStmt{
		Value: value,
		Cases: []entities.MatchCase{}, // case分支将在多行解析中填充
	}, nil
}

// parseMatchExprStart 解析match表达式的开始部分（多行版本）
func (p *SimpleParser) parseMatchExprStart(line string, lineNum int) (entities.Expr, error) {
	// 支持语法：match value {
	// 多行版本：只解析值部分，case分支在Parse函数中处理

	matchLine := strings.TrimSpace(line[6:]) // 移除"match "

	// 找到值和 { 的分界点
	braceIndex := strings.Index(matchLine, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: match expression must have opening brace", lineNum)
	}

	valueStr := strings.TrimSpace(matchLine[:braceIndex])
	value, err := p.parseExpr(valueStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid match value: %v", lineNum, err)
	}

	return &entities.MatchExpr{
		Value: value,
		Cases: []entities.MatchCase{}, // case分支将在多行解析中填充
	}, nil
}

// parseBlock 解析一个代码块（由语句行组成，不包含结束括号）
func (p *SimpleParser) parseBlock(lines []string, startLineNum int) ([]entities.ASTNode, int, error) {
	var statements []entities.ASTNode
	currentLineNum := startLineNum
	i := 0

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "//") {
			i++
			currentLineNum++
			continue
		}

		// 解析单个语句
		stmt, err := p.parseStatement(line, currentLineNum)
		if err != nil {
			// 检查是否是match case行，如果是则跳过
			if strings.Contains(line, "=>") && !strings.HasPrefix(line, "match ") {
				i++
				currentLineNum++
				continue
			}
			return nil, currentLineNum, fmt.Errorf("line %d: %v", currentLineNum, err)
		}

		// 特殊处理：如果遇到多行语句的开始，需要特殊处理
		if stmt != nil {
			if matchStmt, ok := stmt.(*entities.MatchStmt); ok && len(matchStmt.Cases) == 0 {
				// 这是一个match语句的开始，需要解析多行case
				matchCases, consumedLines, err := p.parseMatchCases(lines[i+1:], currentLineNum+1)
				if err != nil {
					return nil, currentLineNum, fmt.Errorf("line %d: match cases error: %v", currentLineNum, err)
				}
				matchStmt.Cases = matchCases
				statements = append(statements, stmt)
				// 跳过已处理的行（case行 + }行）
				i += consumedLines + 1 // +1 for the closing }
				currentLineNum += consumedLines + 1
				continue
			}
			if ifStmt, ok := stmt.(*entities.IfStmt); ok && len(ifStmt.ThenBody) == 0 {
				// 这是一个if语句的开始，需要解析then分支
				thenLines, consumedLines, hasElse := p.extractIfBlock(lines[i+1:], currentLineNum+1)
				if consumedLines == 0 {
					return nil, currentLineNum, fmt.Errorf("line %d: incomplete if statement", currentLineNum)
				}

				// 解析then分支
				thenBody, _, err := p.parseBlock(thenLines, currentLineNum+1)
				if err != nil {
					return nil, currentLineNum, fmt.Errorf("line %d: if then body error: %v", currentLineNum, err)
				}
				ifStmt.ThenBody = thenBody

				if hasElse {
					// 有else分支，检查是否是else if
					nextLineIndex := i + 1 + consumedLines
					if nextLineIndex < len(lines) {
						nextLine := strings.TrimSpace(lines[nextLineIndex])
						if strings.Contains(nextLine, "} else if ") {
							// 处理 } else if condition { 语法
							elseIfLine := strings.TrimSpace(nextLine[1:]) // 移除开头的 }
							elseIfStmt, err := p.parseElseIfStmt(elseIfLine, currentLineNum+1+consumedLines)
							if err != nil {
								return nil, currentLineNum, fmt.Errorf("line %d: invalid else if statement: %v", currentLineNum+1+consumedLines, err)
							}

							// 将 else if 作为嵌套的 if 语句放入 else 分支
							nestedIfStmt := elseIfStmt.(*entities.IfStmt)
							ifStmt.ElseBody = []entities.ASTNode{nestedIfStmt}

							// 递归处理嵌套的if语句
							nestedConsumed, err := p.processNestedIf(&nestedIfStmt, lines[nextLineIndex+1:], currentLineNum+1+consumedLines+1)
							if err != nil {
								return nil, currentLineNum, err
							}
							consumedLines += 1 + nestedConsumed // +1 for the } else if line
						} else {
							// 普通的else分支
							elseLines, elseConsumed, _ := p.extractElseBlock(lines[i+1+consumedLines:], currentLineNum+1+consumedLines)
							if len(elseLines) > 0 {
								elseBody, _, err := p.parseBlock(elseLines, currentLineNum+1+consumedLines)
								if err != nil {
									return nil, currentLineNum, fmt.Errorf("line %d: if else body error: %v", currentLineNum, err)
								}
								ifStmt.ElseBody = elseBody
								consumedLines += elseConsumed
							}
						}
					}
				}

				statements = append(statements, stmt)
				// 跳过已处理的行
				i += consumedLines
				currentLineNum += consumedLines
				continue
			}
		}

		if stmt != nil {
			statements = append(statements, stmt)
		}

		i++
		currentLineNum++
	}

	return statements, currentLineNum, nil
}

func (p *SimpleParser) parseBinaryExpr(expr string) (entities.Expr, error) {
	// 支持多种算术运算符，所有操作符同优先级，从左到右结合
	operators := []string{"+", "-", "*", "/", "%"}

	// 查找所有操作符位置

	// 查找所有操作符位置
	var positions []struct {
		index int
		op    string
	}

	for _, op := range operators {
		opStr := " " + op + " "
		idx := 0
		for {
			pos := strings.Index(expr[idx:], opStr)
			if pos == -1 {
				break
			}
			actualIdx := idx + pos
			positions = append(positions, struct {
				index int
				op    string
			}{actualIdx, op})
			idx = actualIdx + len(opStr)
		}
	}

	// 如果没有操作符，返回错误
	if len(positions) == 0 {
		return nil, fmt.Errorf("no binary operator found")
	}

	// 对位置排序（从左到右）
	// 这里我们简化处理，只处理两个操作数的情况
	// 对于复杂的表达式，我们使用递归的方法

	// 找到最右边的操作符（实现左结合）
	rightmost := positions[0]
	for _, pos := range positions[1:] {
		if pos.index > rightmost.index {
			rightmost = pos
		}
	}

	leftPart := strings.TrimSpace(expr[:rightmost.index])
	rightPart := strings.TrimSpace(expr[rightmost.index+len(" "+rightmost.op+" "):])

	// 递归解析左侧
	left, err := p.parseExpr(leftPart)
	if err != nil {
		return nil, err
	}

	// 递归解析右侧
	right, err := p.parseExpr(rightPart)
	if err != nil {
		return nil, err
	}

	return &entities.BinaryExpr{
		Left:  left,
		Op:    rightmost.op,
		Right: right,
	}, nil
}

// extractIfBlock 提取if语句的then分支内容
func (p *SimpleParser) extractIfBlock(lines []string, startLineNum int) ([]string, int, bool) {
	var blockLines []string
	braceCount := 0
	hasElse := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if line == "{" {
			braceCount++
			if braceCount > 1 {
				blockLines = append(blockLines, lines[i])
			}
		} else if line == "}" {
			braceCount--
			if braceCount == 0 {
				// then分支结束
				break
			} else {
				blockLines = append(blockLines, lines[i])
			}
		} else if braceCount > 0 {
			blockLines = append(blockLines, lines[i])
		} else if strings.HasPrefix(line, "} else") {
			// 处理 } else 或 } else if 情况
			hasElse = true
			break
		} else if strings.HasPrefix(line, "else") {
			hasElse = true
			break
		}
	}

	return blockLines, len(blockLines) + 1, hasElse // +1 for the closing }
}

// processNestedIf 处理嵌套的if语句
func (p *SimpleParser) processNestedIf(ifStmt **entities.IfStmt, lines []string, startLineNum int) (int, error) {
	// 使用与顶层if语句相同的处理逻辑
	currentIf := *ifStmt
	nestedConsumed := 0

	// 提取then分支
	thenLines, consumed, hasElse := p.extractIfBlock(lines, startLineNum)
	if consumed == 0 {
		return 0, fmt.Errorf("line %d: incomplete nested if statement", startLineNum)
	}

	// 解析then分支
	thenBody, _, err := p.parseBlock(thenLines, startLineNum)
	if err != nil {
		return 0, fmt.Errorf("line %d: nested if then body error: %v", startLineNum, err)
	}
	currentIf.ThenBody = thenBody
	nestedConsumed += consumed

	if hasElse {
		// 处理else分支，这里可能包含嵌套的if语句
		elseLines, elseConsumed, _ := p.extractElseBlock(lines[consumed:], startLineNum+consumed)
		if len(elseLines) > 0 {
			elseBody, _, err := p.parseBlock(elseLines, startLineNum+consumed)
			if err != nil {
				return 0, fmt.Errorf("line %d: nested if else body error: %v", startLineNum+consumed, err)
			}
			currentIf.ElseBody = elseBody
			nestedConsumed += elseConsumed
		}
	}

	return nestedConsumed, nil
}

// extractElseBlock 提取else分支内容
func (p *SimpleParser) extractElseBlock(lines []string, startLineNum int) ([]string, int, bool) {
	var blockLines []string
	braceCount := 0
	started := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "else") || strings.HasPrefix(line, "} else") {
			started = true
			if strings.Contains(line, "{") {
				braceCount++
			}
		} else if started {
			if line == "{" {
				braceCount++
			} else if line == "}" {
				braceCount--
				if braceCount == 0 {
					break
				}
			} else if braceCount > 0 {
				blockLines = append(blockLines, lines[i])
			}
		}
	}

	return blockLines, len(blockLines) + 2, false // +2 for "else {" and closing "}"
}

// parseComparisonExpr 解析比较表达式
func (p *SimpleParser) parseComparisonExpr(expr string) (entities.Expr, error) {
	// 支持的比较运算符，按优先级从长到短检查
	operators := []string{" >= ", " <= ", " == ", " != ", " > ", " < "}

	for _, op := range operators {
		if strings.Contains(expr, op) {
			parts := strings.Split(expr, op)
			if len(parts) == 2 {
				left, err := p.parseExpr(strings.TrimSpace(parts[0]))
				if err != nil {
					return nil, err
				}
				right, err := p.parseExpr(strings.TrimSpace(parts[1]))
				if err != nil {
					return nil, err
				}
				return &entities.BinaryExpr{
					Left:  left,
					Op:    strings.TrimSpace(op),
					Right: right,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("unsupported comparison expression: %s", expr)
}

// parseMatchCases 解析多行的match case分支
func (p *SimpleParser) parseMatchCases(lines []string, startLineNum int) ([]entities.MatchCase, int, error) {
	var cases []entities.MatchCase
	consumedLines := 0

	for lineIndex, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			consumedLines++
			continue
		}

		if line == "}" {
			// 匹配结束，消耗这一行
			consumedLines++
			break
		}

		if strings.Contains(line, "=>") {
			parts := strings.SplitN(line, "=>", 2)
			if len(parts) == 2 {
				patternStr := strings.TrimSpace(parts[0])
				actionStr := strings.TrimSpace(parts[1])

				// 检查是否有守卫条件 (if guard)
				var guard entities.Expr
				if strings.Contains(patternStr, " if ") {
					patternParts := strings.SplitN(patternStr, " if ", 2)
					patternStr = strings.TrimSpace(patternParts[0])
					guardStr := strings.TrimSpace(patternParts[1])

					var err error
					guard, err = p.parseExpr(guardStr)
					if err != nil {
						return nil, consumedLines, fmt.Errorf("line %d: invalid guard condition: %v", startLineNum+lineIndex, err)
					}
				}

				// 解析模式
				pattern, err := p.parseMatchPattern(patternStr)
				if err != nil {
					return nil, consumedLines, fmt.Errorf("line %d: invalid match pattern: %v", startLineNum+lineIndex, err)
				}

				var body []entities.ASTNode

				// 移除末尾逗号（如果有）
				actionStr = strings.TrimSuffix(actionStr, ",")
				actionStr = strings.TrimSpace(actionStr)

				// 处理不同的动作类型
				if strings.HasPrefix(actionStr, "print ") {
					// print语句
					valueStr := strings.TrimSpace(actionStr[6:])
					if valueStr == "" {
						return nil, consumedLines, fmt.Errorf("line %d: print statement requires an expression in match case", startLineNum+lineIndex)
					}
					expr, err := p.parseExpr(valueStr)
					if err != nil {
						return nil, consumedLines, fmt.Errorf("line %d: invalid print expression in match case: %v", startLineNum+lineIndex, err)
					}
					body = []entities.ASTNode{&entities.PrintStmt{Value: expr}}
				} else if strings.HasPrefix(actionStr, "return ") {
					// return语句
					valueStr := strings.TrimSpace(actionStr[7:])
					action, err := p.parseExpr(valueStr)
					if err != nil {
						return nil, consumedLines, fmt.Errorf("line %d: invalid return expression in match case: %v", startLineNum+lineIndex, err)
					}
					body = []entities.ASTNode{&entities.ReturnStmt{Value: action}}
				} else if len(actionStr) >= 2 && actionStr[0] == '"' && actionStr[len(actionStr)-1] == '"' {
					// 字符串字面量 - 转换为return语句
					strValue := actionStr[1 : len(actionStr)-1]
					stringLiteral := &entities.StringLiteral{Value: strValue}
					body = []entities.ASTNode{&entities.ReturnStmt{Value: stringLiteral}}
				} else {
					// 首先尝试作为表达式解析
					expr, err := p.parseExpr(actionStr)
					if err == nil && expr != nil {
						// 表达式解析成功，包装为return语句
						body = []entities.ASTNode{&entities.ReturnStmt{Value: expr}}
					} else {
						// 表达式解析失败，尝试作为语句解析
						stmt, err := p.parseStatement(actionStr, startLineNum+lineIndex)
						if err != nil {
							return nil, consumedLines, fmt.Errorf("line %d: invalid match action: %v", startLineNum+lineIndex, err)
						}
						if stmt != nil {
							body = []entities.ASTNode{stmt}
						} else {
							return nil, consumedLines, fmt.Errorf("line %d: empty match action", startLineNum+lineIndex)
						}
					}
				}

				cases = append(cases, entities.MatchCase{
					Pattern: pattern,
					Guard:   guard,
					Body:    body,
				})
			}
		}

		consumedLines++
	}

	return cases, consumedLines, nil
}

// parseCaseBody 解析case的多行body
func (p *SimpleParser) parseCaseBody(lines []string, startLineNum int) ([]string, int, error) {
	var bodyLines []string
	consumedLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			consumedLines++
			continue
		}

		if strings.Contains(line, "=>") || line == "}" {
			// 下一个case或结束
			break
		}

		bodyLines = append(bodyLines, line)
		consumedLines++
	}

	return bodyLines, consumedLines, nil
}

// isFunctionCall 检查是否为函数调用
func (p *SimpleParser) isFunctionCall(line string) bool {
	line = strings.TrimSpace(line)
	if strings.Contains(line, "(") && strings.Contains(line, ")") {
		// 检查括号前的部分是否为有效的函数名（可能包含泛型参数）
		beforeParen := strings.Split(line, "(")[0]
		beforeParen = strings.TrimSpace(beforeParen)

		// 处理泛型函数调用，如 printValue[int]
		if strings.Contains(beforeParen, "[") && strings.HasSuffix(beforeParen, "]") {
			// 提取函数名部分
			bracketIndex := strings.Index(beforeParen, "[")
			funcName := strings.TrimSpace(beforeParen[:bracketIndex])
			return p.isIdentifier(funcName)
		} else {
			// 普通函数调用
			return p.isIdentifier(beforeParen)
		}
	}
	return false
}

// isMethodCall 检查是否为方法调用语句
func (p *SimpleParser) isMethodCall(line string) bool {
	line = strings.TrimSpace(line)
	// 检查是否有 . 和 () 且 . 在 ( 之前
	if strings.Contains(line, ".") && strings.Contains(line, "(") && strings.Contains(line, ")") {
		dotIndex := strings.Index(line, ".")
		parenIndex := strings.Index(line, "(")
		return dotIndex < parenIndex && dotIndex > 0
	}
	return false
}

// parseMethodCall 解析方法调用语句
func (p *SimpleParser) parseMethodCall(line string, lineNum int) (entities.ASTNode, error) {
	line = strings.TrimSpace(line)

	// 解析为方法调用表达式，然后包装为表达式语句
	methodCallExpr, err := p.parseMethodCallExpr(line)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid method call: %v", lineNum, err)
	}

	return &entities.ExprStmt{
		Expression: methodCallExpr,
	}, nil
}

// parseReturnStmt 解析return语句
func (p *SimpleParser) parseReturnStmt(line string, lineNum int) (entities.ASTNode, error) {
	line = strings.TrimSpace(line)

	// 移除 "return " 前缀
	exprStr := strings.TrimSpace(line[6:]) // len("return") = 6

	var value entities.Expr
	if exprStr != "" {
		expr, err := p.parseExpr(exprStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: %v", lineNum, err)
		}
		value = expr
	}

	return &entities.ReturnStmt{
		Value: value,
	}, nil
}

// isIdentifier 检查是否为有效标识符
func (p *SimpleParser) isIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] >= '0' && s[0] <= '9' {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

// isAsyncExpression 检查是否为异步表达式语句
func (p *SimpleParser) isAsyncExpression(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "await ") ||
		strings.HasPrefix(line, "spawn ") ||
		strings.HasPrefix(line, "chan ") ||
		strings.Contains(line, " <- ") ||
		strings.HasPrefix(line, "<- ")
}

// parseAsyncStatement 解析异步表达式语句
func (p *SimpleParser) parseAsyncStatement(line string, lineNum int) (entities.ASTNode, error) {
	line = strings.TrimSpace(line)

	// 尝试解析为表达式，然后包装为ExprStmt
	expr, err := p.parseExpr(line)
	if err != nil {
		return nil, err
	}

	return &entities.ExprStmt{Expression: expr}, nil
}
