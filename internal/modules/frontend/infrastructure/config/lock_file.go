// Package config 实现 echo.lock 的读取与写入
// 使用场景：echoc fetch 成功后写入解析后的精确版本，保证可重复构建；存在 echo.lock 时优先使用锁定版本。
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// LockFile echo.lock 文件结构
// 记录各依赖的解析后精确版本，与 echo.toml 的约束/精确版本对应
type LockFile struct {
	SchemaVersion string            `toml:"schema_version"` // 锁文件格式版本，当前为 "1"
	Dependencies  map[string]string `toml:"dependencies"`   // 包名 -> 锁定版本
}

const lockSchemaVersion = "1"
const lockFileName = "echo.lock"

// LoadLock 从项目根目录读取 echo.lock；不存在或为空时返回 nil, nil
func LoadLock(projectRoot string) (*LockFile, error) {
	p := filepath.Join(projectRoot, lockFileName)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}
	var lf LockFile
	if err := toml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}
	if lf.Dependencies == nil {
		lf.Dependencies = make(map[string]string)
	}
	return &lf, nil
}

// SaveLock 将锁文件写入项目根目录
func SaveLock(projectRoot string, lf *LockFile) error {
	if lf == nil {
		return fmt.Errorf("lock file is nil")
	}
	if lf.SchemaVersion == "" {
		lf.SchemaVersion = lockSchemaVersion
	}
	if lf.Dependencies == nil {
		lf.Dependencies = make(map[string]string)
	}
	data, err := toml.Marshal(lf)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}
	p := filepath.Join(projectRoot, lockFileName)
	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}
	return nil
}

// GetLockedVersion 返回某依赖在 lock 中的锁定版本，无则返回空字符串
func (l *LockFile) GetLockedVersion(packageName string) string {
	if l == nil || l.Dependencies == nil {
		return ""
	}
	return l.Dependencies[packageName]
}

// SetLockedVersion 设置某依赖的锁定版本（内存中）
func (l *LockFile) SetLockedVersion(packageName, version string) {
	if l.Dependencies == nil {
		l.Dependencies = make(map[string]string)
	}
	l.Dependencies[packageName] = version
}
