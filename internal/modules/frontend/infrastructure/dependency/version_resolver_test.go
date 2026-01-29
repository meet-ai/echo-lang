package dependency

import (
	"testing"

	"echo/internal/modules/frontend/infrastructure/config"
)

func TestResolveVersions_AllResolved(t *testing.T) {
	c1, _ := config.ParseVersionConstraint(">=1.0, <2")
	c2, _ := config.ParseVersionConstraint("2.1.0")
	directDeps := map[string]config.Constraint{
		"pkgA": c1,
		"pkgB": c2,
	}
	candidates := map[string][]string{
		"pkgA": {"0.9.0", "1.0.0", "1.5.0", "2.0.0"},
		"pkgB": {"2.0.0", "2.1.0", "2.2.0"},
	}
	resolved, err := ResolveVersions(directDeps, candidates)
	if err != nil {
		t.Fatalf("ResolveVersions: %v", err)
	}
	if resolved["pkgA"] != "1.5.0" {
		t.Errorf("resolved pkgA = %q want 1.5.0", resolved["pkgA"])
	}
	if resolved["pkgB"] != "2.1.0" {
		t.Errorf("resolved pkgB = %q want 2.1.0", resolved["pkgB"])
	}
}

func TestResolveVersions_NoCandidates(t *testing.T) {
	c, _ := config.ParseVersionConstraint("1.0.0")
	directDeps := map[string]config.Constraint{"pkgA": c}
	candidates := map[string][]string{
		"pkgA": {},
	}
	_, err := ResolveVersions(directDeps, candidates)
	if err == nil {
		t.Fatal("ResolveVersions want error when no candidates")
	}
	if err != nil && err.Error() == "" {
		t.Error("error message should contain package name")
	}
}

func TestResolveVersions_NoSatisfying(t *testing.T) {
	c, _ := config.ParseVersionConstraint(">=2.0")
	directDeps := map[string]config.Constraint{"pkgA": c}
	candidates := map[string][]string{
		"pkgA": {"1.0.0", "1.5.0"},
	}
	_, err := ResolveVersions(directDeps, candidates)
	if err == nil {
		t.Fatal("ResolveVersions want error when no satisfying version")
	}
}
