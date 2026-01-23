package parser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"echo/internal/modules/frontend/domain/entities"
	lexicalServices "echo/internal/modules/frontend/domain/lexical/services"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
)

// ParseState 解析状态枚举 - 状态机核心
type ParseState int

const (
	StateNormal          ParseState = iota // 普通状态：处理顶级语句
	StateInFunction                        // 函数定义状态
	StateInAsyncFunction                   // 异步函数定义状态
	StateInIf                              // if语句状态
	StateInWhile                           // while语句状态
	StateInFor                             // for语句状态
	StateInStruct                          // 结构体定义状态
	StateInMethod                          // 方法定义状态
	StateInEnum                            // 枚举定义状态
	StateInTrait                           // trait定义状态
	StateInImpl                            // impl定义状态
	StateInMatch                           // match语句状态
	StateInSelect                          // select语句状态
)

// String 返回状态的字符串表示
func (ps ParseState) String() string {
	switch ps {
	case StateNormal:
		return "Normal"
	case StateInFunction:
		return "InFunction"
	case StateInAsyncFunction:
		return "InAsyncFunction"
	case StateInIf:
		return "InIf"
	case StateInWhile:
		return "InWhile"
	case StateInFor:
		return "InFor"
	case StateInStruct:
		return "InStruct"
	case StateInMethod:
		return "InMethod"
	case StateInEnum:
		return "InEnum"
	case StateInTrait:
		return "InTrait"
	case StateInImpl:
		return "InImpl"
	case StateInMatch:
		return "InMatch"
	case StateInSelect:
		return "InSelect"
	default:
		return "Unknown"
	}
}

// ParserAggregate 语法解析器聚合根
// 职责：协调整个解析过程，维护解析一致性边界
type ParserAggregate struct {
	id                         string
	statementParser            *StatementParser
	expressionParser           *ExpressionParser
	tokenBasedStatementParser  *TokenBasedStatementParser
	tokenBasedExpressionParser *TokenBasedExpressionParser
	lexerService               *lexicalServices.AdvancedLexerService
	blockExtractor             *BlockExtractor
	eventPublisher             *EventPublisher

	currentProgram      *entities.Program
	parseState          ParseState // 状态机当前状态
	inFunctionBody      bool
	inIfBody            bool
	inWhileBody         bool
	inForBody           bool
	inStructBody        bool
	inMethodBody        bool
	inEnumBody          bool
	inTraitBody         bool
	inImplBody          bool
	inMatchBody         bool
	inAsyncFunctionBody bool
	inSelectBody        bool

	parsingElse       bool
	thenBranchEnded   bool
	currentFunction   *entities.FuncDef
	currentIfStmt     *entities.IfStmt
	currentWhileStmt  *entities.WhileStmt
	currentForStmt    *entities.ForStmt
	currentStructDef  *entities.StructDef
	currentMethodDef  *entities.MethodDef
	currentEnumDef    *entities.EnumDef
	currentTraitDef   *entities.TraitDef
	currentImplDef    *entities.ImplDef
	currentMatchStmt  *entities.MatchStmt
	currentSelectStmt *entities.SelectStmt
	currentAsyncFunc  *entities.AsyncFuncDef

	ifBodyLines        []string
	whileBodyLines     []string
	forBodyLines       []string
	structBodyLines    []string
	methodBodyLines    []string
	enumBodyLines      []string
	traitBodyLines     []string
	implBodyLines      []string
	matchBodyLines     []string
	asyncFuncBodyLines []string
	selectBodyLines    []string
}

// NewParserAggregate 创建新的Parser聚合根
func NewParserAggregate() *ParserAggregate {
	blockExtractor := NewBlockExtractor()
	eventPublisher := NewEventPublisher(nil) // TODO: 注入事件总线

	parser := &ParserAggregate{
		id:             generateParserID(),
		blockExtractor: blockExtractor,
		eventPublisher: eventPublisher,
		lexerService:   lexicalServices.NewAdvancedLexerService(),
		currentProgram: &entities.Program{
			Statements: []entities.ASTNode{},
		},
	}

	// 设置BlockExtractor的解析函数，使其能够与StatementParser协作
	// 包装parseBlock方法以匹配SetParseBlockFunc的签名
	blockExtractor.SetParseBlockFunc(func(lines []string, startLineNum int) ([]entities.ASTNode, error) {
		return parser.parseBlock(lines)
	})

	// 创建领域服务，传入parser引用以支持递归解析
	parser.statementParser = NewStatementParser(parser, blockExtractor)
	parser.expressionParser = NewExpressionParser(parser)

	// 创建基于 Token 的解析器
	parser.tokenBasedStatementParser = NewTokenBasedStatementParser()
	parser.tokenBasedExpressionParser = NewTokenBasedExpressionParser()

	return parser
}

// ParseProgram 主入口方法：将源码字符串转换为AST程序
// 使用基于 Token 的解析（新实现）
func (p *ParserAggregate) ParseProgram(sourceCode string) (*entities.Program, error) {
	// 使用基于 Token 的解析
	return p.ParseProgramWithTokens(sourceCode, "main.eo")
}

