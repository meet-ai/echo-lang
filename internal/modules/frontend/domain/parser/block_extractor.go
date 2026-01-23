package parser

import (
	"fmt"
	"strings"

	"echo/internal/modules/frontend/domain/entities"
)

// BlockExtractor 代码块提取领域服务
// 职责：从源码行序列中提取各种类型的代码块，处理嵌套结构
type BlockExtractor struct {
	// parseBlockFunc 解析块内容的函数（由调用者注入，用于与StatementParser协作）
	// 参数：块的行列表，起始行号
	// 返回：解析后的AST节点列表，错误
	parseBlockFunc func(lines []string, startLineNum int) ([]entities.ASTNode, error)
}

// NewBlockExtractor 创建新的代码块提取器
func NewBlockExtractor() *BlockExtractor {
	return &BlockExtractor{
		parseBlockFunc: nil, // 默认不设置，由调用者注入
	}
}

// SetParseBlockFunc 设置解析块内容的函数（用于与StatementParser协作）
func (be *BlockExtractor) SetParseBlockFunc(parseFunc func(lines []string, startLineNum int) ([]entities.ASTNode, error)) {
	be.parseBlockFunc = parseFunc
}

// ExtractIfBlock 提取if语句的then分支 - 领域服务核心方法
func (be *BlockExtractor) ExtractIfBlock(lines []string, startLineNum int) ([]string, int, error) {
	var blockLines []string
	braceCount := 0

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
				return blockLines, len(blockLines) + 1, nil
			} else {
				blockLines = append(blockLines, lines[i])
			}
		} else if braceCount > 0 {
			blockLines = append(blockLines, lines[i])
		} else if strings.HasPrefix(line, "} else") {
			// 遇到else，then分支结束
			return blockLines, len(blockLines) + 1, nil
		} else if strings.HasPrefix(line, "else") {
			// 遇到else，then分支结束
			return blockLines, len(blockLines) + 1, nil
		}
	}

	// 如果没有找到结束符，说明块不完整
	return blockLines, len(blockLines), fmt.Errorf("block not properly closed")
}

// ExtractElseBlock 提取else分支内容 - 领域服务核心方法
func (be *BlockExtractor) ExtractElseBlock(lines []string, startLineNum int) ([]string, int, error) {
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
					return blockLines, len(blockLines) + 2, nil // +2 for "else {" and closing "}"
				}
			} else if braceCount > 0 {
				blockLines = append(blockLines, lines[i])
			}
		}
	}

	return blockLines, len(blockLines) + 2, nil
}

// ExtractNestedIf 处理嵌套的if语句 - 领域服务方法
func (be *BlockExtractor) ExtractNestedIf(ifStmt **entities.IfStmt, lines []string, startLineNum int) (int, error) {
	// 递归处理嵌套if语句的then和else分支
	nestedConsumed := 0

	// 1. 提取then分支
	thenLines, consumed, err := be.ExtractIfBlock(lines, startLineNum)
	if err != nil {
		return 0, fmt.Errorf("line %d: failed to extract if block: %w", startLineNum, err)
	}
	if consumed == 0 {
		return 0, fmt.Errorf("line %d: incomplete nested if statement", startLineNum)
	}

	// 完善实现：与StatementParser协作来解析then分支内容
	if be.parseBlockFunc != nil && ifStmt != nil && *ifStmt != nil {
		thenBody, err := be.parseBlockFunc(thenLines, startLineNum)
		if err != nil {
			return 0, fmt.Errorf("line %d: failed to parse then block: %w", startLineNum, err)
		}
		(*ifStmt).ThenBody = thenBody
	}
	nestedConsumed += consumed

	// 检查是否有else分支
	if consumed < len(lines) {
		nextLine := strings.TrimSpace(lines[consumed])
		if strings.HasPrefix(nextLine, "else") || strings.HasPrefix(nextLine, "} else") {
		// 处理else分支，这里可能包含嵌套的if语句
		elseLines, elseConsumed, err := be.ExtractElseBlock(lines[consumed:], startLineNum+consumed)
		if err != nil {
			return 0, fmt.Errorf("line %d: failed to extract else block: %w", startLineNum+consumed, err)
		}
		// 完善实现：解析else分支内容
		if be.parseBlockFunc != nil && ifStmt != nil && *ifStmt != nil {
			elseBody, err := be.parseBlockFunc(elseLines, startLineNum+consumed)
			if err != nil {
				return 0, fmt.Errorf("line %d: failed to parse else block: %w", startLineNum+consumed, err)
			}
			(*ifStmt).ElseBody = elseBody
		}
		nestedConsumed += elseConsumed
		}
	}

	return nestedConsumed, nil
}

// ExtractFunctionBlock 提取函数体块
func (be *BlockExtractor) ExtractFunctionBlock(lines []string, startLineNum int) ([]string, int, error) {
	var blockLines []string
	braceCount := 0
	inFunctionBody := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "{") && !inFunctionBody {
			// 找到函数体的开始
			inFunctionBody = true
			braceCount++
		} else if inFunctionBody {
			if line == "{" {
				braceCount++
			} else if line == "}" {
				braceCount--
				if braceCount == 0 {
					// 函数体结束
					return blockLines, len(blockLines) + 1, nil
				}
			} else if braceCount > 0 {
				blockLines = append(blockLines, lines[i])
			}
		}
	}

	if inFunctionBody && braceCount > 0 {
		return nil, 0, fmt.Errorf("line %d: incomplete function block", startLineNum)
	}

	return blockLines, len(blockLines), nil
}

// ExtractWhileBlock 提取while循环体块
func (be *BlockExtractor) ExtractWhileBlock(lines []string, startLineNum int) ([]string, int, error) {
	// 与if块提取逻辑类似
	lines, consumed, _ := be.ExtractIfBlock(lines, startLineNum)
	return lines, consumed, nil
}

// ExtractForBlock 提取for循环体块
func (be *BlockExtractor) ExtractForBlock(lines []string, startLineNum int) ([]string, int, error) {
	// 与if块提取逻辑类似
	lines, consumed, _ := be.ExtractIfBlock(lines, startLineNum)
	return lines, consumed, nil
}

// ValidateBlockStructure 验证代码块结构完整性
func (be *BlockExtractor) ValidateBlockStructure(lines []string, startLineNum int) error {
	braceCount := 0
	inBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "{") {
			inBlock = true
			braceCount++
		}

		if inBlock {
			if line == "{" {
				braceCount++
			} else if line == "}" {
				braceCount--
			}
		}

		if braceCount == 0 && inBlock {
			// 块结束
			break
		}
	}

	if braceCount > 0 {
		return fmt.Errorf("line %d: unmatched braces in block", startLineNum)
	}

	return nil
}
