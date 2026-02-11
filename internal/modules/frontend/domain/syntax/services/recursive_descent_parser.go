// Package services 定义语法分析上下文的领域服务
package services

import (
	"context"
	"fmt"
	"strings"

	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// ExpressionParserDelegate 表达式解析委托，由 ParserCoordinator 等实现，用于在解析表达式语句时委托给 Pratt/TokenBased 解析器
type ExpressionParserDelegate interface {
	ParseExpression(ctx context.Context) (sharedVO.ASTNode, error)
}

// RecursiveDescentParser 递归下降解析器
// 实现文档中描述的递归下降解析算法，处理顶层结构
type RecursiveDescentParser struct {
	// Token流
	tokenStream *lexicalVO.EnhancedTokenStream
	position    int
	prevToken *lexicalVO.EnhancedToken // 前一个token（完善实现）

	// 表达式解析委托，非 nil 时 parseExpression 委托给其 ParseExpression，否则走桩实现
	expressionDelegate ExpressionParserDelegate

	// 错误收集
	errors []*sharedVO.ParseError

	// 解析上下文
	sourceFile *lexicalVO.SourceFile

	// 当前解析的 context，从 ParseProgram 传入，委托 ParseExpression 时使用
	parseCtx context.Context
}

// NewRecursiveDescentParser 创建新的递归下降解析器
func NewRecursiveDescentParser() *RecursiveDescentParser {
	return &RecursiveDescentParser{
		position:   0,
		prevToken:  nil, // 初始化为nil，表示没有前一个token
		errors:     make([]*sharedVO.ParseError, 0),
	}
}

// SetExpressionDelegate 设置表达式解析委托，委托与 RecursiveDescent 共用同一 TokenStream，从当前位解析
func (rdp *RecursiveDescentParser) SetExpressionDelegate(delegate ExpressionParserDelegate) {
	rdp.expressionDelegate = delegate
}

// ParseProgram 解析整个程序（递归下降的入口点）
func (rdp *RecursiveDescentParser) ParseProgram(ctx context.Context, tokenStream *lexicalVO.EnhancedTokenStream) (*sharedVO.ProgramAST, error) {
	rdp.tokenStream = tokenStream
	rdp.position = 0
	rdp.prevToken = nil // 重置前一个token
	rdp.errors = make([]*sharedVO.ParseError, 0)
	rdp.sourceFile = tokenStream.SourceFile()
	rdp.parseCtx = ctx

	// 初始化程序AST
	sourceFile := tokenStream.SourceFile()
	// 获取第一个token的位置作为程序AST的位置（如果存在）
	// 注意：Current() 不会改变位置，所以可以安全调用
	var programLocation sharedVO.SourceLocation
	if tokenStream != nil && !tokenStream.IsAtEnd() {
		firstToken := tokenStream.Current()
		if firstToken != nil && firstToken.Type() != lexicalVO.EnhancedTokenTypeEOF {
			// 使用第一个实际token的位置
			programLocation = firstToken.Location()
		} else {
			// 如果没有有效token，使用文件起始位置
			programLocation = sharedVO.NewSourceLocation(sourceFile.Filename(), 1, 1, 0)
		}
	} else {
		// 如果tokenStream为空，使用文件起始位置
		programLocation = sharedVO.NewSourceLocation(sourceFile.Filename(), 1, 1, 0)
	}
	programAST := sharedVO.NewProgramAST(
		make([]sharedVO.ASTNode, 0),
		sourceFile.Filename(),
		programLocation,
	)

	// 解析程序头部（可选）
	if err := rdp.parseProgramHeader(ctx, programAST); err != nil {
		return nil, err
	}

	// 解析顶层声明
	for !rdp.isAtEnd() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 检查是否到达文件末尾
		if rdp.currentToken().Type() == lexicalVO.EnhancedTokenTypeEOF {
			break
		}

		// 解析单个顶层声明
		node, err := rdp.parseTopLevelDeclaration(ctx)
		if err != nil {
			rdp.addError(err.Error(), rdp.currentToken().Location())
			rdp.advance() // 跳过错误token，继续解析
			continue
		}

		if node != nil {
			programAST.AddNode(node)
		}
	}

	// 检查是否有解析错误
	if len(rdp.errors) > 0 {
		errorMessages := make([]string, 0, len(rdp.errors))
		for _, err := range rdp.errors {
			errorMessages = append(errorMessages, err.Message())
		}
		return programAST, fmt.Errorf("parsing completed with %d errors: %v", len(rdp.errors), errorMessages)
	}

	return programAST, nil
}

// parseProgramHeader 解析程序头部（可选）
func (rdp *RecursiveDescentParser) parseProgramHeader(ctx context.Context, programAST *sharedVO.ProgramAST) error {
	// 1. 解析包声明（必须在最前面）
	// match 会检查并消耗 'package' 关键字
	if rdp.match(lexicalVO.EnhancedTokenTypePackage) {
		pkgDecl, err := rdp.parsePackageDeclaration()
		if err != nil {
			return fmt.Errorf("failed to parse package declaration: %w", err)
		}
		programAST.SetPackage(pkgDecl)
	}

	// 2. 检查是否有模块声明（可选，兼容旧代码）
	if rdp.check(lexicalVO.EnhancedTokenTypeIdentifier) && rdp.currentToken().Lexeme() == "module" {
		// 解析模块声明
		moduleDecl, err := rdp.parseModuleDeclaration()
		if err != nil {
			return err
		}
		programAST.SetModuleDeclaration(moduleDecl)
	}

	// 3. 解析导入语句（支持 import 和 from ... import）
	for rdp.check(lexicalVO.EnhancedTokenTypeIdentifier) && rdp.currentToken().Lexeme() == "import" ||
		rdp.check(lexicalVO.EnhancedTokenTypeFrom) {
		importStmt, err := rdp.parseImportStatement()
		if err != nil {
			return err
		}
		programAST.AddImport(importStmt)
	}

	return nil
}