// ParseProgramWithTokens 基于 Token 的解析方法
func (p *ParserAggregate) ParseProgramWithTokens(sourceCode string, filename string) (*entities.Program, error) {
	// 初始化程序
	p.currentProgram = &entities.Program{
		Statements: []entities.ASTNode{},
	}

	// 步骤1：词法分析 - 将源码转换为 Token 流
	ctx := context.Background()
	sourceFile := lexicalVO.NewSourceFile(filename, sourceCode)
	tokenStream, err := p.lexerService.Tokenize(ctx, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("lexical analysis failed: %w", err)
	}

	// 步骤2：语法分析 - 使用 Token 流解析语句
	statements, err := p.parseStatementsFromTokens(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("syntax analysis failed: %w", err)
	}

	p.currentProgram.Statements = statements
	return p.currentProgram, nil
}

// parseStatementsFromTokens 从 Token 流解析语句列表
func (p *ParserAggregate) parseStatementsFromTokens(tokenStream *lexicalVO.EnhancedTokenStream) ([]entities.ASTNode, error) {
	var statements []entities.ASTNode

	// 重置 token stream 位置
	tokenStream.Reset()

	// 循环解析语句，直到遇到 EOF
	for !tokenStream.IsAtEnd() {
		currentToken := tokenStream.Current()
		if currentToken == nil {
			break
		}

		// 跳过 EOF token
		if currentToken.Type() == lexicalVO.EnhancedTokenTypeEOF {
			break
		}

		// 跳过分号（如果 Echo 语言使用分号作为语句结束符，它们可能出现在语句之间）
		if currentToken.Type() == lexicalVO.EnhancedTokenTypeSemicolon {
			tokenStream.Next()
			continue
		}

		// 解析一个语句
		stmt, err := p.tokenBasedStatementParser.ParseStatement(tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse statement at token %s (line %d): %w",
				currentToken.Type(), currentToken.Location().Line, err)
		}

		if stmt != nil {
			statements = append(statements, stmt)
		}

		// 如果当前 token 是分号，消耗它（可选的分号）
		if !tokenStream.IsAtEnd() {
			nextToken := tokenStream.Current()
			if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeSemicolon {
				tokenStream.Next()
			}
		}
	}

	return statements, nil
}

// ParseProgramLegacy 旧版基于行的解析方法（保留用于兼容性）
func (p *ParserAggregate) ParseProgramLegacy(sourceCode string) (*entities.Program, error) {
	// 初始化程序
	p.currentProgram = &entities.Program{
		Statements: []entities.ASTNode{},
	}

	// 按行分割源码
	lines := strings.Split(sourceCode, "\n")

	// 主解析循环 - 状态机实现
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// 根据当前状态处理不同类型的块
		if p.inStructBody {
			if err := p.processStructBody(lines, &i); err != nil {
				return nil, err
			}
			continue // 重要：处理完块内容后继续下一行，不要再调用ParseStatement
		} else if p.inEnumBody {
			if err := p.processEnumBody(lines, &i); err != nil {
				return nil, err
			}
			continue // 重要：处理完块内容后继续下一行
		} else if p.inTraitBody {
			if err := p.processTraitBody(lines, &i); err != nil {
				return nil, err
			}
			continue // 重要：处理完块内容后继续下一行
		} else if p.inImplBody {
			if err := p.processImplBody(lines, &i); err != nil {
				return nil, err
			}
			continue // 重要：处理完块内容后继续下一行
		} else if p.inMatchBody {
			if err := p.processMatchBody(lines, &i); err != nil {
				return nil, err
			}
			continue // 重要：处理完块内容后继续下一行
		} else if p.inSelectBody {
			if err := p.processSelectBody(lines, &i); err != nil {
				return nil, err
			}
			continue // 重要：处理完块内容后继续下一行
		} else if p.parseState == StateInFunction || p.parseState == StateInAsyncFunction {
			// 使用状态机处理函数体
			if err := p.processFunctionState(lines, &i); err != nil {
				return nil, err
			}
			continue
		} else if p.inIfBody || p.inWhileBody || p.inForBody {
			// 在其他块结构中，跳过内容（暂时简化处理）
			if line == "}" {
				// 块结束，重置状态
				p.inIfBody = false
				p.inWhileBody = false
				p.inForBody = false
			}
			continue
		}

		// 处理顶级语句 - 委托给StatementParser领域服务
		stmt, err := p.statementParser.ParseStatement(lines[i], i+1)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}

		// 处理块结构的开始
		if _, ok := stmt.(*entities.IfStmt); ok {
			p.inIfBody = true
			// 暂时跳过body处理
		} else if _, ok := stmt.(*entities.FuncDef); ok {
			p.inFunctionBody = true
			// 暂时跳过body处理
		} else if structDef, ok := stmt.(*entities.StructDef); ok {
			p.inStructBody = true
			p.currentStructDef = structDef
			// 继续处理结构体字段
		} else if enumDef, ok := stmt.(*entities.EnumDef); ok {
			p.inEnumBody = true
			p.currentEnumDef = enumDef
			// 继续处理枚举变体
		} else if traitDef, ok := stmt.(*entities.TraitDef); ok {
			p.inTraitBody = true
			p.currentTraitDef = traitDef
			// 继续处理trait方法
		} else if implDef, ok := stmt.(*entities.ImplDef); ok {
			p.inImplBody = true
			p.currentImplDef = implDef
			// 继续处理impl方法
		} else if matchStmt, ok := stmt.(*entities.MatchStmt); ok {
			p.inMatchBody = true
			p.currentMatchStmt = matchStmt
			// 继续处理match case
		} else if selectStmt, ok := stmt.(*entities.SelectStmt); ok {
			p.inSelectBody = true
			p.currentSelectStmt = selectStmt
			// 继续处理select case
		}

		// 添加有效的语句到程序中（EnumDef除外，它会在枚举结束时添加）
		if stmt != nil {
			if _, isEnumDef := stmt.(*entities.EnumDef); !isEnumDef {
				p.currentProgram.Statements = append(p.currentProgram.Statements, stmt)
			}
		}
	}

	return p.currentProgram, nil
}

