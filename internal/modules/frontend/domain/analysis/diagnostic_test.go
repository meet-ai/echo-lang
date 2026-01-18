package analysis

import (
	"testing"
)

func TestNewDiagnostic(t *testing.T) {
	pos := NewPosition(1, 5, "test.eo")
	diag := NewDiagnostic(Error, "undefined variable", pos)

	if diag.Type() != Error {
		t.Errorf("Expected type Error, got %v", diag.Type())
	}

	if diag.Message() != "undefined variable" {
		t.Errorf("Expected message 'undefined variable', got '%s'", diag.Message())
	}

	if !diag.Position().Equals(pos) {
		t.Errorf("Expected position %v, got %v", pos, diag.Position())
	}

	if diag.Code() != "" {
		t.Errorf("Expected empty code, got '%s'", diag.Code())
	}
}

func TestNewDiagnosticWithCode(t *testing.T) {
	pos := NewPosition(1, 5, "test.eo")
	diag := NewDiagnosticWithCode(Warning, "unused variable", pos, "UNUSED_VAR")

	if diag.Type() != Warning {
		t.Errorf("Expected type Warning, got %v", diag.Type())
	}

	if diag.Message() != "unused variable" {
		t.Errorf("Expected message 'unused variable', got '%s'", diag.Message())
	}

	if diag.Code() != "UNUSED_VAR" {
		t.Errorf("Expected code 'UNUSED_VAR', got '%s'", diag.Code())
	}
}

func TestDiagnostic_IsError(t *testing.T) {
	tests := []struct {
		name     string
		diag     Diagnostic
		expected bool
	}{
		{"error diagnostic", NewDiagnostic(Error, "error message", NewPosition(1, 1, "test.eo")), true},
		{"warning diagnostic", NewDiagnostic(Warning, "warning message", NewPosition(1, 1, "test.eo")), false},
		{"info diagnostic", NewDiagnostic(Info, "info message", NewPosition(1, 1, "test.eo")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diag.IsError(); got != tt.expected {
				t.Errorf("Diagnostic.IsError() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestDiagnostic_IsWarning(t *testing.T) {
	tests := []struct {
		name     string
		diag     Diagnostic
		expected bool
	}{
		{"warning diagnostic", NewDiagnostic(Warning, "warning message", NewPosition(1, 1, "test.eo")), true},
		{"error diagnostic", NewDiagnostic(Error, "error message", NewPosition(1, 1, "test.eo")), false},
		{"info diagnostic", NewDiagnostic(Info, "info message", NewPosition(1, 1, "test.eo")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diag.IsWarning(); got != tt.expected {
				t.Errorf("Diagnostic.IsWarning() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestDiagnostic_IsInfo(t *testing.T) {
	tests := []struct {
		name     string
		diag     Diagnostic
		expected bool
	}{
		{"info diagnostic", NewDiagnostic(Info, "info message", NewPosition(1, 1, "test.eo")), true},
		{"error diagnostic", NewDiagnostic(Error, "error message", NewPosition(1, 1, "test.eo")), false},
		{"warning diagnostic", NewDiagnostic(Warning, "warning message", NewPosition(1, 1, "test.eo")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diag.IsInfo(); got != tt.expected {
				t.Errorf("Diagnostic.IsInfo() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestDiagnostic_String(t *testing.T) {
	tests := []struct {
		name     string
		diag     Diagnostic
		expected string
	}{
		{
			"error without code",
			NewDiagnostic(Error, "undefined variable", NewPosition(1, 5, "test.eo")),
			"[error] undefined variable at test.eo:1:5",
		},
		{
			"warning with code",
			NewDiagnosticWithCode(Warning, "unused variable", NewPosition(2, 10, "main.eo"), "UNUSED_VAR"),
			"[warning:UNUSED_VAR] unused variable at main.eo:2:10",
		},
		{
			"info diagnostic",
			NewDiagnostic(Info, "compilation completed", NewPosition(1, 1, "test.eo")),
			"[info] compilation completed at test.eo:1:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diag.String(); got != tt.expected {
				t.Errorf("Diagnostic.String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestDiagnostic_Equals(t *testing.T) {
	pos1 := NewPosition(1, 5, "test.eo")
	pos2 := NewPosition(1, 5, "test.eo")
	pos3 := NewPosition(2, 5, "test.eo")

	diag1 := NewDiagnostic(Error, "undefined variable", pos1)
	diag2 := NewDiagnostic(Error, "undefined variable", pos2)
	diag3 := NewDiagnostic(Warning, "undefined variable", pos1)
	diag4 := NewDiagnostic(Error, "different message", pos1)
	diag5 := NewDiagnostic(Error, "undefined variable", pos3)
	diag6 := NewDiagnosticWithCode(Error, "undefined variable", pos1, "UNDEFINED_VAR")
	diag7 := NewDiagnosticWithCode(Error, "undefined variable", pos1, "DIFFERENT_CODE")

	tests := []struct {
		name     string
		diag1    Diagnostic
		diag2    Diagnostic
		expected bool
	}{
		{"identical diagnostics", diag1, diag2, true},
		{"different type", diag1, diag3, false},
		{"different message", diag1, diag4, false},
		{"different position", diag1, diag5, false},
		{"with and without code", diag1, diag6, false},
		{"different codes", diag6, diag7, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diag1.Equals(tt.diag2); got != tt.expected {
				t.Errorf("Diagnostic.Equals() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
