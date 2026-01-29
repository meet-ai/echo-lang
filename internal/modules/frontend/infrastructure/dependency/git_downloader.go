// Package dependency 实现 Git 源依赖下载
package dependency

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// GitDownloader Git 下载器实现
type GitDownloader struct{}

// NewGitDownloader 创建 Git 下载器
func NewGitDownloader() *GitDownloader {
	return &GitDownloader{}
}

// Download 从 Git 仓库下载依赖包
func (d *GitDownloader) Download(ctx context.Context, packageName, version, targetDir string, options map[string]interface{}) error {
	opts := NewDownloadOptions("git", options)

	// 检查 Git URL
	if opts.GitURL == "" {
		return fmt.Errorf("git URL is required for git source")
	}

	// 确保目标目录的父目录存在
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// 如果目标目录已存在，先删除（支持重新下载）
	if _, err := os.Stat(targetDir); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	// 克隆仓库到临时目录
	tempDir := targetDir + ".tmp"
	defer os.RemoveAll(tempDir) // 清理临时目录

	// 构建 git clone 命令
	cloneArgs := []string{"clone", "--depth", "1", "--quiet"}
	
	// 根据引用类型添加参数
	if opts.Tag != "" {
		cloneArgs = append(cloneArgs, "--branch", opts.Tag, opts.GitURL, tempDir)
	} else if opts.Branch != "" {
		cloneArgs = append(cloneArgs, "--branch", opts.Branch, opts.GitURL, tempDir)
	} else {
		cloneArgs = append(cloneArgs, opts.GitURL, tempDir)
	}

	// 执行 git clone
	cmd := exec.CommandContext(ctx, "git", cloneArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone git repository: %w", err)
	}

	// 如果指定了 commit，需要 checkout 到该 commit
	if opts.Commit != "" {
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", opts.Commit)
		checkoutCmd.Dir = tempDir
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout commit %s: %w", opts.Commit, err)
		}
	}

	// 查找包文件（.eo 文件）
	// 策略：
	// 1. 查找与包名相同的 .eo 文件（如 mathlib.eo）
	// 2. 查找与包名相同的目录中的包文件（如 mathlib/mathlib.eo）
	// 3. 如果找不到，使用整个仓库内容
	packageFile := d.findPackageFile(tempDir, packageName)
	
	if packageFile != "" {
		// 找到包文件，只复制包文件或包目录
		if err := d.copyPackageFile(packageFile, targetDir, packageName); err != nil {
			return fmt.Errorf("failed to copy package file: %w", err)
		}
	} else {
		// 没找到包文件，复制整个仓库内容
		if err := d.copyDirectory(tempDir, targetDir); err != nil {
			return fmt.Errorf("failed to copy repository: %w", err)
		}
	}

	// 移除 .git 目录（如果存在）
	gitDir := filepath.Join(targetDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		if err := os.RemoveAll(gitDir); err != nil {
			return fmt.Errorf("failed to remove .git directory: %w", err)
		}
	}

	return nil
}

// findPackageFile 查找包文件
// 返回包文件的完整路径，如果找不到返回空字符串
func (d *GitDownloader) findPackageFile(repoDir, packageName string) string {
	// 策略1：查找与包名相同的 .eo 文件
	packageBaseName := filepath.Base(packageName)
	eoFile := filepath.Join(repoDir, packageBaseName+".eo")
	if info, err := os.Stat(eoFile); err == nil && !info.IsDir() {
		return eoFile
	}

	// 策略2：查找与包名相同的目录中的包文件
	packageDir := filepath.Join(repoDir, packageBaseName)
	if info, err := os.Stat(packageDir); err == nil && info.IsDir() {
		packageFile := filepath.Join(packageDir, packageBaseName+".eo")
		if _, err := os.Stat(packageFile); err == nil {
			return packageFile
		}
		// 如果目录存在但没有包文件，返回目录路径
		return packageDir
	}

	// 策略3：查找所有 .eo 文件，选择第一个
	// 这里简化处理，返回空字符串表示使用整个仓库
	return ""
}

// copyPackageFile 复制包文件或包目录到目标目录
func (d *GitDownloader) copyPackageFile(sourcePath, targetDir, packageName string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		// 如果是目录，复制整个目录
		return d.copyDirectory(sourcePath, targetDir)
	}

	// 如果是文件，创建目标目录并复制文件
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// 复制文件
	targetFile := filepath.Join(targetDir, filepath.Base(sourcePath))
	return d.copyFile(sourcePath, targetFile)
}

// copyDirectory 复制目录
func (d *GitDownloader) copyDirectory(sourceDir, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// 使用简单的文件复制（递归）
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过 .git 目录
		if info.IsDir() && filepath.Base(path) == ".git" {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return d.copyFile(path, targetPath)
	})
}

// copyFile 复制文件
func (d *GitDownloader) copyFile(source, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	return err
}