// parseTopLevelDeclaration 解析顶层声明
func (rdp *RecursiveDescentParser) parseTopLevelDeclaration(ctx context.Context) (sharedVO.ASTNode, error) {
	token := rdp.currentToken()

	// 检查是否是 private 关键字（可能在函数、结构体等声明前）
	// 注意：private 关键字由具体的解析方法处理（parseFunctionDeclaration、parseAsyncFunctionDeclaration等）
	// 这里不需要提前处理，让每个解析方法自己处理 private

	switch token.Type() {
	case lexicalVO.EnhancedTokenTypePrivate:
		// private 后面可能是 func、async、struct 等
		// 先消耗 private，然后递归调用解析下一个声明
		rdp.advance() // 消耗 private
		// 递归解析下一个声明（可能是 func、async、struct 等）
		node, err := rdp.parseTopLevelDeclaration(ctx)
		if err != nil {
			return nil, err
		}
		// 如果解析的是函数声明，需要设置可见性
		// 注意：这里需要根据节点类型设置可见性，但为了简化，我们让具体的解析方法处理
		return node, nil
		
	case lexicalVO.EnhancedTokenTypeAsync:
		// 解析异步函数声明
		return rdp.parseAsyncFunctionDeclaration()

	case lexicalVO.EnhancedTokenTypeFunc:
		// 解析函数声明
		return rdp.parseFunctionDeclaration()

	case lexicalVO.EnhancedTokenTypeStruct:
		return rdp.parseStructDeclaration()

	case lexicalVO.EnhancedTokenTypeEnum:
		return rdp.parseEnumDeclaration()

	case lexicalVO.EnhancedTokenTypeTrait:
		return rdp.parseTraitDeclaration()

	case lexicalVO.EnhancedTokenTypeImpl:
		return rdp.parseImplDeclaration()

	case lexicalVO.EnhancedTokenTypeLet:
		// 获取'let'关键字token的位置
		letLocation := rdp.currentToken().Location()
		rdp.consume(lexicalVO.EnhancedTokenTypeLet) // 消耗 'let'
		return rdp.parseGlobalVariableDeclaration(letLocation)

	case lexicalVO.EnhancedTokenTypeIdentifier:
		lexeme := token.Lexeme()
		switch lexeme {
		case "type":
			return rdp.parseTypeAliasDeclaration()
		case "const":
			// 获取'const'关键字token的位置
			constLocation := rdp.currentToken().Location()
			rdp.consume(lexicalVO.EnhancedTokenTypeIdentifier) // 消耗 'const'
			return rdp.parseConstantDeclaration(constLocation)
		default:
			// 可能是表达式语句或错误
			return rdp.parseExpressionStatement()
		}

	case lexicalVO.EnhancedTokenTypeSemicolon:
		// 分号可能是语句结束符，跳过它
		rdp.advance()
		return nil, nil // 返回 nil 表示没有节点，但也没有错误

	case lexicalVO.EnhancedTokenTypeEOF:
		// 文件结束，返回 nil
		return nil, nil

	default:
		// 如果是不认识的 token，跳过它（可能是文件结束或其他情况）
		// 不要尝试解析为表达式语句，因为这会导致错误
		rdp.advance()
		return nil, nil
	}
}

// parseModuleDeclaration 解析模块声明
func (rdp *RecursiveDescentParser) parseModuleDeclaration() (*sharedVO.ModuleDeclaration, error) {
	// 获取'module'关键字token的位置作为起始位置
	// 注意：在consume之前，currentToken()返回的是'module'关键字token
	startLocation := rdp.tokenStream.Current().Location()

	rdp.consume(lexicalVO.EnhancedTokenTypeIdentifier) // 消耗 'module'

	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected module name after 'module'")
	}

	moduleName := rdp.previousToken().Lexeme()

	// 解析可选的模块路径
	var modulePath string
	if rdp.match(lexicalVO.EnhancedTokenTypeString) {
		modulePath = rdp.previousToken().StringValue()
	}

	if !rdp.match(lexicalVO.EnhancedTokenTypeSemicolon) {
		return nil, fmt.Errorf("expected ';' after module declaration")
	}

	// 使用起始位置作为模块声明的位置
	return sharedVO.NewModuleDeclaration(moduleName, modulePath, startLocation), nil
}

// parsePackageDeclaration 解析包声明
// 注意：调用此方法时，'package' 关键字已经被 match 消耗了
func (rdp *RecursiveDescentParser) parsePackageDeclaration() (*sharedVO.PackageDeclaration, error) {
	// 获取前一个 token（即 'package' 关键字）的位置
	prevToken := rdp.previousToken()
	if prevToken == nil {
		return nil, fmt.Errorf("internal error: previous token is nil")
	}
	startLocation := prevToken.Location()

	// 解析包名（标识符，支持多级包名如 math.geometry）
	var packageNameParts []string
	for {
		if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
			if len(packageNameParts) == 0 {
				return nil, fmt.Errorf("expected package name after 'package'")
			}
			break
		}
		packageNameParts = append(packageNameParts, rdp.previousToken().Lexeme())
		
		// 检查是否有点号（多级包名）
		if !rdp.match(lexicalVO.EnhancedTokenTypeDot) {
			break
		}
	}
	
	packageName := ""
	for i, part := range packageNameParts {
		if i > 0 {
			packageName += "."
		}
		packageName += part
	}

	// 可选的分号（match 已经消耗了，不需要再次 consume）
	rdp.match(lexicalVO.EnhancedTokenTypeSemicolon)

	return sharedVO.NewPackageDeclaration(packageName, startLocation), nil
}

// parseImportStatement 解析导入语句（扩展支持 from ... import）
func (rdp *RecursiveDescentParser) parseImportStatement() (*sharedVO.ImportStatement, error) {
	startLocation := rdp.tokenStream.Current().Location()

	// 检查是否是 from ... import 语法
	if rdp.check(lexicalVO.EnhancedTokenTypeFrom) {
		return rdp.parseFromImportStatement(startLocation)
	}

	// 原有的 import "path" 语法
	// 先检查并消耗 import 关键字
	if !rdp.check(lexicalVO.EnhancedTokenTypeIdentifier) || rdp.currentToken().Lexeme() != "import" {
		return nil, fmt.Errorf("expected 'import' or 'from' keyword")
	}
	rdp.consume(lexicalVO.EnhancedTokenTypeIdentifier) // 消耗 'import'
	return rdp.parsePackageImportStatement(startLocation)
}

