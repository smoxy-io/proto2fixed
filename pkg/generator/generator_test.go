package generator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/smoxy-io/proto2fixed/pkg/analyzer"
	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

func TestJSONGenerator(t *testing.T) {
	schema := &parser.Schema{
		Fixed:   true,
		Endian:  "little",
		Version: "1.0.0",
		Messages: []*parser.Message{
			{
				Name: "Simple",
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
					{Name: "temp", Number: 2, Type: parser.TypeFloat},
				},
			},
		},
	}

	// Analyze layout
	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	// Generate JSON
	gen := NewJSONGenerator()
	output, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("JSON generation failed: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Check protocol field
	if result["protocol"] != "fixed-binary" {
		t.Errorf("Expected protocol 'fixed-binary', got '%v'", result["protocol"])
	}

	// Check endian
	if result["endian"] != "little" {
		t.Errorf("Expected endian 'little', got '%v'", result["endian"])
	}

	// Check version
	if result["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%v'", result["version"])
	}

	// Check messages
	messages, ok := result["messages"].(map[string]any)
	if !ok {
		t.Fatal("Expected messages map")
	}

	simple, ok := messages["Simple"].(map[string]any)
	if !ok {
		t.Fatal("Expected Simple message")
	}

	// Check total size
	if simple["totalSize"] != float64(8) {
		t.Errorf("Expected totalSize 8, got %v", simple["totalSize"])
	}
}

func TestArduinoGenerator(t *testing.T) {
	schema := &parser.Schema{
		FileName: "test.proto",
		Fixed:    true,
		Endian:   "little",
		Messages: []*parser.Message{
			{
				Name: "Simple",
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
					{Name: "flag", Number: 2, Type: parser.TypeBool},
				},
			},
		},
	}

	// Analyze layout
	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	// Generate Arduino code
	gen := NewArduinoGenerator()
	output, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Arduino generation failed: %v", err)
	}

	// Check for key elements
	requiredStrings := []string{
		"#ifndef",
		"#define",
		"#pragma pack(push, 1)",
		"#pragma pack(pop)",
		"typedef struct",
		"Simple",
		"uint32_t value",
		"bool flag",
		"static_assert",
		"encodeSimple",
		"decodeSimple",
		"setFixedString",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(output, required) {
			t.Errorf("Arduino output missing required string: %s", required)
		}
	}
}

func TestArduinoGenerator_Enum(t *testing.T) {
	schema := &parser.Schema{
		FileName: "test.proto",
		Fixed:    true,
		Endian:   "little",
		Enums: []*parser.Enum{
			{
				Name: "Status",
				Size: 1,
				Values: []*parser.EnumValue{
					{Name: "UNKNOWN", Number: 0},
					{Name: "ACTIVE", Number: 1},
				},
			},
		},
		Messages: []*parser.Message{
			{
				Name: "Data",
				Fields: []*parser.Field{
					{Name: "status", Number: 1, Type: parser.TypeEnum, EnumType: &parser.Enum{Name: "Status", Size: 1}},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewArduinoGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("Arduino generation failed: %v", err)
	}

	// Check enum is generated
	if !strings.Contains(output, "typedef enum") {
		t.Error("Arduino output missing enum typedef")
	}
	if !strings.Contains(output, "UNKNOWN = 0") {
		t.Error("Arduino output missing enum value UNKNOWN")
	}
	if !strings.Contains(output, "ACTIVE = 1") {
		t.Error("Arduino output missing enum value ACTIVE")
	}
}

func TestGoGenerator(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Fixed:     true,
		Endian:    "little",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name:      "Simple",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
					{Name: "temperature", Number: 2, Type: parser.TypeFloat},
				},
			},
		},
	}

	// Analyze layout
	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	// Generate Go code
	gen := NewGoGenerator()
	output, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Go generation failed: %v", err)
	}

	// Check for key elements
	requiredStrings := []string{
		"package testpkg",
		"const SimpleSize",
		"type SimpleDecoder struct",
		"NewSimpleDecoder",
		"func (d *SimpleDecoder) Decode",
		"type SimpleEncoder struct",
		"NewSimpleEncoder",
		"func (e *SimpleEncoder) Encode",
		"binary.LittleEndian",
		"type testHelpers struct{}",
		"func (testHelpers) decodeString(",
		"func (testHelpers) encodeString(",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(output, required) {
			t.Errorf("Go output missing required string: %s", required)
		}
	}
}

func TestGoGenerator_BigEndian(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Fixed:     true,
		Endian:    "big", // Big endian
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name:      "Simple",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeUint32},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewGoGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("Go generation failed: %v", err)
	}

	// Check for big endian
	if !strings.Contains(output, "binary.BigEndian") {
		t.Error("Go output should use binary.BigEndian")
	}
}