// processLineByState 状态机核心：根据当前状态处理行
func (p *ParserAggregate) processLineByState(lines []string, i *int) error {
	switch p.parseState {
	case StateNormal:
		return p.processNormalState(lines, i)
	case StateInStruct:
		return p.processStructState(lines, i)
	case StateInEnum:
		return p.processEnumState(lines, i)
	case StateInTrait:
		return p.processTraitState(lines, i)
	case StateInImpl:
		return p.processImplState(lines, i)
	case StateInMatch:
		return p.processMatchState(lines, i)
	case StateInSelect:
		return p.processSelectState(lines, i)
	case StateInFunction:
		return p.processFunctionState(lines, i)
	case StateInIf:
		return p.processIfState(lines, i)
	case StateInWhile:
		return p.processWhileState(lines, i)
	case StateInFor:
		return p.processForState(lines, i)
	case StateInMethod:
		return p.processMethodState(lines, i)
	case StateInAsyncFunction:
		return p.processAsyncFunctionState(lines, i)
	default:
		return fmt.Errorf("unknown parse state: %v", p.parseState)
	}
}

// processNormalState 处理普通状态（顶级语句）
func (p *ParserAggregate) processNormalState(lines []string, i *int) error {
	oldState := p.parseState

	// 解析顶级语句
	stmt, err := p.statementParser.ParseStatement(lines[*i], *i+1)
	if err != nil {
		return err
	}

	// 根据语句类型转换到相应的状态
	if stmt != nil {
		if err := p.handleStatementTransition(stmt); err != nil {
			return err
		}

		// 如果状态发生了变化，说明这是一个需要特殊处理的语句
		// 不要添加到程序，等待状态完成时再添加
		if p.parseState != oldState {
			return nil // 不添加到程序，等待状态完成
		}

		// 普通语句直接添加到程序
		p.currentProgram.Statements = append(p.currentProgram.Statements, stmt)
	}

	return nil
}

// handleStatementTransition 处理语句状态转换
func (p *ParserAggregate) handleStatementTransition(stmt entities.ASTNode) error {
	switch stmt.(type) {
	case *entities.StructDef:
		p.parseState = StateInStruct
		p.currentStructDef = stmt.(*entities.StructDef)
	case *entities.EnumDef:
		p.parseState = StateInEnum
		p.currentEnumDef = stmt.(*entities.EnumDef)
	case *entities.TraitDef:
		p.parseState = StateInTrait
		p.currentTraitDef = stmt.(*entities.TraitDef)
	case *entities.ImplDef:
		p.parseState = StateInImpl
		p.currentImplDef = stmt.(*entities.ImplDef)
	case *entities.MatchStmt:
		p.parseState = StateInMatch
		p.currentMatchStmt = stmt.(*entities.MatchStmt)
	case *entities.SelectStmt:
		p.parseState = StateInSelect
		p.currentSelectStmt = stmt.(*entities.SelectStmt)
	case *entities.FuncDef:
		funcDef := stmt.(*entities.FuncDef)
		fmt.Printf("DEBUG: Handling FuncDef: %s, IsAsync: %v\n", funcDef.Name, funcDef.IsAsync)
		if funcDef.IsAsync {
			p.parseState = StateInAsyncFunction
		} else {
			p.parseState = StateInFunction
		}
		p.currentFunction = funcDef
		fmt.Printf("DEBUG: Set currentFunction to: %s, state: %v\n", p.currentFunction.Name, p.parseState)
	case *entities.IfStmt:
		p.parseState = StateInIf
		p.currentIfStmt = stmt.(*entities.IfStmt)
	case *entities.WhileStmt:
		p.parseState = StateInWhile
		p.currentWhileStmt = stmt.(*entities.WhileStmt)
	case *entities.ForStmt:
		p.parseState = StateInFor
		p.currentForStmt = stmt.(*entities.ForStmt)
	}
	return nil
}

// processStructState 处理结构体状态
func (p *ParserAggregate) processStructState(lines []string, i *int) error {
	return p.processStructBody(lines, i)
}

// processEnumState 处理枚举状态
func (p *ParserAggregate) processEnumState(lines []string, i *int) error {
	return p.processEnumBody(lines, i)
}

// processTraitState 处理trait状态
func (p *ParserAggregate) processTraitState(lines []string, i *int) error {
	return p.processTraitBody(lines, i)
}

// processImplState 处理impl状态
func (p *ParserAggregate) processImplState(lines []string, i *int) error {
	return p.processImplBody(lines, i)
}

// processMatchState 处理match状态
func (p *ParserAggregate) processMatchState(lines []string, i *int) error {
	return p.processMatchBody(lines, i)
}

// processSelectState 处理select状态
func (p *ParserAggregate) processSelectState(lines []string, i *int) error {
	return p.processSelectBody(lines, i)
}

// processFunctionState 处理函数状态
func (p *ParserAggregate) processFunctionState(lines []string, i *int) error {
	return p.processFunctionBody(lines, i)
}

// processAsyncFunctionState 处理异步函数状态
func (p *ParserAggregate) processAsyncFunctionState(lines []string, i *int) error {
	return p.processAsyncFunctionBody(lines, i)
}

