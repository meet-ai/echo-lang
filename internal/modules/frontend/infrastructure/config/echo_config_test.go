// Package config 实现配置文件解析测试
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig_Default 测试默认配置加载
func TestLoadConfig_Default(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 测试：配置文件不存在时返回默认配置
	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() 错误 = %v", err)
	}

	if config == nil {
		t.Fatal("LoadConfig() 返回 nil")
	}

	// 检查默认值
	if config.Name != "untitled" {
		t.Errorf("默认名称不匹配: 期望 'untitled', 得到 %q", config.Name)
	}

	if config.Version != "0.1.0" {
		t.Errorf("默认版本不匹配: 期望 '0.1.0', 得到 %q", config.Version)
	}

	if config.Package.Entry != "src/main.eo" {
		t.Errorf("默认入口不匹配: 期望 'src/main.eo', 得到 %q", config.Package.Entry)
	}
}

// TestGetProjectRoot 测试项目根目录查找
func TestGetProjectRoot(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "myproject")
	os.MkdirAll(projectRoot, 0755)

	// 创建 echo.toml 文件
	configFile := filepath.Join(projectRoot, "echo.toml")
	os.WriteFile(configFile, []byte("# test config"), 0644)

	// 测试：从项目根目录查找
	root, err := GetProjectRoot(projectRoot)
	if err != nil {
		t.Fatalf("GetProjectRoot() 错误 = %v", err)
	}

	if root != projectRoot {
		t.Errorf("GetProjectRoot() = %q, 期望 %q", root, projectRoot)
	}

	// 测试：从子目录查找
	subDir := filepath.Join(projectRoot, "src", "main")
	os.MkdirAll(subDir, 0755)

	root, err = GetProjectRoot(subDir)
	if err != nil {
		t.Fatalf("GetProjectRoot() 从子目录查找错误 = %v", err)
	}

	if root != projectRoot {
		t.Errorf("GetProjectRoot() 从子目录 = %q, 期望 %q", root, projectRoot)
	}

	// 测试：找不到配置文件
	otherDir := t.TempDir()
	_, err = GetProjectRoot(otherDir)
	if err == nil {
		t.Error("期望错误，但没有错误（找不到配置文件）")
	}
}

