// Package dependency 测试包注册表客户端、RegistryVersionLister、RegistryDownloader（T-DEV-024）
package dependency

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// mockRegistry 内存实现 PackageRegistry，用于单测
type mockRegistry struct {
	versions map[string][]string
	metadata map[string]map[string]*PackageMetadata // package -> version -> meta
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		versions: make(map[string][]string),
		metadata: make(map[string]map[string]*PackageMetadata),
	}
}

func (m *mockRegistry) ListVersions(ctx context.Context, packageName string) ([]string, error) {
	if v, ok := m.versions[packageName]; ok {
		return v, nil
	}
	return nil, nil
}

func (m *mockRegistry) GetMetadata(ctx context.Context, packageName, version string) (*PackageMetadata, error) {
	if p, ok := m.metadata[packageName]; ok {
		if meta, ok := p[version]; ok {
			return meta, nil
		}
	}
	return nil, nil
}

func TestRegistryVersionLister_ListCandidateVersions(t *testing.T) {
	ctx := context.Background()
	mock := newMockRegistry()
	mock.versions["pkg-a"] = []string{"1.0.0", "1.1.0"}
	fallback := NewDefaultVersionLister()
	lister := NewRegistryVersionLister(mock, fallback)

	// source=registry 时走 Registry
	got, err := lister.ListCandidateVersions(ctx, "pkg-a", "registry", nil)
	if err != nil {
		t.Fatalf("ListCandidateVersions(registry): %v", err)
	}
	if len(got) != 2 || got[0] != "1.0.0" || got[1] != "1.1.0" {
		t.Errorf("ListCandidateVersions(registry) = %v, want [1.0.0 1.1.0]", got)
	}

	// source=git 时走 Fallback（无 git URL 会报错，这里只验证委托）
	_, _ = lister.ListCandidateVersions(ctx, "other", "git", map[string]interface{}{"git": "https://example.com/repo.git"})
}

func TestHTTPRegistryClient_ListVersions(t *testing.T) {
	ctx := context.Background()

	t.Run("array_format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/packages/pkg/versions" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]string{"1.0.0", "2.0.0"})
		}))
		defer server.Close()

		client := NewHTTPRegistryClient(server.URL, WithRegistryCacheTTL(0))
		got, err := client.ListVersions(ctx, "pkg")
		if err != nil {
			t.Fatalf("ListVersions: %v", err)
		}
		if len(got) != 2 || got[0] != "1.0.0" || got[1] != "2.0.0" {
			t.Errorf("ListVersions = %v, want [1.0.0 2.0.0]", got)
		}
	})

	t.Run("object_format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/packages/pkg2/versions" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"versions": []string{"0.1.0"}})
		}))
		defer server.Close()

		client := NewHTTPRegistryClient(server.URL, WithRegistryCacheTTL(0))
		got, err := client.ListVersions(ctx, "pkg2")
		if err != nil {
			t.Fatalf("ListVersions: %v", err)
		}
		if len(got) != 1 || got[0] != "0.1.0" {
			t.Errorf("ListVersions = %v, want [0.1.0]", got)
		}
	})
}

func TestHTTPRegistryClient_GetMetadata(t *testing.T) {
	ctx := context.Background()

	t.Run("200_with_body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/packages/pkg/versions/1.0.0" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version":    "1.0.0",
				"source_url": "https://cdn.example.com/pkg-1.0.0.tar.gz",
				"dependencies": map[string]string{"dep": "1.0.0"},
			})
		}))
		defer server.Close()

		client := NewHTTPRegistryClient(server.URL)
		meta, err := client.GetMetadata(ctx, "pkg", "1.0.0")
		if err != nil {
			t.Fatalf("GetMetadata: %v", err)
		}
		if meta == nil {
			t.Fatal("GetMetadata: want non-nil meta")
		}
		if meta.Version != "1.0.0" || meta.SourceURL != "https://cdn.example.com/pkg-1.0.0.tar.gz" {
			t.Errorf("GetMetadata = %+v", meta)
		}
		if meta.Dependencies["dep"] != "1.0.0" {
			t.Errorf("Dependencies = %v", meta.Dependencies)
		}
	})

	t.Run("404_returns_nil_nil", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewHTTPRegistryClient(server.URL)
		meta, err := client.GetMetadata(ctx, "pkg", "9.9.9")
		if err != nil {
			t.Fatalf("GetMetadata 404: %v", err)
		}
		if meta != nil {
			t.Errorf("GetMetadata 404: want nil meta, got %+v", meta)
		}
	})
}

// minimalTarGz 返回最小合法 .tar.gz 字节（顶层目录 pkg/ 内单文件 pkg.eo，供 HttpDownloader 解压后得到 targetDir/pkg.eo）
func minimalTarGz(t *testing.T) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	// 单层目录 + 文件，解压时会 strip 顶层目录 pkg，得到 pkg.eo
	h := &tar.Header{Name: "pkg/pkg.eo", Size: 0, Mode: 0644}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRegistryDownloader_Download_Integration(t *testing.T) {
	ctx := context.Background()
	tarball := minimalTarGz(t)

	// 启动 mock 注册表：versions、metadata、tarball
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

	client := NewHTTPRegistryClient(server.URL, WithRegistryCacheTTL(0))
	httpDL := NewHttpDownloader()
	regDL := NewRegistryDownloader(client, server.URL, httpDL)

	dir := t.TempDir()
	targetDir := filepath.Join(dir, "vendor", "reg-pkg@1.0.0")
	err := regDL.Download(ctx, "reg-pkg", "1.0.0", targetDir, nil)
	if err != nil {
		t.Fatalf("RegistryDownloader.Download: %v", err)
	}
	// 解压后应存在 pkg.eo
	pkgPath := filepath.Join(targetDir, "pkg.eo")
	if _, err := os.Stat(pkgPath); err != nil {
		t.Errorf("expected pkg.eo after download: %v", err)
	}
}