// processIfState 处理if状态
func (p *ParserAggregate) processIfState(lines []string, i *int) error {
	return p.processIfBody(lines, i)
}

// processWhileState 处理while状态
func (p *ParserAggregate) processWhileState(lines []string, i *int) error {
	return p.processWhileBody(lines, i)
}

// processForState 处理for状态
func (p *ParserAggregate) processForState(lines []string, i *int) error {
	return p.processForBody(lines, i)
}

// processMethodState 处理方法状态
func (p *ParserAggregate) processMethodState(lines []string, i *int) error {
	return p.processMethodBody(lines, i)
}

// resetParseState 重置解析状态
func (p *ParserAggregate) resetParseState() {
	p.parseState = StateNormal
	p.inFunctionBody = false
	p.inIfBody = false
	p.inWhileBody = false
	p.inForBody = false
	p.inStructBody = false
	p.inMethodBody = false
	p.inEnumBody = false
	p.inTraitBody = false
	p.inImplBody = false
	p.inMatchBody = false
	p.inAsyncFunctionBody = false
	p.inSelectBody = false

	p.parsingElse = false
	p.thenBranchEnded = false

	p.currentFunction = nil
	p.currentIfStmt = nil
	p.currentWhileStmt = nil
	p.currentForStmt = nil
	p.currentStructDef = nil
	p.currentMethodDef = nil
	p.currentEnumDef = nil
	p.currentTraitDef = nil
	p.currentImplDef = nil
	p.currentMatchStmt = nil
	p.currentSelectStmt = nil
	p.currentAsyncFunc = nil

	p.ifBodyLines = []string{}
	p.whileBodyLines = []string{}
	p.forBodyLines = []string{}
	p.structBodyLines = []string{}
	p.methodBodyLines = []string{}
	p.enumBodyLines = []string{}
	p.traitBodyLines = []string{}
	p.implBodyLines = []string{}
	p.matchBodyLines = []string{}
	p.asyncFuncBodyLines = []string{}
	p.selectBodyLines = []string{}
}

// handleElseAfterThen 处理then分支结束后可能的else
func (p *ParserAggregate) handleElseAfterThen(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if strings.TrimSpace(line) == "else" {
		// 处理独立行的else关键字
		p.parsingElse = true
		p.inIfBody = true
		p.ifBodyLines = []string{}
		return nil
	}

	if strings.Contains(line, "} else if ") {
		// 处理同一行的 } else if condition {
		return p.processNestedElseIf(line, i)
	}

	if strings.Contains(line, "} else {") {
		// 处理同一行的 } else {
		return p.processElseBlock(line, i)
	}

	if strings.TrimSpace(line) == "else" {
		// 处理换行格式的else
		p.parsingElse = true
		p.inIfBody = true
		p.ifBodyLines = []string{}
		return nil
	}

	// 不是else相关内容，重置状态
	p.thenBranchEnded = false
	p.parsingElse = false
	p.inIfBody = false

	// 重新处理当前行
	*i--
	return nil
}

// processNestedElseIf 处理嵌套的else if
func (p *ParserAggregate) processNestedElseIf(line string, i *int) error {
	// 提取else if条件
	elseIfPattern := "} else if "
	start := strings.Index(line, elseIfPattern)
	if start == -1 {
		return fmt.Errorf("line %d: invalid else if syntax", *i+1)
	}

	conditionStart := start + len(elseIfPattern)
	braceIndex := strings.Index(line[conditionStart:], " {")
	if braceIndex == -1 {
		return fmt.Errorf("line %d: invalid else if syntax, missing opening brace", *i+1)
	}

	conditionStr := strings.TrimSpace(line[conditionStart : conditionStart+braceIndex])

	// 解析else if条件
	condition, err := p.expressionParser.ParseExpr(conditionStr)
	if err != nil {
		return fmt.Errorf("line %d: invalid else if condition: %v", *i+1, err)
	}

	// 创建嵌套if语句
	nestedIf := &entities.IfStmt{
		Condition: condition,
		ThenBody:  []entities.ASTNode{},
		ElseBody:  []entities.ASTNode{},
	}

	// 添加到当前if的else分支
	if p.currentIfStmt != nil {
		p.currentIfStmt.ElseBody = append(p.currentIfStmt.ElseBody, nestedIf)
	}

	// 切换到嵌套if
	p.currentIfStmt = nestedIf
	p.parsingElse = false
	p.ifBodyLines = []string{}

	return nil
}

// processElseBlock 处理else块
func (p *ParserAggregate) processElseBlock(line string, i *int) error {
	p.parsingElse = true
	p.ifBodyLines = []string{}
	return nil
}

// processTopLevelStatement 处理顶级语句
func (p *ParserAggregate) processTopLevelStatement(line string, lineNum int) error {
	stmt, err := p.statementParser.ParseStatement(line, lineNum)
	if err != nil {
		return err
	}

	if stmt != nil {
		p.currentProgram.Statements = append(p.currentProgram.Statements, stmt)
		p.publishStatementParsedEvent(stmt, lineNum)
	}

	return nil
}

