// Package dependency 版本解析与冲突检测（T-DEV-023）
// 使用场景：给定直接依赖约束与候选版本列表，输出选定版本或可读冲突错误。
package dependency

import (
	"fmt"

	"echo/internal/modules/frontend/infrastructure/config"
)

// ResolveVersions 根据直接依赖约束与每包候选版本，解析出每包的选定版本。
// 若某包无候选或无满足约束的版本，返回包含包名与约束信息的可读 error。
func ResolveVersions(
	directDeps map[string]config.Constraint,
	candidates map[string][]string,
) (resolved map[string]string, err error) {
	resolved = make(map[string]string)
	for pkg, c := range directDeps {
		cands := candidates[pkg]
		if len(cands) == 0 {
			return nil, fmt.Errorf("version resolve: package %q has no candidate versions", pkg)
		}
		sel := config.MaxSatisfying(cands, c)
		if sel == "" {
			return nil, fmt.Errorf("version conflict: package %q has no satisfying version in candidates %v", pkg, cands)
		}
		resolved[pkg] = sel
	}
	return resolved, nil
}
