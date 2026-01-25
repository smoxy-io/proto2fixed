package analyzer

import (
	"testing"

	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

func TestValidator_ValidSchema(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "Simple",
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
}

func TestValidator_SizeMismatch(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "BadSize",
				Size: 100, // Declared size
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32}, // Only 4 bytes
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected size mismatch error")
	}
}

func TestValidator_MissingArraySize(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "BadArray",
				Fields: []*parser.Field{
					{
						Name:      "values",
						Number:    1,
						Type:      parser.TypeFloat,
						Repeated:  true,
						ArraySize: 0, // Missing!
					},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	// Either err is set (from layout analysis) or result has errors
	if err == nil && !result.HasErrors() {
		t.Error("Expected error for missing array size")
	}
}

func TestValidator_MissingStringSize(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "BadString",
				Fields: []*parser.Field{
					{
						Name:       "name",
						Number:     1,
						Type:       parser.TypeString,
						StringSize: 0, // Missing!
					},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	// Either err is set (from layout analysis) or result has errors
	if err == nil && !result.HasErrors() {
		t.Error("Expected error for missing string size")
	}
}

func TestValidator_InvalidEndian(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "middle", // Invalid!
		Messages: []*parser.Message{
			{
				Name: "Simple",
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for invalid endian")
	}
}

func TestValidator_FieldNumberGaps(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "GappyFields",
				Fields: []*parser.Field{
					{Name: "field1", Number: 1, Type: parser.TypeUint32},
					{Name: "field5", Number: 5, Type: parser.TypeUint32},   // Gap!
					{Name: "field10", Number: 10, Type: parser.TypeUint32}, // Gap!
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Should get warnings, not errors
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for field number gaps")
	}
}

func TestValidator_InvalidEnumSize(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Enums: []*parser.Enum{
			{
				Name: "BadEnum",
				Size: 3, // Invalid! Must be 1, 2, or 4
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for invalid enum size")
	}
}

func TestValidator_EnumValueOutOfRange(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Enums: []*parser.Enum{
			{
				Name: "TinyEnum",
				Size: 1, // uint8, range: -128 to 127
				Values: []*parser.EnumValue{
					{Name: "TOO_BIG", Number: 200}, // Out of range!
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for enum value out of range")
	}
}

func TestValidator_InvalidAlignment(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name:  "BadAlign",
				Align: 3, // Not a power of 2!
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for invalid alignment")
	}
}

func TestValidator_LargeStringWarning(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "BigString",
				Fields: []*parser.Field{
					{
						Name:       "data",
						Number:     1,
						Type:       parser.TypeString,
						StringSize: 2048, // Large!
					},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Should get a warning
	if len(result.Warnings) == 0 {
		t.Error("Expected warning for large string size")
	}
}

func TestValidator_UnionMessage(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name:  "Union",
				Union: true,
				Fields: []*parser.Field{
					{Name: "small", Number: 1, Type: parser.TypeUint32},
					{Name: "large", Number: 2, Type: parser.TypeUint64},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("Union message should validate successfully, got errors: %v", result.Errors)
	}

	// Union size should be 9 (1 byte discriminator + 8 byte largest field)
	layout, _ := validator.GetAnalyzer().GetLayout("Union")
	if layout.TotalSize != 9 {
		t.Errorf("Expected union size 9, got %d", layout.TotalSize)
	}
}

func TestValidator_GetAnalyzer(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "Simple",
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	_, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	analyzer := validator.GetAnalyzer()
	if analyzer == nil {
		t.Fatal("Expected analyzer to be available")
	}

	layout, exists := analyzer.GetLayout("Simple")
	if !exists {
		t.Error("Expected to find Simple message layout")
	}

	if layout.TotalSize != 4 {
		t.Errorf("Expected Simple size 4, got %d", layout.TotalSize)
	}
}

func TestIsPowerOfTwo(t *testing.T) {
	tests := []struct {
		input    uint32
		expected bool
	}{
		{0, false},
		{1, true},
		{2, true},
		{3, false},
		{4, true},
		{5, false},
		{8, true},
		{15, false},
		{16, true},
		{17, false},
		{32, true},
	}

	for _, test := range tests {
		result := isPowerOfTwo(test.input)
		if result != test.expected {
			t.Errorf("isPowerOfTwo(%d) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestEnumValueFitsInSize(t *testing.T) {
	tests := []struct {
		value    int32
		size     uint32
		expected bool
	}{
		{0, 1, true},
		{127, 1, true},
		{128, 1, false},
		{-128, 1, true},
		{-129, 1, false},
		{32767, 2, true},
		{32768, 2, false},
		{-32768, 2, true},
		{-32769, 2, false},
		{2147483647, 4, true},
		{-2147483648, 4, true},
	}

	for _, test := range tests {
		result := enumValueFitsInSize(test.value, test.size)
		if result != test.expected {
			t.Errorf("enumValueFitsInSize(%d, %d) = %v, expected %v",
				test.value, test.size, result, test.expected)
		}
	}
}

// TestValidationError_Error_WithSourcePos tests ValidationError.Error() with source position
func TestValidationError_Error_WithSourcePos(t *testing.T) {
	err := &ValidationError{
		Message:   "Test error message",
		SourcePos: "test.proto:10:5",
	}

	result := err.Error()
	expected := "Error: test.proto:10:5\n    Test error message"
	if result != expected {
		t.Errorf("Expected error format:\n%s\nGot:\n%s", expected, result)
	}
}

// TestValidationError_Error_WithoutSourcePos tests ValidationError.Error() without source position
func TestValidationError_Error_WithoutSourcePos(t *testing.T) {
	err := &ValidationError{
		Message: "Test error message",
	}

	result := err.Error()
	expected := "Error: Test error message"
	if result != expected {
		t.Errorf("Expected error format: %s, got: %s", expected, result)
	}
}

// TestValidationResult_String tests ValidationResult.String() formatting
func TestValidationResult_String(t *testing.T) {
	result := &ValidationResult{
		Errors: []*ValidationError{
			{Message: "First error", SourcePos: "test.proto:5:1"},
			{Message: "Second error"},
		},
		Warnings: []*ValidationWarning{
			{Message: "First warning", SourcePos: "test.proto:10:1"},
		},
	}

	output := result.String()

	// Check that output contains all errors and warnings
	if !contains(output, "First error") {
		t.Error("Output should contain first error")
	}
	if !contains(output, "Second error") {
		t.Error("Output should contain second error")
	}
	if !contains(output, "First warning") {
		t.Error("Output should contain first warning")
	}
	if !contains(output, "test.proto:5:1") {
		t.Error("Output should contain error source position")
	}
	if !contains(output, "test.proto:10:1") {
		t.Error("Output should contain warning source position")
	}
}

// TestValidationResult_String_Empty tests ValidationResult.String() with no errors or warnings
func TestValidationResult_String_Empty(t *testing.T) {
	result := &ValidationResult{
		Errors:   []*ValidationError{},
		Warnings: []*ValidationWarning{},
	}

	output := result.String()
	if output != "" {
		t.Errorf("Expected empty string for result with no errors/warnings, got: %s", output)
	}
}

// TestValidator_MultipleErrors tests that validator accumulates multiple errors
func TestValidator_MultipleErrors(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "invalid-endian", // Error: invalid endian
		Messages: []*parser.Message{
			{
				Name: "Message1",
				Size: 100, // Error: size mismatch (actual size will be 4 bytes)
				Fields: []*parser.Field{
					{
						Name:   "value",
						Number: 1,
						Type:   parser.TypeUint32,
					},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation should not return error: %v", err)
	}

	if !result.HasErrors() {
		t.Fatal("Expected errors to be present")
	}

	// Should have multiple errors (invalid endian + size mismatch)
	if len(result.Errors) < 2 {
		t.Errorf("Expected at least 2 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

// TestLayoutAnalyzer_GetAllLayouts tests GetAllLayouts returns complete map
func TestLayoutAnalyzer_GetAllLayouts(t *testing.T) {
	schema := &parser.Schema{
		Messages: []*parser.Message{
			{
				Name: "Message1",
				Fields: []*parser.Field{
					{Name: "field1", Number: 1, Type: parser.TypeUint32},
				},
			},
			{
				Name: "Message2",
				Fields: []*parser.Field{
					{Name: "field1", Number: 1, Type: parser.TypeUint64},
				},
			},
		},
	}

	analyzer := NewLayoutAnalyzer()
	err := analyzer.Analyze(schema)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	layouts := analyzer.GetAllLayouts()
	if len(layouts) != 2 {
		t.Errorf("Expected 2 layouts, got %d", len(layouts))
	}

	if _, exists := layouts["Message1"]; !exists {
		t.Error("Expected Message1 layout to exist")
	}
	if _, exists := layouts["Message2"]; !exists {
		t.Error("Expected Message2 layout to exist")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestValidator_MessageIds tests message ID validation
func TestValidator_MessageIds(t *testing.T) {
	schema := &parser.Schema{
		Fixed:         true,
		Endian:        "little",
		MessageIdSize: 4,
		Messages: []*parser.Message{
			{
				Name:      "TopLevel1",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
			{
				Name:      "TopLevel2",
				MessageId: 2,
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
}

// TestValidator_DuplicateMessageIds tests duplicate message ID detection
func TestValidator_DuplicateMessageIds(t *testing.T) {
	schema := &parser.Schema{
		Fixed:         true,
		Endian:        "little",
		MessageIdSize: 4,
		Messages: []*parser.Message{
			{
				Name:      "Message1",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
			{
				Name:      "Message2",
				MessageId: 1, // Duplicate!
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for duplicate message IDs")
	}

	if !contains(result.String(), "duplicate message_id") {
		t.Errorf("Expected duplicate message ID error, got: %s", result.String())
	}
}

// TestValidator_MessageIdExceedsMax tests message ID exceeding maximum for size
func TestValidator_MessageIdExceedsMax(t *testing.T) {
	schema := &parser.Schema{
		Fixed:         true,
		Endian:        "little",
		MessageIdSize: 1, // 1 byte, max value = 255
		Messages: []*parser.Message{
			{
				Name:      "Overflow",
				MessageId: 300, // Exceeds max!
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for message ID exceeding maximum")
	}

	if !contains(result.String(), "exceeds maximum") {
		t.Errorf("Expected maximum exceeded error, got: %s", result.String())
	}
}

// TestValidator_NestedMessageWithId tests warning for nested message with message_id
func TestValidator_NestedMessageWithId(t *testing.T) {
	nestedMsg := &parser.Message{
		Name:      "Nested",
		MessageId: 99, // Nested messages shouldn't have IDs
		Fields: []*parser.Field{
			{Name: "value", Number: 1, Type: parser.TypeUint32},
		},
	}

	schema := &parser.Schema{
		Fixed:         true,
		Endian:        "little",
		MessageIdSize: 4,
		Messages: []*parser.Message{
			{
				Name:      "Parent",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "nested", Number: 1, Type: parser.TypeMessage, MessageType: nestedMsg},
				},
			},
			nestedMsg,
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Error("Expected warning for nested message with message_id")
	}

	if !contains(result.String(), "nested") && !contains(result.String(), "message_id") {
		t.Errorf("Expected nested message warning, got: %s", result.String())
	}
}

// TestValidator_MissingMessageId tests warning for top-level message without message_id
func TestValidator_MissingMessageId(t *testing.T) {
	schema := &parser.Schema{
		Fixed:         true,
		Endian:        "little",
		MessageIdSize: 4,
		Messages: []*parser.Message{
			{
				Name:      "WithId",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
			{
				Name:      "WithoutId",
				MessageId: 0, // Missing!
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Error("Expected warning for missing message_id")
	}

	if !contains(result.String(), "WithoutId") {
		t.Errorf("Expected missing message_id warning for WithoutId, got: %s", result.String())
	}
}

// TestValidator_InvalidMessageIdSize tests invalid message_id_size value
func TestValidator_InvalidMessageIdSize(t *testing.T) {
	schema := &parser.Schema{
		Fixed:         true,
		Endian:        "little",
		MessageIdSize: 3, // Invalid! Must be 1, 2, 4, or 8
		Messages: []*parser.Message{
			{
				Name:      "Message",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for invalid message_id_size")
	}

	if !contains(result.String(), "message_id_size") {
		t.Errorf("Expected message_id_size error, got: %s", result.String())
	}
}

// TestValidator_NoMessageIds tests that at least one message must have a message_id
func TestValidator_NoMessageIds(t *testing.T) {
	schema := &parser.Schema{
		Fixed:         true,
		Endian:        "little",
		MessageIdSize: 4,
		Messages: []*parser.Message{
			{
				Name:      "Command",
				MessageId: 0, // No ID
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
			{
				Name:      "Response",
				MessageId: 0, // No ID
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error when no messages have message_id")
	}

	if !contains(result.String(), "at least one") {
		t.Errorf("Expected 'at least one' error, got: %s", result.String())
	}
}

func TestValidator_UnionFieldNumberExceeds255(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name:  "UnionWithLargeFieldNumber",
				Union: true,
				Fields: []*parser.Field{
					{Name: "field1", Number: 1, Type: parser.TypeUint32, OneofIndex: -1},
					{Name: "field256", Number: 256, Type: parser.TypeUint32, OneofIndex: -1}, // Too large
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for union field number > 255")
	}

	if !contains(result.String(), "exceeds 255") {
		t.Errorf("Expected 'exceeds 255' error, got: %s", result.String())
	}
}

func TestValidator_OneofFieldNumberExceeds255(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "MessageWithOneof",
				Oneofs: []*parser.Oneof{
					{
						Name: "value",
						Fields: []*parser.Field{
							{Name: "intValue", Number: 2, Type: parser.TypeUint32},
							{Name: "largeValue", Number: 300, Type: parser.TypeUint32}, // Too large
						},
					},
				},
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32, OneofIndex: -1},
					{Name: "intValue", Number: 2, Type: parser.TypeUint32, OneofIndex: 0},
					{Name: "largeValue", Number: 300, Type: parser.TypeUint32, OneofIndex: 0},
				},
			},
		},
	}

	validator := NewValidator()
	result, err := validator.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for oneof variant number > 255")
	}

	if !contains(result.String(), "exceeds 255") {
		t.Errorf("Expected 'exceeds 255' error, got: %s", result.String())
	}
}
