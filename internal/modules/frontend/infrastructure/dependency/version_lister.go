// Package dependency 提供依赖候选版本列表能力（T-DEV-023 / T-DEV-024）
// 使用场景：当 echo.toml 使用范围约束且无 lock 时，需候选版本列表以调用 ResolveVersions；首版支持从 Git tag 获取。
package dependency

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"echo/internal/modules/frontend/infrastructure/config"
)

// VersionLister 候选版本列表提供者
// 按 source 类型（git、http 等）返回某依赖的可用版本列表，供 ResolveVersions 使用。
type VersionLister interface {
	// ListCandidateVersions 返回该依赖的候选版本列表（通常从 Git tag 或注册表获取）
	// 若无法获取（如 http 源无版本列表）返回 nil slice 与 nil error，调用方应报错提示使用精确版本或 lock。
	ListCandidateVersions(ctx context.Context, packageName, source string, options map[string]interface{}) ([]string, error)
}

// DefaultVersionLister 默认实现：Git 源通过 git ls-remote --tags 取 tag 列表，http/https 返回空
type DefaultVersionLister struct{}

// NewDefaultVersionLister 创建默认版本列表器
func NewDefaultVersionLister() *DefaultVersionLister {
	return &DefaultVersionLister{}
}

// ListCandidateVersions 实现 VersionLister
func (d *DefaultVersionLister) ListCandidateVersions(ctx context.Context, packageName, source string, options map[string]interface{}) ([]string, error) {
	switch source {
	case "git":
		return d.listGitTags(ctx, options)
	case "http", "https":
		return nil, nil // 无 tag 列表，调用方需使用精确版本或 lock
	case "registry":
		return nil, nil // 由 RegistryVersionLister 处理
	default:
		return nil, nil
	}
}

// RegistryVersionLister 当 source 为 registry 时使用 PackageRegistry.ListVersions，否则委托给 fallback（T-DEV-024）
type RegistryVersionLister struct {
	Registry PackageRegistry
	Fallback VersionLister
}

// NewRegistryVersionLister 创建支持 registry 的版本列表器
func NewRegistryVersionLister(registry PackageRegistry, fallback VersionLister) *RegistryVersionLister {
	return &RegistryVersionLister{Registry: registry, Fallback: fallback}
}

// ListCandidateVersions 实现 VersionLister
func (r *RegistryVersionLister) ListCandidateVersions(ctx context.Context, packageName, source string, options map[string]interface{}) ([]string, error) {
	if source == "registry" && r.Registry != nil {
		return r.Registry.ListVersions(ctx, packageName)
	}
	if r.Fallback != nil {
		return r.Fallback.ListCandidateVersions(ctx, packageName, source, options)
	}
	return nil, nil
}

// tagRefPrefix 匹配 refs/tags/xxx 或 refs/tags/xxx^{}
var tagRefPrefix = regexp.MustCompile(`^refs/tags/(.+?)(\^{})?$`)

func (d *DefaultVersionLister) listGitTags(ctx context.Context, options map[string]interface{}) ([]string, error) {
	gitURL := ""
	if options != nil {
		if u, ok := options["git"].(string); ok && u != "" {
			gitURL = u
		}
	}
	if gitURL == "" {
		return nil, fmt.Errorf("git URL required to list tags")
	}

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--tags", "--refs", gitURL)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-remote --tags: %w", err)
	}

	seen := make(map[string]struct{})
	var versions []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ref := fields[1]
		m := tagRefPrefix.FindStringSubmatch(ref)
		if m == nil {
			continue
		}
		tag := m[1]
		// 去掉常见的 v 前缀便于与 semver 比较
		v := strings.TrimPrefix(tag, "v")
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		versions = append(versions, v)
	}

	// 按版本号升序排序，ResolveVersions 内用 MaxSatisfying 会选最大满足版本
	sort.Slice(versions, func(i, j int) bool {
		return config.Less(versions[i], versions[j])
	})
	return versions, nil
}