// parsePackageImportStatement 解析包级导入
// 注意：import 关键字已在 parseImportStatement 中消耗
// 支持两种语法：
// 1. import "package" (字符串形式，向后兼容)
// 2. import xxpkg (标识符形式，类似 Python)
func (rdp *RecursiveDescentParser) parsePackageImportStatement(startLocation sharedVO.SourceLocation) (*sharedVO.ImportStatement, error) {
	var importPath string
	
	// 优先尝试字符串形式（向后兼容）
	if rdp.match(lexicalVO.EnhancedTokenTypeString) {
		importPath = rdp.previousToken().StringValue()
	} else if rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		// 支持标识符形式（类似 Python）：import xxpkg
		importPath = rdp.previousToken().Lexeme()
	} else {
		return nil, fmt.Errorf("expected import path (string or identifier) after 'import'")
	}

	// 解析可选的别名
	var alias string
	if rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) && rdp.previousToken().Lexeme() == "as" {
		// match 已经消耗了 'as'
		if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
			return nil, fmt.Errorf("expected alias name after 'as'")
		}
		alias = rdp.previousToken().Lexeme()
	}

	// 可选的分号
	rdp.match(lexicalVO.EnhancedTokenTypeSemicolon) // match 已经消耗了分号（如果存在）

	return sharedVO.NewPackageImport(importPath, alias, startLocation), nil
}

// parseFromImportStatement 解析 from ... import 语法
func (rdp *RecursiveDescentParser) parseFromImportStatement(startLocation sharedVO.SourceLocation) (*sharedVO.ImportStatement, error) {
	rdp.consume(lexicalVO.EnhancedTokenTypeFrom) // 消耗 'from'

	// 解析导入路径（字符串或相对路径）
	var importPath string
	if rdp.match(lexicalVO.EnhancedTokenTypeString) {
		importPath = rdp.previousToken().StringValue()
	} else if rdp.match(lexicalVO.EnhancedTokenTypeDot) {
		// 相对路径：./ 或 ../
		importPath = rdp.parseRelativePath()
	} else {
		return nil, fmt.Errorf("expected import path after 'from'")
	}

	// 消耗 'import'
	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) || rdp.previousToken().Lexeme() != "import" {
		return nil, fmt.Errorf("expected 'import' after import path")
	}
	// match 已经消耗了 token，无需再次 consume

	// 解析导入元素列表
	elements := make([]*sharedVO.ImportElement, 0)
	for {
		if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
			return nil, fmt.Errorf("expected element name in import list")
		}

		elementName := rdp.previousToken().Lexeme()
		elementLocation := rdp.previousToken().Location()

		// 解析可选的别名
		var alias string
		if rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) && rdp.previousToken().Lexeme() == "as" {
			// match 已经消耗了 'as'
			if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
				return nil, fmt.Errorf("expected alias name after 'as'")
			}
			alias = rdp.previousToken().Lexeme()
		}

		elements = append(elements, sharedVO.NewImportElement(elementName, alias, elementLocation))

		// 检查是否有更多元素（逗号分隔）
		if rdp.match(lexicalVO.EnhancedTokenTypeComma) {
			// match 已经消耗了逗号
			continue
		}

		break
	}

	// 可选的分号
	rdp.match(lexicalVO.EnhancedTokenTypeSemicolon) // match 已经消耗了分号（如果存在）

	return sharedVO.NewElementImport(importPath, elements, startLocation), nil
}

// parseRelativePath 解析相对路径
func (rdp *RecursiveDescentParser) parseRelativePath() string {
	var pathBuilder strings.Builder

	// 解析 ./ 或 ../
	if rdp.match(lexicalVO.EnhancedTokenTypeDot) {
		pathBuilder.WriteString(".")
		rdp.consume(lexicalVO.EnhancedTokenTypeDot)

		if rdp.match(lexicalVO.EnhancedTokenTypeDot) {
			pathBuilder.WriteString(".")
			rdp.consume(lexicalVO.EnhancedTokenTypeDot)
		}

		// 检查是否有斜杠（可能是字符串中的斜杠，或者单独的标识符）
		// 如果下一个是标识符，说明是路径的一部分
		if rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
			pathBuilder.WriteString("/")
		}
	}

	// 解析路径部分（标识符序列，用点或斜杠分隔）
	for rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		pathBuilder.WriteString(rdp.previousToken().Lexeme())
		
		// 检查是否有分隔符（点或斜杠）
		if rdp.match(lexicalVO.EnhancedTokenTypeDot) {
			pathBuilder.WriteString("/")
			rdp.consume(lexicalVO.EnhancedTokenTypeDot)
		} else if rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
			// 下一个标识符，添加斜杠
			pathBuilder.WriteString("/")
		} else {
			break
		}
	}

	return pathBuilder.String()
}

