package config

import (
	"testing"
)

func TestParseVersionConstraint_Exact(t *testing.T) {
	c, err := ParseVersionConstraint("1.0.0")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}
	exact, ok := c.Exact()
	if !ok || exact != "1.0.0" {
		t.Errorf("Exact() = %q, %v want 1.0.0, true", exact, ok)
	}
	if !c.SatisfiedBy("1.0.0") {
		t.Error("SatisfiedBy(1.0.0) want true")
	}
	if c.SatisfiedBy("1.0.1") {
		t.Error("SatisfiedBy(1.0.1) want false")
	}
}

func TestParseVersionConstraint_Range(t *testing.T) {
	c, err := ParseVersionConstraint(">=1.0, <2")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}
	if _, ok := c.Exact(); ok {
		t.Error("Exact() want false for range")
	}
	if !c.SatisfiedBy("1.5.0") {
		t.Error("SatisfiedBy(1.5.0) want true")
	}
	if c.SatisfiedBy("2.0.0") {
		t.Error("SatisfiedBy(2.0.0) want false")
	}
	if !c.SatisfiedBy("1.0.0") {
		t.Error("SatisfiedBy(1.0.0) want true")
	}
	if c.SatisfiedBy("0.9.0") {
		t.Error("SatisfiedBy(0.9.0) want false")
	}
}

func TestParseVersionConstraint_Caret(t *testing.T) {
	c, err := ParseVersionConstraint("^1.2.3")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}
	if !c.SatisfiedBy("1.2.3") {
		t.Error("^1.2.3 SatisfiedBy(1.2.3) want true")
	}
	if !c.SatisfiedBy("1.9.0") {
		t.Error("^1.2.3 SatisfiedBy(1.9.0) want true")
	}
	if c.SatisfiedBy("2.0.0") {
		t.Error("^1.2.3 SatisfiedBy(2.0.0) want false")
	}
	if c.SatisfiedBy("1.2.2") {
		t.Error("^1.2.3 SatisfiedBy(1.2.2) want false")
	}
}

func TestParseVersionConstraint_Tilde(t *testing.T) {
	c, err := ParseVersionConstraint("~1.2.3")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}
	if !c.SatisfiedBy("1.2.3") {
		t.Error("~1.2.3 SatisfiedBy(1.2.3) want true")
	}
	if !c.SatisfiedBy("1.2.9") {
		t.Error("~1.2.3 SatisfiedBy(1.2.9) want true")
	}
	if c.SatisfiedBy("1.3.0") {
		t.Error("~1.2.3 SatisfiedBy(1.3.0) want false")
	}
	if c.SatisfiedBy("1.2.2") {
		t.Error("~1.2.3 SatisfiedBy(1.2.2) want false")
	}
}

func TestParseVersionConstraint_Invalid(t *testing.T) {
	_, err := ParseVersionConstraint("")
	if err == nil {
		t.Error("empty constraint want error")
	}
	_, err = ParseVersionConstraint(">=x")
	if err == nil {
		t.Error(">=x want error")
	}
	_, err = ParseVersionConstraint(">=1.0, ")
	if err == nil {
		t.Error("trailing comma want error")
	}
}

func TestMaxSatisfying(t *testing.T) {
	c, _ := ParseVersionConstraint(">=1.0, <2")
	candidates := []string{"0.9.0", "1.0.0", "1.5.0", "2.0.0", "1.9.0"}
	got := MaxSatisfying(candidates, c)
	if got != "1.9.0" {
		t.Errorf("MaxSatisfying(>=1.0, <2) = %q want 1.9.0", got)
	}
}

func TestMaxSatisfying_None(t *testing.T) {
	c, _ := ParseVersionConstraint(">=2.0")
	candidates := []string{"1.0.0", "1.5.0"}
	got := MaxSatisfying(candidates, c)
	if got != "" {
		t.Errorf("MaxSatisfying(>=2.0) with [1.0, 1.5] = %q want \"\"", got)
	}
}

func TestMaxSatisfying_Exact(t *testing.T) {
	c, _ := ParseVersionConstraint("1.2.3")
	candidates := []string{"1.0.0", "1.2.3", "2.0.0"}
	got := MaxSatisfying(candidates, c)
	if got != "1.2.3" {
		t.Errorf("MaxSatisfying(1.2.3) = %q want 1.2.3", got)
	}
}
