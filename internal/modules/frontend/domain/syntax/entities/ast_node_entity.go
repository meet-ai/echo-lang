// Package entities 定义语法分析上下文的实体
package entities

import (
	"fmt"
	"time"

	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// ASTNodeEntity AST节点实体（聚合根）
// ASTNodeEntity 是语法分析的核心实体，负责管理AST节点的生命周期
type ASTNodeEntity struct {
	id         string
	node       value_objects.ASTNode
	isValid    bool
	validatedAt *time.Time
	children   []*ASTNodeEntity
	parent     *ASTNodeEntity
}

// NewASTNodeEntity 创建新的AST节点实体
func NewASTNodeEntity(id string, node value_objects.ASTNode) *ASTNodeEntity {
	now := time.Now()
	return &ASTNodeEntity{
		id:          id,
		node:        node,
		isValid:     false,
		validatedAt: &now,
		children:    make([]*ASTNodeEntity, 0),
		parent:      nil,
	}
}

// ID 获取实体ID
func (ane *ASTNodeEntity) ID() string {
	return ane.id
}

// Node 获取AST节点值对象
func (ane *ASTNodeEntity) Node() value_objects.ASTNode {
	return ane.node
}

// IsValid 检查AST节点是否有效
func (ane *ASTNodeEntity) IsValid() bool {
	return ane.isValid
}

// Validate 验证AST节点
func (ane *ASTNodeEntity) Validate() error {
	// 验证节点的基本属性
	if ane.node == nil {
		return fmt.Errorf("AST node cannot be nil")
	}

	if ane.node.NodeType() == "" {
		return fmt.Errorf("AST node type cannot be empty")
	}

	// 检查位置信息
	location := ane.node.Location()
	if !location.IsValid() {
		return fmt.Errorf("AST node location is invalid")
	}

	// 递归验证子节点
	for _, child := range ane.children {
		if err := child.Validate(); err != nil {
			return fmt.Errorf("child node validation failed: %w", err)
		}
	}

	// 验证通过
	now := time.Now()
	ane.isValid = true
	ane.validatedAt = &now

	return nil
}

// ValidatedAt 获取验证时间
func (ane *ASTNodeEntity) ValidatedAt() *time.Time {
	return ane.validatedAt
}

// AddChild 添加子节点
func (ane *ASTNodeEntity) AddChild(child *ASTNodeEntity) {
	child.parent = ane
	ane.children = append(ane.children, child)
}

// Children 获取子节点列表
func (ane *ASTNodeEntity) Children() []*ASTNodeEntity {
	return ane.children
}

// Parent 获取父节点
func (ane *ASTNodeEntity) Parent() *ASTNodeEntity {
	return ane.parent
}

// ChildCount 获取子节点数量
func (ane *ASTNodeEntity) ChildCount() int {
	return len(ane.children)
}

// IsLeaf 检查是否为叶子节点
func (ane *ASTNodeEntity) IsLeaf() bool {
	return len(ane.children) == 0
}

// Depth 获取节点深度（从根节点开始计算）
func (ane *ASTNodeEntity) Depth() int {
	if ane.parent == nil {
		return 0
	}
	return ane.parent.Depth() + 1
}

// FindChildByType 按类型查找子节点
func (ane *ASTNodeEntity) FindChildByType(nodeType string) *ASTNodeEntity {
	for _, child := range ane.children {
		if child.node.NodeType() == nodeType {
			return child
		}
	}
	return nil
}

// FindChildrenByType 按类型查找所有子节点
func (ane *ASTNodeEntity) FindChildrenByType(nodeType string) []*ASTNodeEntity {
	var result []*ASTNodeEntity
	for _, child := range ane.children {
		if child.node.NodeType() == nodeType {
			result = append(result, child)
		}
	}
	return result
}

// Traverse 遍历AST树（深度优先）
func (ane *ASTNodeEntity) Traverse(visitor func(*ASTNodeEntity) error) error {
	// 前序遍历：先访问当前节点
	if err := visitor(ane); err != nil {
		return err
	}

	// 再访问子节点
	for _, child := range ane.children {
		if err := child.Traverse(visitor); err != nil {
			return err
		}
	}

	return nil
}

// String 返回实体的字符串表示
func (ane *ASTNodeEntity) String() string {
	return fmt.Sprintf("ASTNodeEntity{ID: %s, Type: %s, Valid: %t, Children: %d}",
		ane.id, ane.node.NodeType(), ane.isValid, len(ane.children))
}

// ProgramASTEntity 程序AST实体（聚合根）
// ProgramASTEntity 管理整个程序的AST结构
type ProgramASTEntity struct {
	id          string
	sourceFile  *value_objects.SourceFile
	rootNode    *ASTNodeEntity
	isComplete  bool
	createdAt   time.Time
	metadata    *ASTMetadata
}

// ASTMetadata AST元数据
type ASTMetadata struct {
	nodeCount     int
	maxDepth      int
	hasErrors     bool
	validatedAt   *time.Time
}

// NewProgramASTEntity 创建新的程序AST实体
func NewProgramASTEntity(id string, sourceFile *value_objects.SourceFile) *ProgramASTEntity {
	return &ProgramASTEntity{
		id:         id,
		sourceFile: sourceFile,
		isComplete: false,
		createdAt:  time.Now(),
		metadata: &ASTMetadata{
			nodeCount: 0,
			maxDepth:  0,
			hasErrors: false,
		},
	}
}

// ID 获取实体ID
func (pae *ProgramASTEntity) ID() string {
	return pae.id
}

// SourceFile 获取源文件
func (pae *ProgramASTEntity) SourceFile() *value_objects.SourceFile {
	return pae.sourceFile
}

// RootNode 获取根节点
func (pae *ProgramASTEntity) RootNode() *ASTNodeEntity {
	return pae.rootNode
}

// SetRootNode 设置根节点
func (pae *ProgramASTEntity) SetRootNode(rootNode *ASTNodeEntity) {
	pae.rootNode = rootNode
	pae.updateMetadata()
}

// IsComplete 检查AST是否构建完成
func (pae *ProgramASTEntity) IsComplete() bool {
	return pae.isComplete
}

// Complete 标记AST构建完成
func (pae *ProgramASTEntity) Complete() {
	pae.isComplete = true
	pae.updateMetadata()
}

// Validate 验证整个AST
func (pae *ProgramASTEntity) Validate() []*value_objects.ParseError {
	errors := make([]*value_objects.ParseError, 0)

	if pae.rootNode == nil {
		errors = append(errors, value_objects.NewParseError(
			"AST root node is nil",
			value_objects.NewSourceLocation(pae.sourceFile.Filename(), 1, 1, 0),
			value_objects.ErrorTypeSyntax,
			value_objects.SeverityError,
		))
		return errors
	}

	// 验证根节点
	if err := pae.rootNode.Validate(); err != nil {
		errors = append(errors, value_objects.NewParseError(
			fmt.Sprintf("AST validation failed: %v", err),
			pae.rootNode.node.Location(),
			value_objects.ErrorTypeSyntax,
			value_objects.SeverityError,
		))
	}

	return errors
}

// updateMetadata 更新元数据
func (pae *ProgramASTEntity) updateMetadata() {
	if pae.rootNode == nil {
		return
	}

	nodeCount := 0
	maxDepth := 0

	// 遍历AST树收集统计信息
	pae.rootNode.Traverse(func(node *ASTNodeEntity) error {
		nodeCount++
		depth := node.Depth()
		if depth > maxDepth {
			maxDepth = depth
		}
		return nil
	})

	pae.metadata.nodeCount = nodeCount
	pae.metadata.maxDepth = maxDepth
	now := time.Now()
	pae.metadata.validatedAt = &now
}

// Metadata 获取AST元数据
func (pae *ProgramASTEntity) Metadata() *ASTMetadata {
	return pae.metadata
}

// CreatedAt 获取创建时间
func (pae *ProgramASTEntity) CreatedAt() time.Time {
	return pae.createdAt
}

// String 返回实体的字符串表示
func (pae *ProgramASTEntity) String() string {
	rootType := "nil"
	if pae.rootNode != nil {
		rootType = pae.rootNode.node.NodeType()
	}
	return fmt.Sprintf("ProgramASTEntity{ID: %s, RootType: %s, Complete: %t, Nodes: %d}",
		pae.id, rootType, pae.isComplete, pae.metadata.nodeCount)
}
