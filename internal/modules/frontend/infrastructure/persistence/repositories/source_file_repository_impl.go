package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

// sourceFileRepositoryImpl 源文件仓储实现
type sourceFileRepositoryImpl struct {
	db *sql.DB
}

// NewSourceFileRepository 创建源文件仓储
func NewSourceFileRepository(db *sql.DB) SourceFileRepository {
	return &sourceFileRepositoryImpl{
		db: db,
	}
}

// Save 保存源文件
func (r *sourceFileRepositoryImpl) Save(ctx context.Context, sourceFile *entities.SourceFile) error {
	query := `
		INSERT INTO source_files (id, file_path, content, content_hash, file_size, analysis_status, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			content = VALUES(content),
			content_hash = VALUES(content_hash),
			file_size = VALUES(file_size),
			analysis_status = VALUES(analysis_status),
			modified_at = VALUES(modified_at)
	`

	// 由于SourceFile字段是私有的，我们需要一个不同的方法
	// 这里简化处理，直接使用模拟数据
	_, err := r.db.ExecContext(ctx, query,
		sourceFile.ID(),
		sourceFile.FilePath(),
		sourceFile.Content(),
		sourceFile.ContentHash(),
		int64(len(sourceFile.Content())), // 估算文件大小
		string(sourceFile.AnalysisStatus()),
		time.Now(), // createdAt
		time.Now(), // modifiedAt
	)

	if err != nil {
		return fmt.Errorf("failed to save source file: %w", err)
	}

	return nil
}

// FindByID 根据ID查找源文件
func (r *sourceFileRepositoryImpl) FindByID(ctx context.Context, id string) (*entities.SourceFile, error) {
	query := `
		SELECT id, file_path, content, content_hash, file_size, analysis_status,
			   last_analyzed_at, created_at, modified_at
		FROM source_files
		WHERE id = ?
	`

	var sourceFile entities.SourceFile
	var content, contentHash string
	var fileSize int64
	var analysisStatus string
	var lastAnalyzedAt sql.NullTime
	var createdAt, modifiedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&sourceFile.id,
		&sourceFile.filePath,
		&content,
		&contentHash,
		&fileSize,
		&analysisStatus,
		&lastAnalyzedAt,
		&createdAt,
		&modifiedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("source file not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find source file: %w", err)
	}

	// 构建实体（简化版，实际需要完整的实体构造）
	sourceFile.content = content
	sourceFile.contentHash = contentHash
	sourceFile.fileSize = fileSize
	sourceFile.createdAt = createdAt
	sourceFile.modifiedAt = modifiedAt

	// 转换分析状态
	switch analysisStatus {
	case "pending":
		sourceFile.analysisStatus = entities.AnalysisStatusPending
	case "lexical_completed":
		sourceFile.analysisStatus = entities.AnalysisStatusLexical
	case "syntax_completed":
		sourceFile.analysisStatus = entities.AnalysisStatusSyntax
	case "semantic_completed":
		sourceFile.analysisStatus = entities.AnalysisStatusSemantic
	case "failed":
		sourceFile.analysisStatus = entities.AnalysisStatusFailed
	default:
		sourceFile.analysisStatus = entities.AnalysisStatusPending
	}

	if lastAnalyzedAt.Valid {
		sourceFile.lastAnalyzedAt = &lastAnalyzedAt.Time
	}

	return &sourceFile, nil
}

// FindByPath 根据路径查找源文件
func (r *sourceFileRepositoryImpl) FindByPath(ctx context.Context, filePath string) (*entities.SourceFile, error) {
	query := `
		SELECT id, file_path, content, content_hash, file_size, analysis_status,
			   last_analyzed_at, created_at, modified_at
		FROM source_files
		WHERE file_path = ?
	`

	// 复用FindByID的逻辑，简化实现
	var id string
	err := r.db.QueryRowContext(ctx, query, filePath).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("source file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to find source file by path: %w", err)
	}

	return r.FindByID(ctx, id)
}

// FindByStatus 根据状态查找源文件
func (r *sourceFileRepositoryImpl) FindByStatus(ctx context.Context, status entities.AnalysisStatus) ([]*entities.SourceFile, error) {
	query := `
		SELECT id
		FROM source_files
		WHERE analysis_status = ?
		ORDER BY modified_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query source files by status: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan source file id: %w", err)
		}
		ids = append(ids, id)
	}

	// 获取完整的源文件实体
	var sourceFiles []*entities.SourceFile
	for _, id := range ids {
		sourceFile, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get source file %s: %w", id, err)
		}
		sourceFiles = append(sourceFiles, sourceFile)
	}

	return sourceFiles, nil
}

// Update 更新源文件
func (r *sourceFileRepositoryImpl) Update(ctx context.Context, sourceFile *entities.SourceFile) error {
	return r.Save(ctx, sourceFile)
}

// Delete 删除源文件
func (r *sourceFileRepositoryImpl) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM source_files WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete source file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("source file not found: %s", id)
	}

	return nil
}

// Exists 检查源文件是否存在
func (r *sourceFileRepositoryImpl) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT 1 FROM source_files WHERE id = ? LIMIT 1`

	var dummy int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&dummy)

	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check source file existence: %w", err)
	}

	return true, nil
}

// ListAll 列出所有源文件
func (r *sourceFileRepositoryImpl) ListAll(ctx context.Context, offset, limit int) ([]*entities.SourceFile, error) {
	query := `
		SELECT id
		FROM source_files
		ORDER BY modified_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list source files: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan source file id: %w", err)
		}
		ids = append(ids, id)
	}

	// 获取完整的源文件实体
	var sourceFiles []*entities.SourceFile
	for _, id := range ids {
		sourceFile, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get source file %s: %w", id, err)
		}
		sourceFiles = append(sourceFiles, sourceFile)
	}

	return sourceFiles, nil
}

// Count 返回源文件总数
func (r *sourceFileRepositoryImpl) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM source_files`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count source files: %w", err)
	}

	return count, nil
}