// parseFunctionDeclaration 解析函数声明
func (rdp *RecursiveDescentParser) parseFunctionDeclaration() (*sharedVO.FunctionDeclaration, error) {
	// 获取函数声明的起始位置（可能是 private 或 func）
	funcLocation := rdp.currentToken().Location()
	
	// 解析可选的 private 关键字
	// 注意：private 可能在 parseTopLevelDeclaration 中已经处理了（通过 previousToken）
	visibility := sharedVO.VisibilityPublic
	if rdp.previousToken() != nil && rdp.previousToken().Type() == lexicalVO.EnhancedTokenTypePrivate {
		// private 已经在 parseTopLevelDeclaration 中消耗了
		visibility = sharedVO.VisibilityPrivate
		funcLocation = rdp.currentToken().Location() // 当前位置就是 func 关键字
	} else if rdp.match(lexicalVO.EnhancedTokenTypePrivate) {
		// private 在 func 之前（直接相邻）
		visibility = sharedVO.VisibilityPrivate
		// match 已经消耗了 private
		funcLocation = rdp.currentToken().Location() // 更新位置为 func 关键字的位置
	}
	
	rdp.consume(lexicalVO.EnhancedTokenTypeFunc) // 消耗 'func'

	// 解析函数名
	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected function name after 'func'")
	}
	funcName := rdp.previousToken().Lexeme()

	// 解析泛型参数（可选）
	var genericParams []*sharedVO.GenericParameter
	if rdp.match(lexicalVO.EnhancedTokenTypeLessThan) {
		params, err := rdp.parseGenericParameters()
		if err != nil {
			return nil, err
		}
		genericParams = params
	}

	// 解析参数列表
	if !rdp.match(lexicalVO.EnhancedTokenTypeLeftParen) {
		return nil, fmt.Errorf("expected '(' after function name")
	}

	parameters, err := rdp.parseParameterList()
	if err != nil {
		return nil, err
	}

	if !rdp.match(lexicalVO.EnhancedTokenTypeRightParen) {
		return nil, fmt.Errorf("expected ')' after parameter list")
	}

	// 解析返回类型（可选）
	var returnType *sharedVO.TypeAnnotation
	if rdp.match(lexicalVO.EnhancedTokenTypeArrow) {
		typeAnnotation, err := rdp.parseTypeAnnotation()
		if err != nil {
			return nil, err
		}
		returnType = typeAnnotation
	}

	// 解析函数体
	var body *sharedVO.BlockStatement
	if rdp.match(lexicalVO.EnhancedTokenTypeLeftBrace) {
		// 获取'{'的位置作为块语句的起始位置
		blockStartLocation := rdp.previousToken().Location()
		block, err := rdp.parseBlockStatement(blockStartLocation)
		if err != nil {
			return nil, err
		}
		body = block
	} else if rdp.match(lexicalVO.EnhancedTokenTypeAssign) {
		// 函数简写语法: func square(x) = x * x
		// 获取'='的位置作为块语句和return语句的起始位置
		assignLocation := rdp.previousToken().Location()
		expr, err := rdp.parseExpression()
		if err != nil {
			return nil, err
		}
		body = sharedVO.NewBlockStatement([]sharedVO.ASTNode{
			sharedVO.NewReturnStatement(expr, assignLocation),
		}, assignLocation)
	} else {
		// 抽象函数声明（接口中）
		if !rdp.match(lexicalVO.EnhancedTokenTypeSemicolon) {
			return nil, fmt.Errorf("expected function body or ';' for abstract function")
		}
	}

	return sharedVO.NewFunctionDeclarationWithVisibility(
		funcName,
		visibility,
		genericParams,
		parameters,
		returnType,
		body,
		funcLocation,
	), nil
}

// parseAsyncFunctionDeclaration 解析异步函数声明
func (rdp *RecursiveDescentParser) parseAsyncFunctionDeclaration() (*sharedVO.AsyncFunctionDeclaration, error) {
	// 获取异步函数声明的起始位置（可能是 private 或 async）
	asyncLocation := rdp.currentToken().Location()
	
	// 检查 private 关键字（可能在 parseTopLevelDeclaration 中已经消耗了）
	// 如果 previousToken 是 private，说明 private 已经在顶层处理了
	// 我们需要在消耗 async 之前检查，因为消耗 async 后 previousToken 会变成 async
	hasPrivate := rdp.previousToken() != nil && rdp.previousToken().Type() == lexicalVO.EnhancedTokenTypePrivate
	
	rdp.consume(lexicalVO.EnhancedTokenTypeAsync) // 消耗 'async'

	// 解析基础函数声明
	// 如果 hasPrivate 为 true，我们需要手动设置可见性
	funcDecl, err := rdp.parseFunctionDeclaration()
	if err != nil {
		return nil, err
	}

	// 如果检测到 private，需要更新函数声明的可见性
	if hasPrivate && funcDecl.Visibility() == sharedVO.VisibilityPublic {
		// 创建一个新的函数声明，带有 private 可见性
		// 注意：FunctionDeclaration 是不可变的，我们需要使用 NewFunctionDeclarationWithVisibility
		funcDecl = sharedVO.NewFunctionDeclarationWithVisibility(
			funcDecl.Name(),
			sharedVO.VisibilityPrivate,
			funcDecl.GenericParams(),
			funcDecl.Parameters(),
			funcDecl.ReturnType(),
			funcDecl.Body(),
			funcDecl.Location(),
		)
	}

	return sharedVO.NewAsyncFunctionDeclaration(funcDecl, asyncLocation), nil
}

// parseStructDeclaration 解析结构体声明
func (rdp *RecursiveDescentParser) parseStructDeclaration() (*sharedVO.StructDeclaration, error) {
	// 获取'struct'关键字token的位置作为结构体声明的起始位置
	structLocation := rdp.currentToken().Location()
	rdp.consume(lexicalVO.EnhancedTokenTypeStruct) // 消耗 'struct'

	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected struct name after 'struct'")
	}
	structName := rdp.previousToken().Lexeme()

	if !rdp.match(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after struct name")
	}

	// 解析字段
	fields := make([]*sharedVO.StructField, 0)
	for !rdp.isAtEnd() && !rdp.check(lexicalVO.EnhancedTokenTypeRightBrace) {
		field, err := rdp.parseStructField()
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)

		if !rdp.match(lexicalVO.EnhancedTokenTypeComma) && !rdp.check(lexicalVO.EnhancedTokenTypeRightBrace) {
			return nil, fmt.Errorf("expected ',' or '}' after struct field")
		}
	}

	if !rdp.match(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after struct fields")
	}

	return sharedVO.NewStructDeclaration(structName, fields, structLocation), nil
}

// parseEnumDeclaration 解析枚举声明
func (rdp *RecursiveDescentParser) parseEnumDeclaration() (*sharedVO.EnumDeclaration, error) {
	// 获取'enum'关键字token的位置作为枚举声明的起始位置
	enumLocation := rdp.currentToken().Location()
	rdp.consume(lexicalVO.EnhancedTokenTypeEnum) // 消耗 'enum'

	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected enum name after 'enum'")
	}
	enumName := rdp.previousToken().Lexeme()

	if !rdp.match(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after enum name")
	}

	// 解析枚举变体
	variants := make([]*sharedVO.EnumVariant, 0)
	for !rdp.isAtEnd() && !rdp.check(lexicalVO.EnhancedTokenTypeRightBrace) {
		if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
			return nil, fmt.Errorf("expected enum variant name")
		}
		variantNameToken := rdp.previousToken()
		variantName := variantNameToken.Lexeme()
		// 使用变体名token的位置作为枚举变体的位置
		variantLocation := variantNameToken.Location()
		variants = append(variants, sharedVO.NewEnumVariant(variantName, variantLocation))

		if !rdp.match(lexicalVO.EnhancedTokenTypeComma) && !rdp.check(lexicalVO.EnhancedTokenTypeRightBrace) {
			return nil, fmt.Errorf("expected ',' or '}' after enum variant")
		}
	}

	if !rdp.match(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after enum variants")
	}

	return sharedVO.NewEnumDeclaration(enumName, variants, enumLocation), nil
}

