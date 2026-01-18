package impl

import (
	"testing"
)

func TestTypeMapperImpl_MapPrimitiveType(t *testing.T) {
	tm := NewTypeMapperImpl()

	tests := []struct {
		input    string
		expected interface{}
		hasError bool
	}{
		{"int", "i32", false},
		{"bool", "i1", false},
		{"string", "i8*", false},
		{"float", "float", false},
		{"unknown", nil, true},
	}

	for _, test := range tests {
		result, err := tm.MapPrimitiveType(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input '%s', but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("For input '%s', expected %v, got %v", test.input, test.expected, result)
			}
		}
	}
}

func TestTypeMapperImpl_MapArrayType(t *testing.T) {
	tm := NewTypeMapperImpl()

	// 测试静态大小数组
	result, err := tm.MapArrayType("int", []int{5})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "[5 x i32]"
	if result != expected {
		t.Errorf("Expected %s, got %v", expected, result)
	}

	// 测试动态大小数组
	result, err = tm.MapArrayType("int", []int{-1})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected = "[i32]*"
	if result != expected {
		t.Errorf("Expected %s, got %v", expected, result)
	}
}

func TestTypeMapperImpl_MapStructType(t *testing.T) {
	tm := NewTypeMapperImpl()

	fields := map[string]string{
		"id":   "int",
		"name": "string",
	}

	result, err := tm.MapStructType("Person", fields)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "%Person = type { i32, i8* }"
	if result != expected {
		t.Errorf("Expected %s, got %v", expected, result)
	}
}

func TestTypeMapperImpl_MapFunctionType(t *testing.T) {
	tm := NewTypeMapperImpl()

	result, err := tm.MapFunctionType("int", []string{"int", "bool"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "i32 (i32, i1)"
	if result != expected {
		t.Errorf("Expected %s, got %v", expected, result)
	}
}

func TestTypeMapperImpl_IsCompatible(t *testing.T) {
	tm := NewTypeMapperImpl()

	tests := []struct {
		from     string
		to       string
		expected bool
	}{
		{"int", "int", true},
		{"bool", "bool", true},
		{"int", "bool", false},
		{"string", "int", false},
	}

	for _, test := range tests {
		result := tm.IsCompatible(test.from, test.to)
		if result != test.expected {
			t.Errorf("IsCompatible(%s, %s) = %v, expected %v",
				test.from, test.to, result, test.expected)
		}
	}
}

func TestTypeMapperImpl_GetDefaultValue(t *testing.T) {
	tm := NewTypeMapperImpl()

	tests := []struct {
		input    string
		expected interface{}
	}{
		{"int", "0"},
		{"bool", "false"},
		{"string", `""`},
		{"float", "0.0"},
		{"unknown", "zeroinitializer"},
	}

	for _, test := range tests {
		result := tm.GetDefaultValue(test.input)
		if result != test.expected {
			t.Errorf("GetDefaultValue(%s) = %v, expected %v",
				test.input, result, test.expected)
		}
	}
}

func TestTypeMapperImpl_RegisterCustomType(t *testing.T) {
	tm := NewTypeMapperImpl()

	// 注册自定义类型
	err := tm.RegisterCustomType("MyType", "custom_type")
	if err != nil {
		t.Errorf("Failed to register custom type: %v", err)
	}

	// 查找已注册的类型
	result, exists := tm.LookupType("MyType")
	if !exists {
		t.Error("Custom type should exist after registration")
	}
	if result != "custom_type" {
		t.Errorf("Expected custom_type, got %v", result)
	}

	// 重复注册应该失败
	err = tm.RegisterCustomType("MyType", "another_type")
	if err == nil {
		t.Error("Expected error for duplicate custom type registration")
	}
}

func TestTypeMapperImpl_LookupType(t *testing.T) {
	tm := NewTypeMapperImpl()

	// 查找基本类型
	result, exists := tm.LookupType("int")
	if !exists {
		t.Error("Primitive type 'int' should exist")
	}
	if result != "i32" {
		t.Errorf("Expected i32, got %v", result)
	}

	// 查找不存在的类型
	_, exists = tm.LookupType("nonexistent")
	if exists {
		t.Error("Non-existent type should not be found")
	}
}
