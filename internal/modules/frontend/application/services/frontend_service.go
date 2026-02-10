package services

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"echo/internal/modules/frontend"
	"echo/internal/modules/frontend/domain/commands"
	"echo/internal/modules/frontend/domain/dtos"
	"echo/internal/modules/frontend/domain/entities"
	"echo/internal/modules/frontend/domain/services"
)

// IFrontendService 前端服务接口
type IFrontendService interface {
	PerformLexicalAnalysis(ctx context.Context, cmd frontend.PerformLexicalAnalysisCommand) (*commands.LexicalAnalysisResult, error)
	PerformSyntaxAnalysis(ctx context.Context, cmd commands.PerformSyntaxAnalysisCommand) (*commands.SyntaxAnalysisResult, error)
	PerformSemanticAnalysis(ctx context.Context, cmd commands.PerformSemanticAnalysisCommand) (*commands.SemanticAnalysisResult, error)
	HandleCompilationErrors(ctx context.Context, cmd commands.HandleCompilationErrorsCommand) (*commands.ErrorHandlingResult, error)
	CompileFile(filePath string) (*dtos.CompilationResult, error)
}

// AnalysisCompletedEvent 分析完成事件
type AnalysisCompletedEvent struct {
	EventID      string        `json:"event_id"`
	EventType    string        `json:"event_type"`
	Timestamp    time.Time     `json:"timestamp"`
	SourceFileID string        `json:"source_file_id"`
	AnalysisType string        `json:"analysis_type"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
}

// SourceFileRepository 源文件仓储接口
type SourceFileRepository interface {
	Save(ctx context.Context, sourceFile *entities.SourceFile) error
	FindByID(ctx context.Context, id string) (*entities.SourceFile, error)
	Update(ctx context.Context, sourceFile *entities.SourceFile) error
}

// ASTRepository AST仓储接口
type ASTRepository interface {
	Save(ctx context.Context, sourceFileID string, ast *entities.ASTNode) error
	FindBySourceFileID(ctx context.Context, sourceFileID string) (*entities.ASTNode, error)
}

// EventPublisher 事件发布器接口
type EventPublisher interface {
	Publish(ctx context.Context, event interface{}) error
}

// BackendService 后端服务接口（可选依赖）
type BackendService interface {
	GenerateCode(program *entities.Program, target string) (string, error)
	GenerateExecutable(program *entities.Program, target string) (string, error)
}

// frontendService 前端应用服务实现
type frontendService struct {
	lexicalAnalyzer  services.LexicalAnalyzer
	syntaxAnalyzer   services.SyntaxAnalyzer
	semanticAnalyzer services.SemanticAnalyzer
	errorHandler     services.ErrorHandler
	sourceFileRepo   SourceFileRepository
	astRepo          ASTRepository
	eventPublisher   EventPublisher
	parser           ParserFacade
	backendService   BackendService // 后端服务（可选）
}

// NewFrontendService 创建前端应用服务
func NewFrontendService(
	lexicalAnalyzer services.LexicalAnalyzer,
	syntaxAnalyzer services.SyntaxAnalyzer,
	semanticAnalyzer services.SemanticAnalyzer,
	errorHandler services.ErrorHandler,
	sourceFileRepo SourceFileRepository,
	astRepo ASTRepository,
	eventPublisher EventPublisher,
	parser ParserFacade,
) IFrontendService {
	return &frontendService{
		lexicalAnalyzer:  lexicalAnalyzer,
		syntaxAnalyzer:   syntaxAnalyzer,
		semanticAnalyzer: semanticAnalyzer,
		errorHandler:     errorHandler,
		sourceFileRepo:   sourceFileRepo,
		astRepo:          astRepo,
		eventPublisher:   eventPublisher,
		parser:           parser,
		backendService:   nil, // 默认不注入，可通过WithBackendService设置
	}
}

// WithBackendService 设置后端服务（可选）
func (s *frontendService) WithBackendService(backendService BackendService) *frontendService {
	s.backendService = backendService
	return s
}

// PerformLexicalAnalysis 执行词法分析用例
func (s *frontendService) PerformLexicalAnalysis(ctx context.Context, cmd frontend.PerformLexicalAnalysisCommand) (*commands.LexicalAnalysisResult, error) {
	// 1. 验证命令
	if err := s.validateLexicalAnalysisCommand(cmd); err != nil {
		return nil, fmt.Errorf("invalid command: %w", err)
	}

	// 2. 获取或创建源文件
	sourceFile, err := s.getOrCreateSourceFile(ctx, cmd.SourceFileID, cmd.SourceCode, cmd.SourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get source file: %w", err)
	}

	// 3. 执行词法分析
	result, err := s.lexicalAnalyzer.Analyze(ctx, sourceFile)
	if err != nil {
		// 处理错误
		handleErr := s.errorHandler.HandleAnalysisError(ctx, sourceFile.ID(), "lexical", err)
		sourceFile.AddErrorMessage(handleErr.Message)

		// 发布事件
		s.publishAnalysisEvent(ctx, sourceFile.ID(), "lexical", false, time.Since(time.Now()))

		return nil, fmt.Errorf("lexical analysis failed: %w", err)
	}

	// 4. 更新源文件状态
	sourceFile.SetTokens(result.Tokens)
	sourceFile.SetAnalysisStatus(entities.AnalysisStatusLexical)

	// 5. 保存到仓库
	if err := s.sourceFileRepo.Save(ctx, sourceFile); err != nil {
		return nil, fmt.Errorf("failed to save source file: %w", err)
	}

	// 6. 发布成功事件
	s.publishAnalysisEvent(ctx, sourceFile.ID(), "lexical", true, result.Duration)

	return &frontend.LexicalAnalysisResult{
		SourceFileID: sourceFile.ID(),
		TokenCount:   result.TokenCount,
		Success:      true,
		Duration:     result.Duration,
	}, nil
}

// PerformSyntaxAnalysis 执行语法分析用例
func (s *frontendService) PerformSyntaxAnalysis(ctx context.Context, cmd commands.PerformSyntaxAnalysisCommand) (*commands.SyntaxAnalysisResult, error) {
	// 获取源文件
	sourceFile, err := s.sourceFileRepo.FindByID(ctx, cmd.SourceFileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find source file: %w", err)
	}

	// 验证前置条件
	if sourceFile.AnalysisStatus() < entities.AnalysisStatusLexical {
		return nil, fmt.Errorf("lexical analysis must be completed first")
	}

	// 执行语法分析
	result, err := s.syntaxAnalyzer.Analyze(ctx, sourceFile)
	if err != nil {
		handleErr := s.errorHandler.HandleAnalysisError(ctx, sourceFile.ID(), "syntax", err)
		sourceFile.AddErrorMessage(handleErr.Message)
		s.publishAnalysisEvent(ctx, sourceFile.ID(), "syntax", false, time.Since(time.Now()))
		return nil, fmt.Errorf("syntax analysis failed: %w", err)
	}

	// 更新源文件
	sourceFile.SetAST(result.AST)
	sourceFile.SetAnalysisStatus(entities.AnalysisStatusSyntax)

	// 保存并发布事件
	if err := s.sourceFileRepo.Save(ctx, sourceFile); err != nil {
		return nil, fmt.Errorf("failed to save source file: %w", err)
	}

	s.publishAnalysisEvent(ctx, sourceFile.ID(), "syntax", true, result.Duration)

	return &frontend.SyntaxAnalysisResult{
		SourceFileID: sourceFile.ID(),
		Success:      true,
		Duration:     result.Duration,
	}, nil
}

// PerformSemanticAnalysis 执行语义分析用例
func (s *frontendService) PerformSemanticAnalysis(ctx context.Context, cmd commands.PerformSemanticAnalysisCommand) (*commands.SemanticAnalysisResult, error) {
	// 获取源文件
	sourceFile, err := s.sourceFileRepo.FindByID(ctx, cmd.SourceFileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find source file: %w", err)
	}

	// 验证前置条件
	if sourceFile.AnalysisStatus() < entities.AnalysisStatusSyntax {
		return nil, fmt.Errorf("syntax analysis must be completed first")
	}

	// 执行语义分析
	result, err := s.semanticAnalyzer.Analyze(ctx, sourceFile)
	if err != nil {
		handleErr := s.errorHandler.HandleAnalysisError(ctx, sourceFile.ID(), "semantic", err)
		sourceFile.AddErrorMessage(handleErr.Message)
		s.publishAnalysisEvent(ctx, sourceFile.ID(), "semantic", false, time.Since(time.Now()))
		return nil, fmt.Errorf("semantic analysis failed: %w", err)
	}

	// 更新源文件
	if symbolTable, ok := result.SymbolTable.(*entities.SymbolTable); ok {
		sourceFile.SetSymbolTable(symbolTable)
	}
	sourceFile.SetAnalysisStatus(entities.AnalysisStatusSemantic)

	// 保存并发布事件
	if err := s.sourceFileRepo.Save(ctx, sourceFile); err != nil {
		return nil, fmt.Errorf("failed to save source file: %w", err)
	}

	s.publishAnalysisEvent(ctx, sourceFile.ID(), "semantic", true, result.Duration)

	return &frontend.SemanticAnalysisResult{
		SourceFileID: sourceFile.ID(),
		Success:      true,
		Duration:     result.Duration,
	}, nil
}

// HandleCompilationErrors 处理编译错误用例
func (s *frontendService) HandleCompilationErrors(ctx context.Context, cmd commands.HandleCompilationErrorsCommand) (*commands.ErrorHandlingResult, error) {
	// 获取源文件
	sourceFile, err := s.sourceFileRepo.FindByID(ctx, cmd.SourceFileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find source file: %w", err)
	}

	// 处理错误
	result := s.errorHandler.HandleCompilationErrors(ctx, sourceFile.ID(), cmd.Errors)

	// 更新源文件错误信息
	for _, errMsg := range result.Suggestions {
		sourceFile.AddErrorMessage(errMsg)
	}

	// 保存更新
	if err := s.sourceFileRepo.Save(ctx, sourceFile); err != nil {
		return nil, fmt.Errorf("failed to save source file: %w", err)
	}

	return &commands.ErrorHandlingResult{
		SourceFileID:   result.SourceFileID,
		Suggestions:    result.Suggestions,
		ProcessedCount: result.ProcessedCount,
		Success:        result.Success,
	}, nil
}

// Helper methods

func (s *frontendService) validateLexicalAnalysisCommand(cmd frontend.PerformLexicalAnalysisCommand) error {
	if cmd.SourceFileID == "" {
		return fmt.Errorf("source file ID is required")
	}
	if cmd.SourceCode == "" {
		return fmt.Errorf("source code is required")
	}
	if cmd.SourceFilePath == "" {
		return fmt.Errorf("source file path is required")
	}
	return nil
}

func (s *frontendService) getOrCreateSourceFile(ctx context.Context, id, code, path string) (*entities.SourceFile, error) {
	// Try to find existing
	if existing, err := s.sourceFileRepo.FindByID(ctx, id); err == nil {
		// Check if content changed
		if existing.Content() != code {
			if err := existing.UpdateContent(code); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}

	// Create new
	sourceFile, err := entities.NewSourceFile(id, path, code)
	if err != nil {
		return nil, err
	}

	return sourceFile, nil
}

func (s *frontendService) publishAnalysisEvent(ctx context.Context, sourceFileID, analysisType string, success bool, duration time.Duration) {
	event := &AnalysisCompletedEvent{
		EventID:      generateEventID(),
		EventType:    "frontend.analysis.completed",
		Timestamp:    time.Now(),
		SourceFileID: sourceFileID,
		AnalysisType: analysisType,
		Success:      success,
		Duration:     duration,
	}

	s.eventPublisher.Publish(ctx, event)
}

func generateEventID() string {
	return fmt.Sprintf("frontend-event-%d", time.Now().UnixNano())
}

// CompileFile 编译源文件 (简化版本，用于演示)
func (s *frontendService) CompileFile(filePath string) (*dtos.CompilationResult, error) {
	// 读取文件内容
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return &dtos.CompilationResult{
			SourceFile: filePath,
			Success:    false,
			Error:      err.Error(),
		}, nil
	}

	// 解析文件
	program, err := s.parser.Parse(string(content))
	if err != nil {
		return &dtos.CompilationResult{
			SourceFile: filePath,
			AST:        "",
			Success:    false,
			Error:      err.Error(),
		}, nil
	}

	// 调用backend服务生成代码
	var generatedCode string
	if s.backendService != nil {
		// parser.Parse 已经返回 *entities.Program，直接使用
		code, err := s.backendService.GenerateCode(program, "ocaml")
		if err != nil {
			return &dtos.CompilationResult{
				SourceFile:    filePath,
				AST:           program.String(),
				GeneratedCode: "",
				Success:       false,
				Error:         fmt.Sprintf("code generation failed: %v", err),
			}, nil
		}
		generatedCode = code
	} else {
		// 如果没有注入backend服务，返回占位符
		generatedCode = "(* Backend service not configured. Please inject BackendService via WithBackendService() *)"
	}

	return &dtos.CompilationResult{
		SourceFile:    filePath,
		AST:           program.String(),
		GeneratedCode: generatedCode,
		Success:       true,
		Error:         "",
	}, nil
}