// processIfBody 处理if语句体
func (p *ParserAggregate) processIfBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		if p.parsingElse {
			// 当前在else分支内，else分支结束
			if err := p.finalizeElseBranch(); err != nil {
				return err
			}
		} else {
			// then分支结束，等待可能的else
			if err := p.finalizeThenBranch(); err != nil {
				return err
			}
		}
		return nil
	}

	if strings.Contains(line, "} else if ") {
		// 处理同一行的 } else if condition {
		return p.processNestedElseIf(line, i)
	}

	if strings.Contains(line, "} else {") {
		// 处理同一行的 } else {
		return p.processElseBlock(line, i)
	}

	// 收集语句行
	p.ifBodyLines = append(p.ifBodyLines, lines[*i])
	return nil
}

// finalizeThenBranch 完成then分支处理
func (p *ParserAggregate) finalizeThenBranch() error {
	// 解析then分支内容
	thenBody, err := p.parseBlock(p.ifBodyLines)
	if err != nil {
		return err
	}

	if p.currentIfStmt != nil {
		p.currentIfStmt.ThenBody = thenBody
	}

	p.thenBranchEnded = true
	p.parsingElse = false
	p.ifBodyLines = []string{}

	return nil
}

// finalizeElseBranch 完成else分支处理
func (p *ParserAggregate) finalizeElseBranch() error {
	// 解析else分支内容
	elseBody, err := p.parseBlock(p.ifBodyLines)
	if err != nil {
		return err
	}

	if p.currentIfStmt != nil {
		p.currentIfStmt.ElseBody = elseBody
	}

	// 添加完整的if语句到程序
	if p.currentIfStmt != nil {
		p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentIfStmt)
	}

	// 重置状态
	p.resetIfState()
	return nil
}

// resetIfState 重置if相关状态
func (p *ParserAggregate) resetIfState() {
	p.inIfBody = false
	p.parsingElse = false
	p.thenBranchEnded = false
	p.currentIfStmt = nil
	p.ifBodyLines = []string{}
}

// processFunctionBody 处理函数体
func (p *ParserAggregate) processFunctionBody(lines []string, i *int) error {
	// 类似if语句体的处理，但针对函数
	line := strings.TrimSpace(lines[*i])

	// 跳过开头的 "{"
	if line == "{" {
		return nil
	}

	if line == "}" {
		return p.finalizeFunction()
	}

	p.methodBodyLines = append(p.methodBodyLines, lines[*i])
	return nil
}

// finalizeFunction 完成函数处理
func (p *ParserAggregate) finalizeFunction() error {
	if p.currentFunction == nil {
		return fmt.Errorf("no current function to finalize")
	}

	body, err := p.parseBlock(p.methodBodyLines)
	if err != nil {
		return err
	}

	p.currentFunction.Body = body
	p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentFunction)

	p.resetFunctionState()
	return nil
}

// resetFunctionState 重置函数相关状态
func (p *ParserAggregate) resetFunctionState() {
	p.inFunctionBody = false
	p.currentFunction = nil
	p.methodBodyLines = []string{}
}

// processWhileBody 处理while循环体
func (p *ParserAggregate) processWhileBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		return p.finalizeWhileLoop()
	}

	p.whileBodyLines = append(p.whileBodyLines, lines[*i])
	return nil
}

// finalizeWhileLoop 完成while循环处理
func (p *ParserAggregate) finalizeWhileLoop() error {
	if p.currentWhileStmt == nil {
		return fmt.Errorf("no current while statement to finalize")
	}

	body, err := p.parseBlock(p.whileBodyLines)
	if err != nil {
		return err
	}

	p.currentWhileStmt.Body = body
	p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentWhileStmt)

	p.resetWhileState()
	return nil
}

// resetWhileState 重置while相关状态
func (p *ParserAggregate) resetWhileState() {
	p.inWhileBody = false
	p.currentWhileStmt = nil
	p.whileBodyLines = []string{}
}

// processForBody 处理for循环体
func (p *ParserAggregate) processForBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		return p.finalizeForLoop()
	}

	p.forBodyLines = append(p.forBodyLines, lines[*i])
	return nil
}

// finalizeForLoop 完成for循环处理
func (p *ParserAggregate) finalizeForLoop() error {
	if p.currentForStmt == nil {
		return fmt.Errorf("no current for statement to finalize")
	}

	body, err := p.parseBlock(p.forBodyLines)
	if err != nil {
		return err
	}

	p.currentForStmt.Body = body
	p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentForStmt)

	p.resetForState()
	return nil
}

// resetForState 重置for相关状态
func (p *ParserAggregate) resetForState() {
	p.inForBody = false
	p.currentForStmt = nil
	p.forBodyLines = []string{}
}

// processStructBody 处理结构体体
func (p *ParserAggregate) processStructBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])
	fmt.Printf("DEBUG: processStructBody processing: '%s'\n", line)

	if line == "}" {
		fmt.Printf("DEBUG: Struct definition completed: %s with %d fields\n", p.currentStructDef.Name, len(p.currentStructDef.Fields))
		p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentStructDef)
		p.resetStructState()
		return nil
	}

	// 解析字段定义，如 "x: int,"
	if strings.Contains(line, ":") && strings.HasSuffix(line, ",") {
		fieldLine := strings.TrimSpace(strings.TrimSuffix(line, ","))
		colonIndex := strings.Index(fieldLine, ":")
		if colonIndex != -1 {
			fieldName := strings.TrimSpace(fieldLine[:colonIndex])
			fieldType := strings.TrimSpace(fieldLine[colonIndex+1:])

			field := entities.StructField{
				Name: fieldName,
				Type: fieldType,
			}
			p.currentStructDef.Fields = append(p.currentStructDef.Fields, field)
			fmt.Printf("DEBUG: Added field: %s %s\n", fieldName, fieldType)
		}
	}

	return nil
}