// parseTraitDeclaration 解析Trait声明
func (rdp *RecursiveDescentParser) parseTraitDeclaration() (*sharedVO.TraitDeclaration, error) {
	// 获取'trait'关键字token的位置作为Trait声明的起始位置
	traitLocation := rdp.currentToken().Location()
	rdp.consume(lexicalVO.EnhancedTokenTypeTrait) // 消耗 'trait'

	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected trait name after 'trait'")
	}
	traitName := rdp.previousToken().Lexeme()

	// 解析泛型参数（可选）
	var genericParams []*sharedVO.GenericParameter
	if rdp.match(lexicalVO.EnhancedTokenTypeLessThan) {
		params, err := rdp.parseGenericParameters()
		if err != nil {
			return nil, err
		}
		genericParams = params
	}

	if !rdp.match(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after trait name")
	}

	// 解析trait方法
	methods := make([]*sharedVO.TraitMethod, 0)
	for !rdp.isAtEnd() && !rdp.check(lexicalVO.EnhancedTokenTypeRightBrace) {
		method, err := rdp.parseTraitMethod()
		if err != nil {
			return nil, err
		}
		methods = append(methods, method)
	}

	if !rdp.match(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after trait methods")
	}

	return sharedVO.NewTraitDeclaration(traitName, genericParams, methods, traitLocation), nil
}

// parseImplDeclaration 解析Impl声明
func (rdp *RecursiveDescentParser) parseImplDeclaration() (*sharedVO.ImplDeclaration, error) {
	// 获取'impl'关键字token的位置作为实现声明的起始位置
	implLocation := rdp.currentToken().Location()
	rdp.consume(lexicalVO.EnhancedTokenTypeImpl) // 消耗 'impl'

	// 解析Trait类型
	traitType, err := rdp.parseTypeAnnotation()
	if err != nil {
		return nil, err
	}

	// 解析目标类型
	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) || rdp.currentToken().Lexeme() != "for" {
		return nil, fmt.Errorf("expected 'for' after trait type in impl declaration")
	}
	rdp.consume(lexicalVO.EnhancedTokenTypeIdentifier) // 消耗 'for'

	targetType, err := rdp.parseTypeAnnotation()
	if err != nil {
		return nil, err
	}

	if !rdp.match(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after impl declaration")
	}

	// 解析实现的方法
	methods := make([]*sharedVO.FunctionDeclaration, 0)
	for !rdp.isAtEnd() && !rdp.check(lexicalVO.EnhancedTokenTypeRightBrace) {
		method, err := rdp.parseFunctionDeclaration()
		if err != nil {
			return nil, err
		}
		methods = append(methods, method)
	}

	if !rdp.match(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after impl methods")
	}

	return sharedVO.NewImplDeclaration(traitType, targetType, methods, implLocation), nil
}

// 工具方法

// currentToken 获取当前位置的Token
func (rdp *RecursiveDescentParser) currentToken() *lexicalVO.EnhancedToken {
	return rdp.tokenStream.Current()
}

// previousToken 获取前一个Token
func (rdp *RecursiveDescentParser) previousToken() *lexicalVO.EnhancedToken {
	// 完善实现：返回保存的前一个token
	return rdp.prevToken
}

// peek 查看下一个Token
func (rdp *RecursiveDescentParser) peek() *lexicalVO.EnhancedToken {
	return rdp.tokenStream.Peek()
}

// advance 前进到下一个Token
func (rdp *RecursiveDescentParser) advance() *lexicalVO.EnhancedToken {
	// 完善实现：保存当前token作为prevToken
	currentToken := rdp.tokenStream.Current()
	
	// 更新prevToken为调用Next()之前的当前token
	rdp.prevToken = currentToken
	
	// 调用Next()获取下一个token并移动流位置
	token := rdp.tokenStream.Next()
	
	return token
}

// match 如果当前Token匹配指定类型则消耗它
func (rdp *RecursiveDescentParser) match(tokenType lexicalVO.EnhancedTokenType) bool {
	if rdp.check(tokenType) {
		rdp.advance()
		return true
	}
	return false
}

// check 检查当前Token是否为指定类型
func (rdp *RecursiveDescentParser) check(tokenType lexicalVO.EnhancedTokenType) bool {
	if rdp.isAtEnd() {
		return false
	}
	return rdp.currentToken().Type() == tokenType
}

// consume 消耗指定类型的Token，如果不匹配则报错
func (rdp *RecursiveDescentParser) consume(tokenType lexicalVO.EnhancedTokenType) *lexicalVO.EnhancedToken {
	if rdp.check(tokenType) {
		return rdp.advance()
	}

	token := rdp.currentToken()
	panic(fmt.Sprintf("Expected token %s but found %s at %s",
		tokenType, token.Type(), token.Location()))
}

// isAtEnd 检查是否到达Token流末尾
func (rdp *RecursiveDescentParser) isAtEnd() bool {
	return rdp.tokenStream.IsAtEnd() ||
		rdp.currentToken().Type() == lexicalVO.EnhancedTokenTypeEOF
}

// addError 添加解析错误
func (rdp *RecursiveDescentParser) addError(message string, location sharedVO.SourceLocation) {
	error := sharedVO.NewParseError(message, location,
		sharedVO.ErrorTypeSyntax, sharedVO.SeverityError)
	rdp.errors = append(rdp.errors, error)
}

// 辅助解析方法（声明，实际实现需要补充）

