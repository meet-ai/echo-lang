// Package config 版本号规范化与比较（T-DEV-022 / T-DEV-023）
// 使用场景：echo.toml 版本约束解析、依赖版本冲突检测、lock 校验。
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseVersion 将版本字符串解析为三段整数 MAJOR.MINOR.PATCH，缺段补 0。
// 支持 "1"、"1.2"、"1.2.3"；首版仅支持 x.y.z，不支持预发布/元数据（如 1.0.0-alpha）。
func ParseVersion(version string) (major, minor, patch int, err error) {
	s := strings.TrimSpace(version)
	if s == "" {
		return 0, 0, 0, fmt.Errorf("version is empty")
	}
	parts := strings.Split(s, ".")
	if len(parts) > 3 {
		return 0, 0, 0, fmt.Errorf("version has more than three segments: %s", version)
	}
	for i := 0; i < 3; i++ {
		var v int
		if i < len(parts) {
			part := strings.TrimSpace(parts[i])
			if part == "" {
				return 0, 0, 0, fmt.Errorf("version segment is empty: %s", version)
			}
			v, err = strconv.Atoi(part)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("version segment is not numeric: %s", version)
			}
			if v < 0 {
				return 0, 0, 0, fmt.Errorf("version segment is negative: %s", version)
			}
		}
		switch i {
		case 0:
			major = v
		case 1:
			minor = v
		case 2:
			patch = v
		}
	}
	return major, minor, patch, nil
}

// Normalize 将版本号规范化为 "MAJOR.MINOR.PATCH" 形式，缺段补 0。
// 例如 "1" -> "1.0.0", "1.2" -> "1.2.0", "1.2.3" -> "1.2.3"。
func Normalize(version string) (string, error) {
	major, minor, patch, err := ParseVersion(version)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

// Less 比较版本 a < b（按 MAJOR、MINOR、PATCH 整型比较）。
// 若任一侧解析失败，返回 false（不改变排序结果）。
func Less(a, b string) bool {
	ma, na, pa, errA := ParseVersion(a)
	mb, nb, pb, errB := ParseVersion(b)
	if errA != nil || errB != nil {
		return false
	}
	if ma != mb {
		return ma < mb
	}
	if na != nb {
		return na < nb
	}
	return pa < pb
}

// Equal 比较版本 a == b（规范化后逐段相等）。
// 若任一侧解析失败，返回 false。
func Equal(a, b string) bool {
	ma, na, pa, errA := ParseVersion(a)
	mb, nb, pb, errB := ParseVersion(b)
	if errA != nil || errB != nil {
		return false
	}
	return ma == mb && na == nb && pa == pb
}
