// Package config 版本约束解析（T-DEV-022 / T-DEV-023）
// 使用场景：echo.toml version 字段解析、SatisfiedBy/Exact、MaxSatisfying 选版。
package config

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Constraint 版本约束：可判断某版本是否满足，或表示精确版本。
type Constraint interface {
	// SatisfiedBy 判断 version 是否满足约束
	SatisfiedBy(version string) bool
	// Exact 若为精确版本则返回版本号与 true，否则返回 "", false
	Exact() (string, bool)
}

// exactConstraint 精确版本约束，仅接受单一版本
type exactConstraint struct {
	version string // 已规范化，如 "1.0.0"
}

func (c *exactConstraint) SatisfiedBy(version string) bool {
	return Equal(c.version, version)
}

func (c *exactConstraint) Exact() (string, bool) {
	return c.version, true
}

// pred 单条比较谓词：op 为 ">=" "<=" ">" "<"，v 为已规范化版本
type pred struct {
	op string
	v  string
}

// rangeConstraint 范围约束，由多条谓词取交集
type rangeConstraint struct {
	preds []pred
}

func (c *rangeConstraint) SatisfiedBy(version string) bool {
	for _, p := range c.preds {
		if !satisfyPred(p.op, p.v, version) {
			return false
		}
	}
	return true
}

func (c *rangeConstraint) Exact() (string, bool) {
	return "", false
}

func satisfyPred(op, bound, version string) bool {
	switch op {
	case ">=":
		return Equal(bound, version) || Less(bound, version)
	case "<=":
		return Equal(bound, version) || Less(version, bound)
	case ">":
		return Less(bound, version)
	case "<":
		return Less(version, bound)
	default:
		return false
	}
}

// 仅数字与点的模式，用于识别“精确版本”写法
var exactVersionPattern = regexp.MustCompile(`^\s*[\d.]+\s*$`)

// ParseVersionConstraint 将 echo.toml 的 version 字符串解析为 Constraint。
// 支持：精确版本 "1.0.0"/"1"/"1.2"；比较 ">=1.0" "<2"；多约束 ">=1.0, <2"；^1.2.3 ~1.2.3（与 T-DEV-022 一致）。
func ParseVersionConstraint(raw string) (Constraint, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, fmt.Errorf("version constraint is empty")
	}

	// ^1.2.3 -> >=1.2.3, <2.0.0
	if strings.HasPrefix(s, "^") {
		v := strings.TrimSpace(s[1:])
		maj, min, patch, err := ParseVersion(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ^ version: %w", err)
		}
		lower := fmt.Sprintf("%d.%d.%d", maj, min, patch)
		upper := fmt.Sprintf("%d.0.0", maj+1)
		return &rangeConstraint{preds: []pred{{">=", lower}, {"<", upper}}}, nil
	}

	// ~1.2.3 -> >=1.2.3, <1.3.0
	if strings.HasPrefix(s, "~") {
		v := strings.TrimSpace(s[1:])
		maj, min, patch, err := ParseVersion(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ~ version: %w", err)
		}
		lower := fmt.Sprintf("%d.%d.%d", maj, min, patch)
		upper := fmt.Sprintf("%d.%d.0", maj, min+1)
		return &rangeConstraint{preds: []pred{{">=", lower}, {"<", upper}}}, nil
	}

	// 含逗号：多约束取交集
	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		var preds []pred
		for _, p := range parts {
			op, v, err := parseSinglePred(strings.TrimSpace(p))
			if err != nil {
				return nil, err
			}
			preds = append(preds, pred{op: op, v: v})
		}
		return &rangeConstraint{preds: preds}, nil
	}

	// 单条比较 >= / <= / > / <
	if strings.HasPrefix(s, ">=") || strings.HasPrefix(s, "<=") || strings.HasPrefix(s, ">") || strings.HasPrefix(s, "<") {
		op, v, err := parseSinglePred(s)
		if err != nil {
			return nil, err
		}
		return &rangeConstraint{preds: []pred{{op, v}}}, nil
	}

	// 仅数字与点：精确版本
	if exactVersionPattern.MatchString(s) {
		norm, err := Normalize(s)
		if err != nil {
			return nil, fmt.Errorf("invalid exact version: %w", err)
		}
		return &exactConstraint{version: norm}, nil
	}

	return nil, fmt.Errorf("unrecognized version constraint: %s", raw)
}

// parseSinglePred 解析单条比较，如 ">=1.0" "<2" ">1.2.3"
func parseSinglePred(s string) (op string, version string, err error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return "", "", fmt.Errorf("invalid predicate: %s", s)
	}
	if strings.HasPrefix(s, ">=") {
		v := strings.TrimSpace(s[2:])
		norm, err := Normalize(v)
		if err != nil {
			return "", "", err
		}
		return ">=", norm, nil
	}
	if strings.HasPrefix(s, "<=") {
		v := strings.TrimSpace(s[2:])
		norm, err := Normalize(v)
		if err != nil {
			return "", "", err
		}
		return "<=", norm, nil
	}
	if strings.HasPrefix(s, ">") {
		v := strings.TrimSpace(s[1:])
		norm, err := Normalize(v)
		if err != nil {
			return "", "", err
		}
		return ">", norm, nil
	}
	if strings.HasPrefix(s, "<") {
		v := strings.TrimSpace(s[1:])
		norm, err := Normalize(v)
		if err != nil {
			return "", "", err
		}
		return "<", norm, nil
	}
	return "", "", fmt.Errorf("invalid predicate: %s", s)
}

// MaxSatisfying 在候选版本中选出满足约束的最大版本；若无满足则返回空字符串。
func MaxSatisfying(candidates []string, c Constraint) string {
	var satisfied []string
	for _, v := range candidates {
		if c.SatisfiedBy(v) {
			satisfied = append(satisfied, v)
		}
	}
	if len(satisfied) == 0 {
		return ""
	}
	// 规范化后按 Less 排序，取最后一个为最大
	normalized := make([]string, len(satisfied))
	for i, v := range satisfied {
		n, err := Normalize(v)
		if err != nil {
			continue
		}
		normalized[i] = n
	}
	sort.Slice(normalized, func(i, j int) bool { return Less(normalized[i], normalized[j]) })
	return normalized[len(normalized)-1]
}