// resetStructState 重置结构体相关状态
func (p *ParserAggregate) resetStructState() {
	p.inStructBody = false
	p.currentStructDef = nil
	p.structBodyLines = []string{}
}

// processMethodBody 处理方法体
func (p *ParserAggregate) processMethodBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		return p.finalizeMethod()
	}

	p.methodBodyLines = append(p.methodBodyLines, lines[*i])
	return nil
}

// finalizeMethod 完成方法处理
func (p *ParserAggregate) finalizeMethod() error {
	if p.currentMethodDef == nil {
		return fmt.Errorf("no current method to finalize")
	}

	body, err := p.parseBlock(p.methodBodyLines)
	if err != nil {
		return err
	}

	p.currentMethodDef.Body = body
	p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentMethodDef)

	p.resetMethodState()
	return nil
}

// resetMethodState 重置方法相关状态
func (p *ParserAggregate) resetMethodState() {
	p.inMethodBody = false
	p.currentMethodDef = nil
	p.methodBodyLines = []string{}
}

// processEnumBody 处理枚举体
func (p *ParserAggregate) processEnumBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentEnumDef)
		p.resetEnumState()
		return nil
	}

	// 解析变体，如 "Red,"
	if strings.HasSuffix(line, ",") {
		variantName := strings.TrimSpace(strings.TrimSuffix(line, ","))
		if variantName != "" {
			variant := entities.EnumVariant{
				Name: variantName,
			}
			p.currentEnumDef.Variants = append(p.currentEnumDef.Variants, variant)
		}
	}

	return nil
}

// processTraitBody 处理trait体
func (p *ParserAggregate) processTraitBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentTraitDef)
		p.resetTraitState()
		return nil
	}

	// 暂时简化处理：跳过trait方法定义
	return nil
}

// processImplBody 处理impl体
func (p *ParserAggregate) processImplBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentImplDef)
		p.resetImplState()
		return nil
	}

	// 暂时简化处理：跳过impl方法定义
	return nil
}

// resetTraitState 重置trait相关状态
func (p *ParserAggregate) resetTraitState() {
	p.inTraitBody = false
	p.currentTraitDef = nil
	p.traitBodyLines = []string{}
}

// resetImplState 重置impl相关状态
func (p *ParserAggregate) resetImplState() {
	p.inImplBody = false
	p.currentImplDef = nil
	p.implBodyLines = []string{}
}

// resetEnumState 重置枚举相关状态
func (p *ParserAggregate) resetEnumState() {
	p.parseState = StateNormal // 重置解析状态
	p.inEnumBody = false
	p.currentEnumDef = nil
	p.enumBodyLines = []string{}
}

// processMatchBody 处理match语句体
func (p *ParserAggregate) processMatchBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		// match体结束，解析所有case
		if err := p.parseMatchCases(); err != nil {
			return fmt.Errorf("line %d: error parsing match cases: %w", *i+1, err)
		}
		p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentMatchStmt)
		p.resetMatchState()
		return nil
	}

	p.matchBodyLines = append(p.matchBodyLines, lines[*i])
	return nil
}

// processSelectBody 处理select语句体
// parseSelectCases 解析select语句的所有case - 状态机实现
func (p *ParserAggregate) parseSelectCases() error {
	for _, caseLine := range p.selectBodyLines {
		caseLine = strings.TrimSpace(caseLine)
		if caseLine == "" {
			continue
		}

		// Select case状态机：
		// case recvExpr => statement     (接收操作)
		// case sendExpr <- value => statement  (发送操作)
		// default => statement          (默认分支)

		if strings.HasPrefix(caseLine, "case ") {
			// case分支 - 可能是接收或发送操作
			caseContent := strings.TrimPrefix(caseLine, "case ")
			arrowIndex := strings.Index(caseContent, " => ")
			if arrowIndex == -1 {
				return fmt.Errorf("invalid select case format: %s", caseLine)
			}

			caseExprStr := strings.TrimSpace(caseContent[:arrowIndex])
			bodyStr := strings.TrimSpace(caseContent[arrowIndex+4:])

			var selectCase entities.SelectCase

			// 判断是接收还是发送操作
			if strings.Contains(caseExprStr, " <- ") {
				// 发送操作: channel <- value
				parts := strings.Split(caseExprStr, " <- ")
				if len(parts) != 2 {
					return fmt.Errorf("invalid send case format: %s", caseExprStr)
				}

				chanExpr, err := p.expressionParser.ParseExpr(strings.TrimSpace(parts[0]))
				if err != nil {
					return fmt.Errorf("invalid channel expression in send case %s: %w", parts[0], err)
				}

				valueExpr, err := p.expressionParser.ParseExpr(strings.TrimSpace(parts[1]))
				if err != nil {
					return fmt.Errorf("invalid value expression in send case %s: %w", parts[1], err)
				}

				selectCase = entities.SelectCase{
					Chan:   chanExpr,
					Value:  valueExpr,
					Body:   []entities.ASTNode{},
					IsSend: true,
				}
			} else {
				// 接收操作: <- channel 或 value := <- channel
				// 检查是否有赋值操作 (value := <- channel)
				var receiveExpr entities.Expr
				var err error

				if strings.Contains(caseExprStr, ":=") {
					// 有赋值操作: value := <- channel
					parts := strings.Split(caseExprStr, ":=")
					if len(parts) != 2 {
						return fmt.Errorf("invalid receive case with assignment format: %s", caseExprStr)
					}
					// 解析接收表达式部分 (<- channel)
					receiveExprStr := strings.TrimSpace(parts[1])
					receiveExpr, err = p.expressionParser.ParseExpr(receiveExprStr)
					if err != nil {
						return fmt.Errorf("invalid receive expression in case %s: %w", receiveExprStr, err)
					}
					// 注意：变量绑定信息可以在后续语义分析阶段处理
					// 这里只解析接收表达式本身
				} else {
					// 简单接收: <- channel
					receiveExpr, err = p.expressionParser.ParseExpr(caseExprStr)
					if err != nil {
						return fmt.Errorf("invalid receive expression in case %s: %w", caseExprStr, err)
					}
				}

				// 验证解析结果是ReceiveExpr类型
				if _, ok := receiveExpr.(*entities.ReceiveExpr); !ok {
					return fmt.Errorf("expected receive expression, got %T in case: %s", receiveExpr, caseExprStr)
				}

				selectCase = entities.SelectCase{
					Chan:   receiveExpr,
					Value:  nil, // 接收操作没有发送值
					Body:   []entities.ASTNode{},
					IsSend: false,
				}
			}

			// 解析body语句
			bodyStmt, err := p.statementParser.ParseStatement(bodyStr, 0)
			if err != nil {
				return fmt.Errorf("invalid select case body %s: %w", bodyStr, err)
			}
			selectCase.Body = []entities.ASTNode{bodyStmt}

			p.currentSelectStmt.Cases = append(p.currentSelectStmt.Cases, selectCase)

		} else if strings.HasPrefix(caseLine, "default => ") {
			// default分支
			bodyStr := strings.TrimPrefix(caseLine, "default => ")

			// 解析body语句
			bodyStmt, err := p.statementParser.ParseStatement(bodyStr, 0)
			if err != nil {
				return fmt.Errorf("invalid select default body %s: %w", bodyStr, err)
			}

			// default分支存储在SelectStmt的DefaultBody中
			p.currentSelectStmt.DefaultBody = append(p.currentSelectStmt.DefaultBody, bodyStmt)

		} else {
			return fmt.Errorf("unknown select case format: %s", caseLine)
		}
	}

	return nil
}