func (rdp *RecursiveDescentParser) parseGenericParameters() ([]*sharedVO.GenericParameter, error) {
	// 已经消耗了 '<'，现在解析泛型参数列表
	// 格式：T 或 T, U 或 T: Trait 或 T: Trait1 + Trait2
	params := make([]*sharedVO.GenericParameter, 0)

	for {
		// 解析参数名
		if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
			return nil, fmt.Errorf("expected generic parameter name")
		}
		paramNameToken := rdp.previousToken()
		paramName := paramNameToken.Lexeme()
		// 获取参数名token的位置作为泛型参数的位置
		paramLocation := paramNameToken.Location()

		// 解析约束（可选）
		var constraints []string
		if rdp.match(lexicalVO.EnhancedTokenTypeColon) {
			// 解析约束列表：Trait1 + Trait2 + ...
			for {
				if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
					return nil, fmt.Errorf("expected trait name after ':' in generic parameter constraint")
				}
				constraintName := rdp.previousToken().Lexeme()
				constraints = append(constraints, constraintName)

				// 检查是否有更多约束（用 + 连接）
				if !rdp.match(lexicalVO.EnhancedTokenTypePlus) {
					break
				}
			}
		}

		// 创建泛型参数，使用参数名token的位置
		genericParam := sharedVO.NewGenericParameter(paramName, constraints, paramLocation)
		params = append(params, genericParam)

		// 检查是否有更多参数
		if !rdp.match(lexicalVO.EnhancedTokenTypeComma) {
			break
		}
	}

	// 消耗 '>'
	if !rdp.match(lexicalVO.EnhancedTokenTypeGreaterThan) {
		return nil, fmt.Errorf("expected '>' after generic parameters")
	}

	return params, nil
}

func (rdp *RecursiveDescentParser) parseParameterList() ([]*sharedVO.Parameter, error) {
	// 已经消耗了 '('，现在解析参数列表
	// 格式：name: type 或 name: type, name2: type2 或空列表
	parameters := make([]*sharedVO.Parameter, 0)

	// 检查是否为空参数列表
	if rdp.check(lexicalVO.EnhancedTokenTypeRightParen) {
		return parameters, nil
	}

	for {
		// 解析参数名
		if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
			return nil, fmt.Errorf("expected parameter name")
		}
		paramNameToken := rdp.previousToken()
		paramName := paramNameToken.Lexeme()
		// 获取参数名token的位置作为参数的位置
		paramLocation := paramNameToken.Location()

		// 解析类型注解（必需）
		if !rdp.match(lexicalVO.EnhancedTokenTypeColon) {
			return nil, fmt.Errorf("expected ':' after parameter name")
		}

		typeAnnotation, err := rdp.parseTypeAnnotation()
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter type: %w", err)
		}

		// 创建参数，使用参数名token的位置
		param := sharedVO.NewParameter(paramName, typeAnnotation, paramLocation)
		parameters = append(parameters, param)

		// 检查是否有更多参数
		if !rdp.match(lexicalVO.EnhancedTokenTypeComma) {
			break
		}
	}

	return parameters, nil
}

func (rdp *RecursiveDescentParser) parseTypeAnnotation() (*sharedVO.TypeAnnotation, error) {
	// 解析类型注解
	// 格式：typeName 或 *typeName 或 typeName<args> 或 *typeName<args> 或 [T]（数组类型）

	// 检查是否为指针类型，如果是，从*开始记录位置
	var startLocation sharedVO.SourceLocation
	isPointer := false
	if rdp.match(lexicalVO.EnhancedTokenTypeMultiply) {
		isPointer = true
		// 获取*的位置作为起始位置
		startLocation = rdp.previousToken().Location()
	}

	// 数组类型 [T]，如 [int]、[string]
	if rdp.match(lexicalVO.EnhancedTokenTypeLeftBracket) {
		bracketLoc := rdp.previousToken().Location()
		elem, err := rdp.parseTypeAnnotation()
		if err != nil {
			return nil, err
		}
		if !rdp.match(lexicalVO.EnhancedTokenTypeRightBracket) {
			return nil, fmt.Errorf("expected ']' after array element type")
		}
		return sharedVO.NewTypeAnnotation("[]", []*sharedVO.TypeAnnotation{elem}, false, bracketLoc), nil
	}

	// 解析类型名（identifier 或关键字如 chan）
	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) && !rdp.match(lexicalVO.EnhancedTokenTypeChan) {
		return nil, fmt.Errorf("expected type name in type annotation")
	}
	typeNameToken := rdp.previousToken()
	typeName := typeNameToken.Lexeme()
	// 如果不是指针类型，使用类型名token的位置；如果是指针类型，使用之前记录的*的位置
	if !isPointer {
		startLocation = typeNameToken.Location()
	}

	var genericArgs []*sharedVO.TypeAnnotation
	// chan 后接元素类型，如 chan string
	if typeName == "chan" && (rdp.check(lexicalVO.EnhancedTokenTypeIdentifier) || rdp.check(lexicalVO.EnhancedTokenTypeChan)) {
		elem, err := rdp.parseTypeAnnotation()
		if err != nil {
			return nil, err
		}
		genericArgs = []*sharedVO.TypeAnnotation{elem}
	}
	// 解析泛型参数（可选）：< T, U > 或 [ T ]（如 Future[string]）
	if genericArgs == nil && (rdp.match(lexicalVO.EnhancedTokenTypeLessThan) || rdp.match(lexicalVO.EnhancedTokenTypeLeftBracket)) {
		useBracket := rdp.previousToken().Type() == lexicalVO.EnhancedTokenTypeLeftBracket
		for {
			argType, err := rdp.parseTypeAnnotation()
			if err != nil {
				return nil, fmt.Errorf("failed to parse generic argument type: %w", err)
			}
			genericArgs = append(genericArgs, argType)
			if !rdp.match(lexicalVO.EnhancedTokenTypeComma) {
				break
			}
		}
		if useBracket {
			if !rdp.match(lexicalVO.EnhancedTokenTypeRightBracket) {
				return nil, fmt.Errorf("expected ']' after generic type arguments")
			}
		} else if !rdp.match(lexicalVO.EnhancedTokenTypeGreaterThan) {
			return nil, fmt.Errorf("expected '>' after generic type arguments")
		}
	}

	return sharedVO.NewTypeAnnotation(typeName, genericArgs, isPointer, startLocation), nil
}

