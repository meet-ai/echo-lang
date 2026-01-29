// Package dependency 测试依赖下载服务
package dependency

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"echo/internal/modules/frontend/infrastructure/config"
)

func TestDownloadService_DownloadDependency(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "echo-download-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 echo.toml 配置文件
	configContent := `name = "test-project"
version = "1.0.0"

[package]
entry = "src/main.eo"
module = "test"

[dependencies]
test-lib = { version = "1.0.0", source = "git", git = "https://github.com/echo-lang/test-lib.git", tag = "v1.0.0" }
`
	configPath := filepath.Join(tempDir, "echo.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 创建下载服务
	service, err := NewDownloadService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create download service: %v", err)
	}

	// 测试获取存储路径
	storagePath := service.GetDependencyStoragePath("test-lib")
	expectedPath := filepath.Join(tempDir, "vendor", "test-lib@1.0.0")
	if storagePath != expectedPath {
		t.Errorf("Expected storage path %s, got %s", expectedPath, storagePath)
	}

	// 注意：实际下载测试需要真实的 Git 仓库，这里只测试路径解析
	// 实际下载测试应该在集成测试中进行
}

func TestDownloadService_DownloadAllDependencies(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "echo-download-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 echo.toml 配置文件（无依赖）
	configContent := `name = "test-project"
version = "1.0.0"

[package]
entry = "src/main.eo"
module = "test"
`
	configPath := filepath.Join(tempDir, "echo.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 创建下载服务
	service, err := NewDownloadService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create download service: %v", err)
	}

	// 测试下载所有依赖（应该成功，因为没有依赖）
	ctx := context.Background()
	if err := service.DownloadAllDependencies(ctx); err != nil {
		t.Errorf("Expected no error for empty dependencies, got: %v", err)
	}
}

func TestEchoConfig_GetDependencyStoragePath(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "echo-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 echo.toml 配置文件
	configContent := `name = "test-project"
version = "1.0.0"

[dependencies]
mathlib = "2.1.0"
"network/http" = "1.0.0"
`
	configPath := filepath.Join(tempDir, "echo.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 加载配置
	cfg, err := config.LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 测试存储路径生成
	tests := []struct {
		packageName string
		expected    string
	}{
		{"mathlib", filepath.Join("vendor", "mathlib@2.1.0")},
		{"network/http", filepath.Join("vendor", "network-http@1.0.0")},
	}

	for _, tt := range tests {
		t.Run(tt.packageName, func(t *testing.T) {
			path := cfg.GetDependencyStoragePath(tt.packageName)
			if path != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, path)
			}
		})
	}
}

func TestDownloadService_GetDependencyStoragePath_UsesLockWhenPresent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "echo-download-lock-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// echo.toml: test-lib = 1.0.0
	configContent := `name = "test-project"
version = "1.0.0"

[package]
entry = "src/main.eo"
module = "test"

[dependencies]
test-lib = { version = "1.0.0", source = "git", git = "https://example.com/test-lib.git", tag = "v1.0.0" }
`
	if err := os.WriteFile(filepath.Join(tempDir, "echo.toml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write echo.toml: %v", err)
	}

	// echo.lock: test-lib locked at 2.0.0
	lockContent := `schema_version = "1"
[dependencies]
test-lib = "2.0.0"
`
	if err := os.WriteFile(filepath.Join(tempDir, "echo.lock"), []byte(lockContent), 0644); err != nil {
		t.Fatalf("Failed to write echo.lock: %v", err)
	}

	service, err := NewDownloadService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create download service: %v", err)
	}

	// 应使用 lock 中的 2.0.0，而非 echo.toml 的 1.0.0
	storagePath := service.GetDependencyStoragePath("test-lib")
	expectedPath := filepath.Join(tempDir, "vendor", "test-lib@2.0.0")
	if storagePath != expectedPath {
		t.Errorf("GetDependencyStoragePath() = %s, want %s (should use locked version 2.0.0)", storagePath, expectedPath)
	}
}

// TestDownloadService_DownloadDependency_RegistrySource 集成测试：source=registry 时通过 mock 注册表下载（T-DEV-024）
func TestDownloadService_DownloadDependency_RegistrySource(t *testing.T) {
	tarball := minimalTarGz(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/packages/reg-pkg/versions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]string{"1.0.0"})
		case "/packages/reg-pkg/versions/1.0.0":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version":    "1.0.0",
				"source_url": "http://" + r.Host + "/tarball.tar.gz",
			})
		case "/tarball.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(tarball)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	configContent := `name = "test-project"
version = "1.0.0"

[package]
entry = "src/main.eo"
module = "test"

[registry]
url = "` + server.URL + `"

[dependencies]
reg-pkg = { version = "1.0.0", source = "registry" }
`
	if err := os.WriteFile(filepath.Join(tempDir, "echo.toml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("write echo.toml: %v", err)
	}

	svc, err := NewDownloadService(tempDir)
	if err != nil {
		t.Fatalf("NewDownloadService: %v", err)
	}
	ctx := context.Background()
	if err := svc.DownloadDependency(ctx, "reg-pkg"); err != nil {
		t.Fatalf("DownloadDependency(reg-pkg): %v", err)
	}
	expectedDir := filepath.Join(tempDir, "vendor", "reg-pkg@1.0.0")
	if _, err := os.Stat(expectedDir); err != nil {
		t.Errorf("vendor/reg-pkg@1.0.0 not found: %v", err)
	}
	pkgPath := filepath.Join(expectedDir, "pkg.eo")
	if _, err := os.Stat(pkgPath); err != nil {
		t.Errorf("vendor/reg-pkg@1.0.0/pkg.eo not found: %v", err)
	}
	lockPath := filepath.Join(tempDir, "echo.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("echo.lock not written: %v", err)
	}
}

func TestEchoConfig_GetDependencyGitInfo(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "echo-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 echo.toml 配置文件（带 Git 配置）
	configContent := `name = "test-project"
version = "1.0.0"

[dependencies]
test-lib = { version = "1.0.0", source = "git", git = "https://github.com/echo-lang/test-lib.git", tag = "v1.0.0" }
another-lib = { version = "2.0.0", source = "git", git = "https://github.com/echo-lang/another-lib.git", branch = "main" }
`
	configPath := filepath.Join(tempDir, "echo.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 加载配置
	cfg, err := config.LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 测试获取 Git 信息
	tests := []struct {
		packageName string
		source      string
		gitURL      string
		refType     string
		refValue    string
	}{
		{"test-lib", "git", "https://github.com/echo-lang/test-lib.git", "tag", "v1.0.0"},
		{"another-lib", "git", "https://github.com/echo-lang/another-lib.git", "branch", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.packageName, func(t *testing.T) {
			source := cfg.GetDependencySource(tt.packageName)
			if source != tt.source {
				t.Errorf("Expected source %s, got %s", tt.source, source)
			}

			gitURL := cfg.GetDependencyGitURL(tt.packageName)
			if gitURL != tt.gitURL {
				t.Errorf("Expected git URL %s, got %s", tt.gitURL, gitURL)
			}

			refType, refValue := cfg.GetDependencyGitRef(tt.packageName)
			if refType != tt.refType || refValue != tt.refValue {
				t.Errorf("Expected ref %s=%s, got %s=%s", tt.refType, tt.refValue, refType, refValue)
			}
		})
	}
}