// parseMatchCases 解析match语句的所有case
func (p *ParserAggregate) parseMatchCases() error {
	for _, caseLine := range p.matchBodyLines {
		caseLine = strings.TrimSpace(caseLine)
		if caseLine == "" {
			continue
		}

		// 解析case: pattern [if guard] => expression
		caseParts := strings.Split(caseLine, "=>")
		if len(caseParts) != 2 {
			return fmt.Errorf("invalid match case format: %s", caseLine)
		}

		patternPart := strings.TrimSpace(caseParts[0])
		expressionPart := strings.TrimSpace(caseParts[1])

		// 检查是否有guard条件 (if condition)
		var patternStr, guardStr string
		if strings.Contains(patternPart, " if ") {
			parts := strings.Split(patternPart, " if ")
			if len(parts) == 2 {
				patternStr = strings.TrimSpace(parts[0])
				guardStr = strings.TrimSpace(parts[1])
			} else {
				patternStr = patternPart
			}
		} else {
			patternStr = patternPart
		}

		// 解析pattern
		pattern, err := p.expressionParser.ParseExpr(patternStr)
		if err != nil {
			return fmt.Errorf("invalid match pattern %s: %w", patternStr, err)
		}

		// 解析guard (如果有)
		var guard entities.Expr
		if guardStr != "" {
			guard, err = p.expressionParser.ParseExpr(guardStr)
			if err != nil {
				return fmt.Errorf("invalid match guard %s: %w", guardStr, err)
			}
		}

		// 解析expression为语句
		stmt, err := p.statementParser.ParseStatement(expressionPart, 0) // 行号暂时设为0
		if err != nil {
			return fmt.Errorf("invalid match expression %s: %w", expressionPart, err)
		}

		// 创建MatchCase
		matchCase := entities.MatchCase{
			Pattern: pattern,
			Guard:   guard,
			Body:    []entities.ASTNode{stmt}, // case体可以是一个语句
		}

		p.currentMatchStmt.Cases = append(p.currentMatchStmt.Cases, matchCase)
	}

	return nil
}

// resetMatchState 重置match相关状态
func (p *ParserAggregate) resetMatchState() {
	p.inMatchBody = false
	p.currentMatchStmt = nil
	p.matchBodyLines = []string{}
}

// resetSelectState 重置select相关状态
func (p *ParserAggregate) resetSelectState() {
	p.inSelectBody = false
	p.currentSelectStmt = nil
	p.selectBodyLines = []string{}
}

// processAsyncFunctionBody 处理异步函数体
func (p *ParserAggregate) processAsyncFunctionBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		return p.finalizeAsyncFunction()
	}

	p.asyncFuncBodyLines = append(p.asyncFuncBodyLines, lines[*i])
	return nil
}

// finalizeAsyncFunction 完成异步函数处理
func (p *ParserAggregate) finalizeAsyncFunction() error {
	if p.currentAsyncFunc == nil {
		return fmt.Errorf("no current async function to finalize")
	}

	body, err := p.parseBlock(p.asyncFuncBodyLines)
	if err != nil {
		return err
	}

	p.currentAsyncFunc.Body = body
	p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentAsyncFunc)

	p.resetAsyncFunctionState()
	return nil
}