// parseBlockStatementInner 解析块内单条语句（let/return/const/表达式语句）
func (rdp *RecursiveDescentParser) parseBlockStatementInner() (sharedVO.ASTNode, error) {
	token := rdp.currentToken()
	switch token.Type() {
	case lexicalVO.EnhancedTokenTypeLet:
		letLocation := token.Location()
		rdp.advance()
		return rdp.parseGlobalVariableDeclaration(letLocation)
	case lexicalVO.EnhancedTokenTypeReturn:
		returnLocation := token.Location()
		rdp.advance()
		var expr sharedVO.ASTNode
		if !rdp.isAtEnd() && !rdp.check(lexicalVO.EnhancedTokenTypeSemicolon) && !rdp.check(lexicalVO.EnhancedTokenTypeRightBrace) {
			var err error
			expr, err = rdp.parseExpression()
			if err != nil {
				return nil, err
			}
		}
		rdp.match(lexicalVO.EnhancedTokenTypeSemicolon)
		return sharedVO.NewReturnStatement(expr, returnLocation), nil
	case lexicalVO.EnhancedTokenTypeIdentifier:
		if token.Lexeme() == "const" {
			constLocation := token.Location()
			rdp.advance()
			return rdp.parseConstantDeclaration(constLocation)
		}
	}
	return rdp.parseExpressionStatement()
}

func (rdp *RecursiveDescentParser) parseBlockStatement(startLocation sharedVO.SourceLocation) (*sharedVO.BlockStatement, error) {
	// 已经消耗了 '{'，现在解析块内的语句
	statements := make([]sharedVO.ASTNode, 0)

	for !rdp.isAtEnd() {
		if rdp.check(lexicalVO.EnhancedTokenTypeRightBrace) {
			rdp.advance()
			break
		}
		if rdp.match(lexicalVO.EnhancedTokenTypeSemicolon) {
			continue
		}

		stmt, err := rdp.parseBlockStatementInner()
		if err != nil {
			rdp.addError(err.Error(), rdp.currentToken().Location())
			rdp.advance()
			continue
		}
		if stmt != nil {
			statements = append(statements, stmt)
		}
	}

	// 使用传入的起始位置作为块语句的位置
	return sharedVO.NewBlockStatement(statements, startLocation), nil
}

func (rdp *RecursiveDescentParser) parseExpression() (sharedVO.ASTNode, error) {
	if rdp.expressionDelegate != nil {
		return rdp.expressionDelegate.ParseExpression(rdp.parseContextForDelegate())
	}
	// 无委托时的桩行为
	token := rdp.currentToken()
	switch token.Type() {
	case lexicalVO.EnhancedTokenTypeIdentifier:
		rdp.advance()
		return nil, fmt.Errorf("expression parsing should be delegated to Pratt parser via ParserCoordinator")
	case lexicalVO.EnhancedTokenTypeNumber:
		rdp.advance()
		return nil, fmt.Errorf("expression parsing should be delegated to Pratt parser via ParserCoordinator")
	case lexicalVO.EnhancedTokenTypeString:
		rdp.advance()
		return nil, fmt.Errorf("expression parsing should be delegated to Pratt parser via ParserCoordinator")
	default:
		return nil, fmt.Errorf("unexpected token in expression: %s", token.Type())
	}
}

// parseContextForDelegate 返回用于委托解析的 context，流位置由共享 tokenStream 保证
func (rdp *RecursiveDescentParser) parseContextForDelegate() context.Context {
	if rdp.parseCtx != nil {
		return rdp.parseCtx
	}
	return context.Background()
}

func (rdp *RecursiveDescentParser) parseStructField() (*sharedVO.StructField, error) {
	// 解析结构体字段
	// 格式：fieldName: fieldType 或 fieldName: fieldType,

	// 解析字段名
	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected struct field name")
	}
	fieldNameToken := rdp.previousToken()
	fieldName := fieldNameToken.Lexeme()
	// 使用字段名token的位置作为结构体字段的位置
	fieldLocation := fieldNameToken.Location()

	// 解析类型注解（必需）
	if !rdp.match(lexicalVO.EnhancedTokenTypeColon) {
		return nil, fmt.Errorf("expected ':' after struct field name")
	}

	fieldType, err := rdp.parseTypeAnnotation()
	if err != nil {
		return nil, fmt.Errorf("failed to parse struct field type: %w", err)
	}

	return sharedVO.NewStructField(fieldName, fieldType, fieldLocation), nil
}

func (rdp *RecursiveDescentParser) parseTraitMethod() (*sharedVO.TraitMethod, error) {
	// 解析Trait方法
	// 格式：func methodName(params) -> returnType; 或 func methodName(params) -> returnType { body }
	
	// 获取'func'关键字token的位置作为Trait方法的起始位置
	funcLocation := rdp.currentToken().Location()
	// 解析func关键字
	if !rdp.match(lexicalVO.EnhancedTokenTypeFunc) {
		return nil, fmt.Errorf("expected 'func' in trait method")
	}

	// 解析方法名
	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected trait method name after 'func'")
	}
	methodName := rdp.previousToken().Lexeme()

	// 解析泛型参数（可选）：func name[T, U](...)
	var methodGenericParams []*sharedVO.GenericParameter
	if rdp.match(lexicalVO.EnhancedTokenTypeLessThan) {
		params, err := rdp.parseGenericParameters()
		if err != nil {
			return nil, err
		}
		methodGenericParams = params
	}

	// 解析参数列表
	if !rdp.match(lexicalVO.EnhancedTokenTypeLeftParen) {
		return nil, fmt.Errorf("expected '(' after trait method name")
	}

	parameters, err := rdp.parseParameterList()
	if err != nil {
		return nil, err
	}

	if !rdp.match(lexicalVO.EnhancedTokenTypeRightParen) {
		return nil, fmt.Errorf("expected ')' after parameter list")
	}

	// 解析返回类型（可选）
	// 支持两种语法：
	// 1. func name() -> ReturnType (旧语法，保持向后兼容)
	// 2. func name() ReturnType (新语法，不需要箭头)
	var returnType *sharedVO.TypeAnnotation
	if rdp.match(lexicalVO.EnhancedTokenTypeArrow) {
		// 旧语法：使用 -> 箭头
		typeAnnotation, err := rdp.parseTypeAnnotation()
		if err != nil {
			return nil, err
		}
		returnType = typeAnnotation
	} else {
		// 新语法：尝试直接解析类型（如果后面不是 '{'）
		// 检查下一个 token 是否是 '{'，如果不是，可能是返回类型
		if !rdp.check(lexicalVO.EnhancedTokenTypeLeftBrace) {
			// 尝试解析类型（可能是返回类型）
			// 保存当前位置以便回退
			savedPosition := rdp.position
			typeAnnotation, err := rdp.parseTypeAnnotation()
			if err == nil {
				// 解析成功，检查后面是否是 '{'
				if rdp.check(lexicalVO.EnhancedTokenTypeLeftBrace) {
					// 确实是返回类型
					returnType = typeAnnotation
				} else {
					// 不是返回类型，回退
					rdp.position = savedPosition
				}
			} else {
				// 解析失败，回退
				rdp.position = savedPosition
			}
		}
	}

	// 检查是否有方法体（抽象方法没有方法体）
	isAbstract := false
	if rdp.match(lexicalVO.EnhancedTokenTypeLeftBrace) {
		// 有方法体，解析并跳过（Trait方法可以有默认实现）
		// 获取'{'的位置作为块语句的起始位置
		blockStartLocation := rdp.previousToken().Location()
		_, err := rdp.parseBlockStatement(blockStartLocation)
		if err != nil {
			return nil, err
		}
	} else if rdp.match(lexicalVO.EnhancedTokenTypeSemicolon) {
		// 抽象方法，以分号结尾
		isAbstract = true
	} else {
		return nil, fmt.Errorf("expected method body '{' or ';' for abstract method")
	}

	return sharedVO.NewTraitMethod(methodName, methodGenericParams, parameters, returnType, isAbstract, funcLocation), nil
}