func TestCommonHelpers(t *testing.T) {
	tests := []struct {
		name     string
		function func(string) string
		input    string
		expected string
	}{
		{"toSnakeCase", toSnakeCase, "CamelCase", "camel_case"},
		{"toSnakeCase", toSnakeCase, "HTTPServer", "h_t_t_p_server"},
		{"toCamelCase", toCamelCase, "snake_case", "snakeCase"},
		{"toCamelCase", toCamelCase, "simple", "simple"},
		{"toPascalCase", toPascalCase, "snake_case", "SnakeCase"},
		{"toPascalCase", toPascalCase, "simple", "Simple"},
	}

	for _, test := range tests {
		result := test.function(test.input)
		if result != test.expected {
			t.Errorf("%s(%s) = %s, expected %s", test.name, test.input, result, test.expected)
		}
	}
}

func TestGetTypeNameC(t *testing.T) {
	tests := []struct {
		fieldType parser.FieldType
		expected  string
	}{
		{parser.TypeBool, "bool"},
		{parser.TypeInt32, "int32_t"},
		{parser.TypeUint32, "uint32_t"},
		{parser.TypeInt64, "int64_t"},
		{parser.TypeUint64, "uint64_t"},
		{parser.TypeFloat, "float"},
		{parser.TypeDouble, "double"},
		{parser.TypeString, "char"},
		{parser.TypeBytes, "uint8_t"},
	}

	for _, test := range tests {
		field := &parser.Field{Type: test.fieldType}
		result := getTypeNameC(field)
		if result != test.expected {
			t.Errorf("getTypeNameC(%v) = %s, expected %s", test.fieldType, result, test.expected)
		}
	}
}

func TestGetTypeNameGo(t *testing.T) {
	tests := []struct {
		fieldType parser.FieldType
		expected  string
	}{
		{parser.TypeBool, "bool"},
		{parser.TypeInt32, "int32"},
		{parser.TypeUint32, "uint32"},
		{parser.TypeInt64, "int64"},
		{parser.TypeUint64, "uint64"},
		{parser.TypeFloat, "float32"},
		{parser.TypeDouble, "float64"},
		{parser.TypeString, "string"},
		{parser.TypeBytes, "[]byte"},
	}

	for _, test := range tests {
		field := &parser.Field{Type: test.fieldType}
		result := getTypeNameGo(field)
		if result != test.expected {
			t.Errorf("getTypeNameGo(%v) = %s, expected %s", test.fieldType, result, test.expected)
		}
	}
}

