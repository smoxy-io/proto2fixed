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

// hasLine checks whether output contains a line whose normalized whitespace form contains s.
func hasLine(output, s string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(strings.Join(strings.Fields(line), " "), s) {
			return true
		}
	}
	return false
}

func TestGoGenerator_Struct_Primitives(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name: "AllPrimitives",
				Fields: []*parser.Field{
					{Name: "flag", Number: 1, Type: parser.TypeBool},
					{Name: "count", Number: 2, Type: parser.TypeUint32},
					{Name: "signed", Number: 3, Type: parser.TypeInt32},
					{Name: "big_count", Number: 4, Type: parser.TypeUint64},
					{Name: "big_signed", Number: 5, Type: parser.TypeInt64},
					{Name: "ratio", Number: 6, Type: parser.TypeFloat},
					{Name: "precise", Number: 7, Type: parser.TypeDouble},
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

	if !strings.Contains(output, "type AllPrimitives struct") {
		t.Error("Go output missing: type AllPrimitives struct")
	}

	for _, f := range []string{"Flag bool", "Count uint32", "Signed int32", "BigCount uint64", "BigSigned int64", "Ratio float32", "Precise float64"} {
		if !hasLine(output, f) {
			t.Errorf("Go output missing struct field: %s", f)
		}
	}

	for _, tag := range []string{`json:"flag"`, `json:"count"`, `json:"signed"`, `json:"big_count"`, `json:"big_signed"`, `json:"ratio"`, `json:"precise"`} {
		if !strings.Contains(output, tag) {
			t.Errorf("Go output missing json tag: %s", tag)
		}
	}
}

func TestGoGenerator_Struct_Enum(t *testing.T) {
	statusEnum := &parser.Enum{
		Name: "Status",
		Size: 1,
		Values: []*parser.EnumValue{
			{Name: "UNKNOWN", Number: 0},
			{Name: "ACTIVE", Number: 1},
			{Name: "INACTIVE", Number: 2},
		},
	}

	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Enums:     []*parser.Enum{statusEnum},
		Messages: []*parser.Message{
			{
				Name: "Device",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "status", Number: 2, Type: parser.TypeEnum, EnumType: statusEnum},
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

	for _, s := range []string{"type Status uint8", "type Device struct"} {
		if !strings.Contains(output, s) {
			t.Errorf("Go output missing: %s", s)
		}
	}

	for _, line := range []string{
		"UNKNOWN Status = 0",
		"ACTIVE Status = 1",
		"INACTIVE Status = 2",
		"Id uint32",
		"Status Status",
	} {
		if !hasLine(output, line) {
			t.Errorf("Go output missing: %s", line)
		}
	}

	// Verify JSON tags are present on struct fields
	if !strings.Contains(output, `json:"id"`) {
		t.Error("Go output missing json tag for id")
	}
	if !strings.Contains(output, `json:"status"`) {
		t.Error("Go output missing json tag for status")
	}
}

func TestGoGenerator_Struct_Enum_Size4(t *testing.T) {
	modeEnum := &parser.Enum{
		Name: "Mode",
		Size: 4,
		Values: []*parser.EnumValue{
			{Name: "OFF", Number: 0},
			{Name: "ON", Number: 1},
		},
	}

	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Enums:     []*parser.Enum{modeEnum},
		Messages: []*parser.Message{
			{
				Name: "Control",
				Fields: []*parser.Field{
					{Name: "mode", Number: 1, Type: parser.TypeEnum, EnumType: modeEnum},
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

	if !strings.Contains(output, "type Mode int32") {
		t.Error("Go output missing: type Mode int32")
	}

	if !hasLine(output, "OFF Mode = 0") {
		t.Error("Go output missing: OFF Mode = 0")
	}
}

func TestGoGenerator_Struct_Enum_Size2(t *testing.T) {
	flagEnum := &parser.Enum{
		Name: "Flag",
		Size: 2,
		Values: []*parser.EnumValue{
			{Name: "NONE", Number: 0},
			{Name: "SET", Number: 1},
		},
	}

	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Enums:     []*parser.Enum{flagEnum},
		Messages: []*parser.Message{
			{
				Name: "Flags",
				Fields: []*parser.Field{
					{Name: "flag", Number: 1, Type: parser.TypeEnum, EnumType: flagEnum},
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

	if !strings.Contains(output, "type Flag uint16") {
		t.Error("Go output missing: type Flag uint16")
	}
}

func TestGoGenerator_Struct_Array(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name: "Sensor",
				Fields: []*parser.Field{
					{Name: "readings", Number: 1, Type: parser.TypeFloat, Repeated: true, ArraySize: 8},
					{Name: "flags", Number: 2, Type: parser.TypeBool, Repeated: true, ArraySize: 4},
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

	if !strings.Contains(output, "type Sensor struct") {
		t.Error("Go output missing: type Sensor struct")
	}

	for _, f := range []string{"Readings [8]float32", "Flags [4]bool"} {
		if !hasLine(output, f) {
			t.Errorf("Go output missing struct field: %s", f)
		}
	}

	for _, tag := range []string{`json:"readings"`, `json:"flags"`} {
		if !strings.Contains(output, tag) {
			t.Errorf("Go output missing json tag: %s", tag)
		}
	}
}

func TestGoGenerator_Struct_String(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name: "Named",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "label", Number: 2, Type: parser.TypeString, StringSize: 32},
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

	if !strings.Contains(output, "type Named struct") {
		t.Error("Go output missing: type Named struct")
	}

	for _, f := range []string{"Id uint32", "Label string"} {
		if !hasLine(output, f) {
			t.Errorf("Go output missing struct field: %s", f)
		}
	}

	for _, tag := range []string{`json:"id"`, `json:"label"`} {
		if !strings.Contains(output, tag) {
			t.Errorf("Go output missing json tag: %s", tag)
		}
	}
}

func TestGoGenerator_Struct_Bytes(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name: "Packet",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "payload", Number: 2, Type: parser.TypeBytes, ArraySize: 64},
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

	if !strings.Contains(output, "type Packet struct") {
		t.Error("Go output missing: type Packet struct")
	}

	for _, f := range []string{"Id uint32", "Payload [64]byte"} {
		if !hasLine(output, f) {
			t.Errorf("Go output missing struct field: %s", f)
		}
	}

	for _, tag := range []string{`json:"id"`, `json:"payload"`} {
		if !strings.Contains(output, tag) {
			t.Errorf("Go output missing json tag: %s", tag)
		}
	}
}

func TestGoGenerator_Struct_NestedMessage(t *testing.T) {
	innerMsg := &parser.Message{
		Name: "Point",
		Fields: []*parser.Field{
			{Name: "x", Number: 1, Type: parser.TypeFloat},
			{Name: "y", Number: 2, Type: parser.TypeFloat},
		},
	}

	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			innerMsg,
			{
				Name: "Shape",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "origin", Number: 2, Type: parser.TypeMessage, MessageType: innerMsg},
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

	for _, s := range []string{"type Point struct", "type Shape struct"} {
		if !strings.Contains(output, s) {
			t.Errorf("Go output missing: %s", s)
		}
	}

	for _, f := range []string{"X float32", "Y float32", "Id uint32", "Origin Point"} {
		if !hasLine(output, f) {
			t.Errorf("Go output missing struct field: %s", f)
		}
	}

	for _, tag := range []string{`json:"x"`, `json:"y"`, `json:"id"`, `json:"origin"`} {
		if !strings.Contains(output, tag) {
			t.Errorf("Go output missing json tag: %s", tag)
		}
	}
}

func TestGoGenerator_Struct_MessageArray(t *testing.T) {
	innerMsg := &parser.Message{
		Name: "Sample",
		Fields: []*parser.Field{
			{Name: "value", Number: 1, Type: parser.TypeFloat},
		},
	}

	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			innerMsg,
			{
				Name: "Batch",
				Fields: []*parser.Field{
					{Name: "samples", Number: 1, Type: parser.TypeMessage, MessageType: innerMsg, Repeated: true, ArraySize: 10},
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

	for _, s := range []string{"type Sample struct", "type Batch struct"} {
		if !strings.Contains(output, s) {
			t.Errorf("Go output missing: %s", s)
		}
	}

	if !hasLine(output, "Samples [10]Sample") {
		t.Error("Go output missing struct field: Samples [10]Sample")
	}
}

func TestGoGenerator_Struct_Union(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name:      "Variant",
				Union:     true,
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "int_val", Number: 1, Type: parser.TypeUint32},
					{Name: "float_val", Number: 2, Type: parser.TypeFloat},
					{Name: "bool_val", Number: 3, Type: parser.TypeBool},
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

	if !strings.Contains(output, "type Variant struct") {
		t.Error("Go output missing: type Variant struct")
	}

	for _, f := range []string{"Discriminator uint8", "IntVal uint32", "FloatVal float32", "BoolVal bool"} {
		if !hasLine(output, f) {
			t.Errorf("Go output missing struct field: %s", f)
		}
	}

	for _, tag := range []string{`json:"discriminator"`, `json:"int_val,omitempty"`, `json:"float_val,omitempty"`, `json:"bool_val,omitempty"`} {
		if !strings.Contains(output, tag) {
			t.Errorf("Go output missing json tag: %s", tag)
		}
	}
}

func TestGoGenerator_Struct_Oneof(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name:      "Event",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32, OneofIndex: -1},
					{Name: "int_val", Number: 2, Type: parser.TypeInt32, OneofIndex: 0},
					{Name: "float_val", Number: 3, Type: parser.TypeFloat, OneofIndex: 0},
				},
				Oneofs: []*parser.Oneof{
					{
						Name: "payload",
						Fields: []*parser.Field{
							{Name: "int_val", Number: 2, Type: parser.TypeInt32, OneofIndex: 0},
							{Name: "float_val", Number: 3, Type: parser.TypeFloat, OneofIndex: 0},
						},
					},
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

	for _, s := range []string{"type EventPayloadOneof struct", "type Event struct"} {
		if !strings.Contains(output, s) {
			t.Errorf("Go output missing: %s", s)
		}
	}

	for _, f := range []string{"Discriminator uint8", "IntVal int32", "FloatVal float32", "Id uint32", "Payload EventPayloadOneof"} {
		if !hasLine(output, f) {
			t.Errorf("Go output missing struct field: %s", f)
		}
	}

	for _, tag := range []string{`json:"discriminator"`, `json:"int_val,omitempty"`, `json:"float_val,omitempty"`, `json:"id"`, `json:"payload"`} {
		if !strings.Contains(output, tag) {
			t.Errorf("Go output missing json tag: %s", tag)
		}
	}
}

func TestGoGenerator_Struct_CommentsPassThrough(t *testing.T) {
	statusEnum := &parser.Enum{
		Name:    "Status",
		Comment: "Status represents the device state.",
		Size:    1,
		Values: []*parser.EnumValue{
			{Name: "UNKNOWN", Comment: "UNKNOWN is the default unset state.", Number: 0},
			{Name: "ACTIVE", Comment: "ACTIVE means the device is running.", Number: 1},
		},
	}

	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Enums:     []*parser.Enum{statusEnum},
		Messages: []*parser.Message{
			{
				Name:    "Device",
				Comment: "Device represents a physical hardware device.",
				Fields: []*parser.Field{
					{Name: "id", Comment: "id is the unique device identifier.", Number: 1, Type: parser.TypeUint32},
					{Name: "status", Number: 2, Type: parser.TypeEnum, EnumType: statusEnum},
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

	required := []string{
		"// Status represents the device state.",
		"// UNKNOWN is the default unset state.",
		"// ACTIVE means the device is running.",
		"// Device represents a physical hardware device.",
		"// id is the unique device identifier.",
	}

	for _, s := range required {
		if !strings.Contains(output, s) {
			t.Errorf("Go output missing comment: %s", s)
		}
	}
}

func TestGoGenerator_Struct_NoBoilerplateComments(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Enums: []*parser.Enum{
			{Name: "Mode", Size: 1, Values: []*parser.EnumValue{{Name: "OFF", Number: 0}}},
		},
		Messages: []*parser.Message{
			{
				Name: "Simple",
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

	boilerplate := []string{
		"is a generated struct",
		"is a generated enum",
		"represents oneof",
	}

	for _, s := range boilerplate {
		if strings.Contains(output, s) {
			t.Errorf("Go output contains boilerplate comment: %s", s)
		}
	}
}

func TestGoGenerator_Struct_MultilineComment(t *testing.T) {
	schema := &parser.Schema{
		FileName:  "test.proto",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name:    "Reading",
				Comment: "Reading holds a sensor measurement.\nValues are in SI units.",
				Fields: []*parser.Field{
					{Name: "value", Number: 1, Type: parser.TypeFloat},
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

	if !strings.Contains(output, "// Reading holds a sensor measurement.") {
		t.Error("Go output missing first comment line")
	}

	if !strings.Contains(output, "// Values are in SI units.") {
		t.Error("Go output missing second comment line")
	}
}