func (rdp *RecursiveDescentParser) parseGlobalVariableDeclaration(startLocation sharedVO.SourceLocation) (sharedVO.ASTNode, error) {
	// 解析全局变量声明
	// 格式：let name: type = value; 或 let name = value;
	
	// 已经消耗了 'let'，现在解析变量名
	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected variable name after 'let'")
	}
	varName := rdp.previousToken().Lexeme()

	// 解析类型注解（可选）
	var varType *sharedVO.TypeAnnotation
	if rdp.match(lexicalVO.EnhancedTokenTypeColon) {
		typeAnnotation, err := rdp.parseTypeAnnotation()
		if err != nil {
			return nil, fmt.Errorf("failed to parse variable type: %w", err)
		}
		varType = typeAnnotation
	}

	// 解析初始化表达式（必需）
	if !rdp.match(lexicalVO.EnhancedTokenTypeAssign) {
		return nil, fmt.Errorf("expected '=' after variable name in let declaration")
	}

	initializer, err := rdp.parseExpression()
	if err != nil {
		return nil, fmt.Errorf("failed to parse variable initializer: %w", err)
	}

	// 消耗分号（可选）
	rdp.match(lexicalVO.EnhancedTokenTypeSemicolon)

	return sharedVO.NewVariableDeclaration(varName, varType, initializer, startLocation), nil
}

func (rdp *RecursiveDescentParser) parseTypeAliasDeclaration() (sharedVO.ASTNode, error) {
	// 解析类型别名声明
	// 格式：type AliasName = OriginalType;
	
	// 获取'type'关键字的位置
	typeToken := rdp.previousToken() // 'type'关键字已经在parseTopLevelDeclaration中消耗
	typeLocation := typeToken.Location()
	
	// 解析别名名称
	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected type alias name after 'type'")
	}
	aliasNameToken := rdp.previousToken()
	aliasName := aliasNameToken.Lexeme()

	// 解析等号
	if !rdp.match(lexicalVO.EnhancedTokenTypeAssign) {
		return nil, fmt.Errorf("expected '=' after type alias name")
	}

	// 解析原始类型
	originalType, err := rdp.parseTypeAnnotation()
	if err != nil {
		return nil, fmt.Errorf("failed to parse original type: %w", err)
	}

	// 消耗分号（可选）
	rdp.match(lexicalVO.EnhancedTokenTypeSemicolon)

	// 创建类型别名声明节点
	return sharedVO.NewTypeAliasDeclaration(aliasName, originalType, typeLocation), nil
}

func (rdp *RecursiveDescentParser) parseConstantDeclaration(startLocation sharedVO.SourceLocation) (sharedVO.ASTNode, error) {
	// 解析常量声明
	// 格式：const ConstantName: type = value; 或 const ConstantName = value;
	
	// 已经消耗了 'const'，现在解析常量名
	if !rdp.match(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected constant name after 'const'")
	}
	constName := rdp.previousToken().Lexeme()

	// 解析类型注解（可选）
	var constType *sharedVO.TypeAnnotation
	if rdp.match(lexicalVO.EnhancedTokenTypeColon) {
		typeAnnotation, err := rdp.parseTypeAnnotation()
		if err != nil {
			return nil, fmt.Errorf("failed to parse constant type: %w", err)
		}
		constType = typeAnnotation
	}

	// 解析初始化表达式（必需）
	if !rdp.match(lexicalVO.EnhancedTokenTypeAssign) {
		return nil, fmt.Errorf("expected '=' after constant name in const declaration")
	}

	initializer, err := rdp.parseExpression()
	if err != nil {
		return nil, fmt.Errorf("failed to parse constant initializer: %w", err)
	}

	// 消耗分号（可选）
	rdp.match(lexicalVO.EnhancedTokenTypeSemicolon)

	// TODO: ConstantDeclaration AST节点类型还没有定义，暂时使用VariableDeclaration
	// 后续需要在ast.go中定义ConstantDeclaration节点
	// 暂时返回VariableDeclaration，但标记为常量
	// 使用传入的起始位置作为变量声明的位置
	return sharedVO.NewVariableDeclaration(constName, constType, initializer, startLocation), nil
}

func (rdp *RecursiveDescentParser) parseExpressionStatement() (sharedVO.ASTNode, error) {
	// 表达式语句：expression;
	// 获取表达式起始位置
	exprStartLocation := rdp.currentToken().Location()
	
	// 解析表达式
	expr, err := rdp.parseExpression()
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression in expression statement: %w", err)
	}

	// 消耗分号（可选，某些语言允许省略）
	rdp.match(lexicalVO.EnhancedTokenTypeSemicolon)

	// 创建表达式语句节点
	return sharedVO.NewExpressionStatement(expr, exprStartLocation), nil
}