func TestJSONGenerator_WithArray(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "WithArray",
				Fields: []*parser.Field{
					{
						Name:      "values",
						Number:    1,
						Type:      parser.TypeFloat,
						Repeated:  true,
						ArraySize: 10,
					},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewJSONGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("JSON generation failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	messages := result["messages"].(map[string]any)
	withArray := messages["WithArray"].(map[string]any)
	structure := withArray["structure"].([]any)
	field := structure[0].(map[string]any)

	// Check array properties
	if field["count"] != float64(10) {
		t.Errorf("Expected count 10, got %v", field["count"])
	}
	if field["elementSize"] != float64(4) {
		t.Errorf("Expected elementSize 4, got %v", field["elementSize"])
	}
}

func TestJSONGenerator_WithString(t *testing.T) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "WithString",
				Fields: []*parser.Field{
					{
						Name:       "name",
						Number:     1,
						Type:       parser.TypeString,
						StringSize: 32,
					},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewJSONGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("JSON generation failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	messages := result["messages"].(map[string]any)
	withString := messages["WithString"].(map[string]any)
	structure := withString["structure"].([]any)
	field := structure[0].(map[string]any)

	// Check string encoding
	if field["encoding"] != "null-terminated" {
		t.Errorf("Expected encoding 'null-terminated', got %v", field["encoding"])
	}
	if field["size"] != float64(32) {
		t.Errorf("Expected size 32, got %v", field["size"])
	}
}

// TestArduinoGenerator_NestedMessages tests Arduino generation for nested messages
func TestArduinoGenerator_NestedMessages(t *testing.T) {
	innerMsg := &parser.Message{
		Name: "Inner",
		Fields: []*parser.Field{
			{Name: "value", Number: 1, Type: parser.TypeUint32},
		},
	}

	schema := &parser.Schema{
		Messages: []*parser.Message{
			innerMsg,
			{
				Name: "Outer",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "nested", Number: 2, Type: parser.TypeMessage, MessageType: innerMsg},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewArduinoGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("Arduino generation failed: %v", err)
	}

	// Check that both structs are generated
	if !strings.Contains(output, "typedef struct") {
		t.Error("Expected typedef struct")
	}
	if !strings.Contains(output, "Inner") {
		t.Error("Expected Inner type definition")
	}
	if !strings.Contains(output, "Outer") {
		t.Error("Expected Outer type definition")
	}
}

// TestArduinoGenerator_BytesField tests Arduino generation for bytes fields
func TestArduinoGenerator_BytesField(t *testing.T) {
	schema := &parser.Schema{
		Messages: []*parser.Message{
			{
				Name: "BytesMessage",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "data", Number: 2, Type: parser.TypeBytes, ArraySize: 64},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewArduinoGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("Arduino generation failed: %v", err)
	}

	// Check that bytes field is generated as uint8_t array
	if !strings.Contains(output, "uint8_t") {
		t.Error("Expected uint8_t for bytes field")
	}
	// Check that data field exists (may be formatted as "data[64]" or just "data")
	if !strings.Contains(output, "data") {
		t.Error("Expected data field in output")
	}
}

// TestArduinoGenerator_UnionMessage tests Arduino union generation
func TestArduinoGenerator_UnionMessage(t *testing.T) {
	schema := &parser.Schema{
		Messages: []*parser.Message{
			{
				Name:  "UnionData",
				Union: true,
				Fields: []*parser.Field{
					{Name: "int_value", Number: 1, Type: parser.TypeUint32},
					{Name: "float_value", Number: 2, Type: parser.TypeFloat},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewArduinoGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("Arduino generation failed: %v", err)
	}

	// Check that union is generated
	if !strings.Contains(output, "typedef union") {
		t.Error("Expected typedef union for union message")
	}
}

// TestGoGenerator_NestedMessageDecoder tests Go decoder for nested messages
func TestGoGenerator_NestedMessageDecoder(t *testing.T) {
	innerMsg := &parser.Message{
		Name: "Inner",
		Fields: []*parser.Field{
			{Name: "value", Number: 1, Type: parser.TypeUint32},
		},
	}

	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			innerMsg,
			{
				Name:      "Outer",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "nested", Number: 2, Type: parser.TypeMessage, MessageType: innerMsg},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewGoGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("Go generation failed: %v", err)
	}

	// Check that decoder handles nested messages
	if !strings.Contains(output, "Decode(") {
		t.Error("Expected Decode method")
	}
	if !strings.Contains(output, "Inner") {
		t.Error("Expected Inner type")
	}
}

// TestGoGenerator_BytesFieldCodec tests Go encoder/decoder for bytes fields
func TestGoGenerator_BytesFieldCodec(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name:      "BytesMessage",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "data", Number: 2, Type: parser.TypeBytes, ArraySize: 64},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewGoGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("Go generation failed: %v", err)
	}

	// Check that bytes field is handled
	if !strings.Contains(output, "[]byte") {
		t.Error("Expected []byte type for bytes field")
	}
}

// TestGoGenerator_UnionMessage tests Go encoder/decoder for union messages
func TestGoGenerator_UnionMessage(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name:      "UnionData",
				Union:     true,
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "int_value", Number: 1, Type: parser.TypeUint32},
					{Name: "float_value", Number: 2, Type: parser.TypeFloat},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewGoGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("Go generation failed: %v", err)
	}

	// Check that union message is generated
	if !strings.Contains(output, "UnionData") {
		t.Error("Expected UnionData type")
	}
	// Union fields should all be at same offset
	if !strings.Contains(output, "Decode") {
		t.Error("Expected Decode method")
	}
}

// TestJSONGenerator_NestedMessages tests JSON schema for nested messages
func TestJSONGenerator_NestedMessages(t *testing.T) {
	innerMsg := &parser.Message{
		Name: "Inner",
		Fields: []*parser.Field{
			{Name: "value", Number: 1, Type: parser.TypeUint32},
		},
	}

	schema := &parser.Schema{
		Messages: []*parser.Message{
			innerMsg,
			{
				Name: "Outer",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "nested", Number: 2, Type: parser.TypeMessage, MessageType: innerMsg},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewJSONGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("JSON generation failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Check that both messages are in the schema
	messages := result["messages"].(map[string]any)
	if _, exists := messages["Inner"]; !exists {
		t.Error("Expected Inner message in schema")
	}
	if _, exists := messages["Outer"]; !exists {
		t.Error("Expected Outer message in schema")
	}
}

// TestJSONGenerator_BytesField tests JSON schema for bytes fields
func TestJSONGenerator_BytesField(t *testing.T) {
	schema := &parser.Schema{
		Messages: []*parser.Message{
			{
				Name: "BytesMessage",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "data", Number: 2, Type: parser.TypeBytes, ArraySize: 64},
				},
			},
		},
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	gen := NewJSONGenerator()
	output, err := gen.Generate(schema, layoutAnalyzer.GetAllLayouts())
	if err != nil {
		t.Fatalf("JSON generation failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Check that bytes field has correct type
	messages := result["messages"].(map[string]any)
	bytesMsg := messages["BytesMessage"].(map[string]any)
	structure := bytesMsg["structure"].([]any)

	// Find the data field
	for _, f := range structure {
		field := f.(map[string]any)
		if field["name"] == "data" {
			if field["type"] != "bytes" {
				t.Errorf("Expected type 'bytes', got %v", field["type"])
			}
			if field["size"] != float64(64) {
				t.Errorf("Expected size 64, got %v", field["size"])
			}
		}
	}
}