// resetAsyncFunctionState 重置异步函数相关状态
func (p *ParserAggregate) resetAsyncFunctionState() {
	p.inAsyncFunctionBody = false
	p.currentAsyncFunc = nil
	p.asyncFuncBodyLines = []string{}
}

// processSelectBody 处理select语句体
func (p *ParserAggregate) processSelectBody(lines []string, i *int) error {
	line := strings.TrimSpace(lines[*i])

	if line == "}" {
		// select体结束，解析所有case
		if err := p.finalizeSelectStatement(); err != nil {
			return fmt.Errorf("line %d: error parsing select cases: %w", *i+1, err)
		}
		p.exitSelectState()
		return nil
	}

	// 收集select体内的行，稍后解析
	p.selectBodyLines = append(p.selectBodyLines, lines[*i])
	return nil
}

// finalizeSelectStatement 完成select语句的解析
func (p *ParserAggregate) finalizeSelectStatement() error {
	if p.currentSelectStmt == nil {
		return fmt.Errorf("no current select statement")
	}

	// 解析所有case分支
	if err := p.parseSelectCases(); err != nil {
		return err
	}

	// 将完成的select语句添加到程序中
	p.currentProgram.Statements = append(p.currentProgram.Statements, p.currentSelectStmt)
	return nil
}

// exitSelectState 退出select状态
func (p *ParserAggregate) exitSelectState() {
	p.parseState = StateNormal
	p.inSelectBody = false
	p.currentSelectStmt = nil
	p.selectBodyLines = []string{}
}

// parseBlock 解析语句块 - 使用基于Token的解析器
func (p *ParserAggregate) parseBlock(lines []string) ([]entities.ASTNode, error) {
	var statements []entities.ASTNode

	// 将字符串数组合并为完整的源代码
	blockSource := strings.Join(lines, "\n")
	if blockSource == "" {
		return statements, nil
	}

	// 创建临时的源文件对象
	sourceFile := lexicalVO.NewSourceFile("block", blockSource)

	// 使用词法分析器将源代码转换为Token流
	ctx := context.Background()
	tokenStream, err := p.lexerService.Tokenize(ctx, sourceFile)
	if err != nil {
		// 如果词法分析失败，回退到旧的基于字符串的解析器
		return p.parseBlockFallback(lines)
	}

	// 使用基于Token的语句解析器解析语句块
	for !tokenStream.IsAtEnd() {
		stmt, err := p.tokenBasedStatementParser.ParseStatement(tokenStream)
		if err != nil {
			// 如果解析失败，尝试回退到旧的解析器
			return p.parseBlockFallback(lines)
		}

		if stmt != nil {
			statements = append(statements, stmt)
		} else {
			// 如果返回nil，可能是到达了块结束符，继续处理
			if tokenStream.IsAtEnd() {
				break
			}
			// 跳过当前token，继续解析
			tokenStream.Next()
		}
	}

	return statements, nil
}

// parseBlockFallback 回退到旧的基于字符串的解析器
func (p *ParserAggregate) parseBlockFallback(lines []string) ([]entities.ASTNode, error) {
	var statements []entities.ASTNode

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		stmt, err := p.statementParser.ParseStatement(line, i+1)
		if err != nil {
			return nil, err
		}

		if stmt != nil {
			statements = append(statements, stmt)
		}
	}

	return statements, nil
}

// validateParseResult 验证解析结果的完整性
func (p *ParserAggregate) validateParseResult() error {
	// 检查是否有未关闭的块
	if p.inFunctionBody || p.inIfBody || p.inWhileBody || p.inForBody ||
		p.inStructBody || p.inMethodBody || p.inEnumBody || p.inImplBody ||
		p.inMatchBody || p.inAsyncFunctionBody || p.inSelectBody {
		return fmt.Errorf("unclosed block detected")
	}

	// 检查程序结构完整性
	if len(p.currentProgram.Statements) == 0 {
		return fmt.Errorf("empty program")
	}

	return nil
}

// publishStatementParsedEvent 发布语句解析完成事件
func (p *ParserAggregate) publishStatementParsedEvent(stmt entities.ASTNode, lineNum int) {
	var stmtType string
	var hasExpressions bool
	var complexity int

	switch s := stmt.(type) {
	case *entities.IfStmt:
		stmtType = "if"
		hasExpressions = true
		complexity = 2
		if len(s.ElseBody) > 0 {
			complexity += 1
		}
	case *entities.WhileStmt:
		stmtType = "while"
		hasExpressions = true
		complexity = 3
	case *entities.ForStmt:
		stmtType = "for"
		hasExpressions = true
		complexity = 3
	case *entities.FuncDef:
		stmtType = "function"
		hasExpressions = false
		complexity = len(s.Params) + 2
	case *entities.VarDecl:
		stmtType = "variable_declaration"
		hasExpressions = s.Value != nil
		complexity = 1
	case *entities.AssignStmt:
		stmtType = "assignment"
		hasExpressions = true
		complexity = 1
	case *entities.PrintStmt:
		stmtType = "print"
		hasExpressions = true
		complexity = 1
	case *entities.ReturnStmt:
		stmtType = "return"
		hasExpressions = s.Value != nil
		complexity = 1
	default:
		stmtType = "other"
		hasExpressions = false
		complexity = 1
	}

	p.eventPublisher.PublishStatementParsed("", stmtType, lineNum, hasExpressions, complexity)
}

// generateParserID 生成解析器ID
func generateParserID() string {
	return fmt.Sprintf("parser-%d", time.Now().UnixNano())
}
