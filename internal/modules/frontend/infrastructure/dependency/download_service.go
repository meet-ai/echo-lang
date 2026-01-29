// Package dependency 实现依赖包下载服务
package dependency

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"echo/internal/modules/frontend/infrastructure/config"
)

// DownloadService 依赖下载服务
// 职责：协调依赖包的下载、存储和管理；存在 echo.lock 时优先使用锁定版本以保证可重复构建
type DownloadService struct {
	projectRoot   string
	config        *config.EchoConfig
	lock          *config.LockFile
	versionLister VersionLister  // 可选：无 lock 且为范围约束时用于获取候选版本（T-DEV-023）
	registry      PackageRegistry // 可选：source=registry 时用于 ListVersions 与下载（T-DEV-024）
}

// NewDownloadService 创建依赖下载服务；若存在 echo.lock 则加载并优先使用锁定版本
// 若 echo.toml 配置了 [registry] url，则创建 HTTP 注册表客户端并支持 source=registry（T-DEV-024）
func NewDownloadService(projectRoot string) (*DownloadService, error) {
	cfg, err := config.LoadConfig(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	lock, _ := config.LoadLock(projectRoot)
	var lister VersionLister
	var reg PackageRegistry
	if url := cfg.GetRegistryURL(); url != "" {
		reg = NewHTTPRegistryClient(url)
		lister = NewRegistryVersionLister(reg, NewDefaultVersionLister())
	} else {
		lister = NewDefaultVersionLister()
	}
	return &DownloadService{
		projectRoot:   projectRoot,
		config:       cfg,
		lock:         lock,
		versionLister: lister,
		registry:     reg,
	}, nil
}

// NewDownloadServiceWithVersionLister 创建依赖下载服务并注入候选版本列表器（测试或注册表扩展用）
func NewDownloadServiceWithVersionLister(projectRoot string, lister VersionLister) (*DownloadService, error) {
	cfg, err := config.LoadConfig(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	lock, _ := config.LoadLock(projectRoot)
	var reg PackageRegistry
	if url := cfg.GetRegistryURL(); url != "" {
		reg = NewHTTPRegistryClient(url)
	}
	return &DownloadService{
		projectRoot:   projectRoot,
		config:       cfg,
		lock:         lock,
		versionLister: lister,
		registry:     reg,
	}, nil
}

// getResolvedVersion 返回用于下载/路径的版本：有 lock 且该包已锁定则用锁定版本，否则用 echo.toml 中的版本（精确版本或约束解析后的单版本）。
// 当无 lock 且 echo.toml 为范围约束时返回空字符串，调用方需通过 ResolveVersions + 候选版本得到选定版本后再下载（T-DEV-023）。
func (s *DownloadService) getResolvedVersion(packageName string) string {
	if s.lock != nil {
		if v := s.lock.GetLockedVersion(packageName); v != "" {
			return v
		}
	}
	raw := s.config.GetDependencyVersion(packageName)
	if raw == "" {
		return "latest"
	}
	c, err := config.ParseVersionConstraint(raw)
	if err != nil {
		return raw // 解析失败时退回原始字符串，由下载器处理
	}
	if exact, ok := c.Exact(); ok {
		return exact
	}
	// 范围约束且无 lock：需候选版本列表 + ResolveVersions，此处无法解析，返回空表示“需解析”
	return ""
}

// resolveVersionWithCandidates 在无 lock 且 echo.toml 为范围约束时，通过 VersionLister 取候选版本并调用 ResolveVersions 得到选定版本（T-DEV-023 D1-12）。
// 若无法获取候选或无满足版本则返回 ("", error)；成功则返回 (version, nil)。
func (s *DownloadService) resolveVersionWithCandidates(ctx context.Context, packageName string) (string, error) {
	raw := s.config.GetDependencyVersion(packageName)
	if raw == "" {
		return "", nil
	}
	c, err := config.ParseVersionConstraint(raw)
	if err != nil {
		return "", fmt.Errorf("invalid version constraint for %s: %w", packageName, err)
	}
	if s.versionLister == nil {
		return "", nil
	}
	source := s.config.GetDependencySource(packageName)
	options := s.buildDownloadOptions(packageName)
	candidates, err := s.versionLister.ListCandidateVersions(ctx, packageName, source, options)
	if err != nil {
		return "", fmt.Errorf("list candidate versions for %s: %w", packageName, err)
	}
	if len(candidates) == 0 {
		return "", nil
	}
	directDeps := map[string]config.Constraint{packageName: c}
	candMap := map[string][]string{packageName: candidates}
	resolved, err := ResolveVersions(directDeps, candMap)
	if err != nil {
		return "", err
	}
	return resolved[packageName], nil
}

// ValidateLockAgainstConfig 校验 echo.lock 中已锁定版本是否仍满足 echo.toml 的版本约束（T-DEV-023）。
// 若不满足则返回可读错误，提示运行 echoc fetch 或删除 echo.lock 后重试。
func (s *DownloadService) ValidateLockAgainstConfig() error {
	if s.lock == nil {
		return nil
	}
	for packageName := range s.config.Dependencies {
		locked := s.lock.GetLockedVersion(packageName)
		if locked == "" {
			continue
		}
		raw := s.config.GetDependencyVersion(packageName)
		if raw == "" {
			continue
		}
		c, err := config.ParseVersionConstraint(raw)
		if err != nil {
			return fmt.Errorf("invalid version constraint for %s: %w", packageName, err)
		}
		if !c.SatisfiedBy(locked) {
			return fmt.Errorf("lock file out of date: package %s locked at %s does not satisfy echo.toml constraint %q; run echoc fetch or remove echo.lock and retry", packageName, locked, raw)
		}
	}
	return nil
}

// DownloadDependency 下载单个依赖包；成功后写入 echo.lock 以锁定该依赖版本
func (s *DownloadService) DownloadDependency(ctx context.Context, packageName string) error {
	// 检查依赖是否存在
	if !s.config.HasDependency(packageName) {
		return fmt.Errorf("dependency %s not found in echo.toml", packageName)
	}

	// 使用锁定版本（存在 echo.lock）或 echo.toml 中的版本
	version := s.getResolvedVersion(packageName)
	if version == "" {
		// 范围约束且无 lock：尝试通过 VersionLister 获取候选版本并解析（T-DEV-023 D1-12）
		var resolveErr error
		version, resolveErr = s.resolveVersionWithCandidates(ctx, packageName)
		if resolveErr != nil {
			return resolveErr
		}
		if version == "" {
			raw := s.config.GetDependencyVersion(packageName)
			return fmt.Errorf("dependency %s has version constraint %q but no lock entry and no candidate versions; use exact version in echo.toml or ensure git source has tags", packageName, raw)
		}
	}
	source := s.config.GetDependencySource(packageName)

	// 如果没有指定 source，跳过下载（可能是本地依赖）
	if source == "" {
		return fmt.Errorf("dependency %s has no source specified, skipping download", packageName)
	}

	// 存储路径按解析后的版本计算
	storagePath := config.GetDependencyStoragePathForVersion(packageName, version)
	fullStoragePath := filepath.Join(s.projectRoot, storagePath)

	// 检查是否已存在
	if _, err := os.Stat(fullStoragePath); err == nil {
		// 已存在，跳过下载（lock 已存在则保持，否则不写 lock）
		return nil
	}

	// 获取下载器（含 source=registry 时使用 PackageRegistry，T-DEV-024）
	downloader, err := s.getDownloader(source)
	if err != nil {
		return fmt.Errorf("failed to get downloader for source %s: %w", source, err)
	}

	// 构建下载选项
	options := s.buildDownloadOptions(packageName)

	// 执行下载
	if err := downloader.Download(ctx, packageName, version, fullStoragePath, options); err != nil {
		return fmt.Errorf("failed to download dependency %s: %w", packageName, err)
	}

	// 下载成功后写入 echo.lock
	if err := s.writeLockFor(packageName, version); err != nil {
		return fmt.Errorf("failed to update echo.lock: %w", err)
	}
	return nil
}

// writeLockFor 更新内存中的 lock 并写回 echo.lock（追加或创建该依赖的锁定版本）
func (s *DownloadService) writeLockFor(packageName, version string) error {
	if s.lock == nil {
		s.lock = &config.LockFile{SchemaVersion: "1", Dependencies: make(map[string]string)}
	}
	s.lock.SetLockedVersion(packageName, version)
	return config.SaveLock(s.projectRoot, s.lock)
}

// DownloadAllDependencies 下载所有依赖包
func (s *DownloadService) DownloadAllDependencies(ctx context.Context) error {
	if len(s.config.Dependencies) == 0 {
		return nil // 没有依赖，直接返回
	}
	// fetch 前 lock 校验：若存在 echo.lock，校验每条锁定版本满足当前 echo.toml 约束（T-DEV-023）
	if err := s.ValidateLockAgainstConfig(); err != nil {
		return err
	}

	var errors []error
	for packageName := range s.config.Dependencies {
		if err := s.DownloadDependency(ctx, packageName); err != nil {
			errors = append(errors, fmt.Errorf("failed to download %s: %w", packageName, err))
			// 继续下载其他依赖，不中断
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to download some dependencies: %v", errors)
	}

	return nil
}

// getDownloader 根据 source 返回下载器；source=registry 时使用 s.registry（T-DEV-024）
func (s *DownloadService) getDownloader(source string) (Downloader, error) {
	if source == "registry" {
		if s.registry == nil {
			return nil, fmt.Errorf("registry not configured: add [registry] url in echo.toml for source=registry")
		}
		return NewRegistryDownloader(s.registry, s.config.GetRegistryURL(), NewHttpDownloader()), nil
	}
	return GetDownloader(source)
}

// buildDownloadOptions 构建下载选项
func (s *DownloadService) buildDownloadOptions(packageName string) map[string]interface{} {
	options := make(map[string]interface{})

	dep := s.config.Dependencies[packageName]
	if dep == nil {
		return options
	}

	// 如果依赖是对象，提取所有字段
	if depMap, ok := dep.(map[string]interface{}); ok {
		// 复制所有字段到 options
		for k, v := range depMap {
			options[k] = v
		}
	}

	return options
}

// GetDependencyStoragePath 获取依赖包的存储路径（完整路径）；存在 echo.lock 时使用锁定版本对应的路径
func (s *DownloadService) GetDependencyStoragePath(packageName string) string {
	version := s.getResolvedVersion(packageName)
	storagePath := config.GetDependencyStoragePathForVersion(packageName, version)
	return filepath.Join(s.projectRoot, storagePath)
}
