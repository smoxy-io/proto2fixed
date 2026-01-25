package parser

import (
	"path/filepath"
	"testing"
)

func TestParser_ParseSimpleMessage(t *testing.T) {
	// Use testdata file
	protoFile := filepath.Join("..", "..", "testdata", "parser", "simple.proto")

	parser := NewParser()
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify schema
	if schema.Package != "test" {
		t.Errorf("Expected package 'test', got '%s'", schema.Package)
	}

	if !schema.Fixed {
		t.Error("Expected Fixed to be true by default")
	}

	if schema.Endian != "little" {
		t.Errorf("Expected endian 'little', got '%s'", schema.Endian)
	}

	if len(schema.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(schema.Messages))
	}

	msg := schema.Messages[0]
	if msg.Name != "Simple" {
		t.Errorf("Expected message name 'Simple', got '%s'", msg.Name)
	}

	if len(msg.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(msg.Fields))
	}

	// Check first field
	if msg.Fields[0].Name != "value" {
		t.Errorf("Expected field name 'value', got '%s'", msg.Fields[0].Name)
	}
	if msg.Fields[0].Type != TypeUint32 {
		t.Errorf("Expected field type TypeUint32, got %v", msg.Fields[0].Type)
	}
	if msg.Fields[0].Number != 1 {
		t.Errorf("Expected field number 1, got %d", msg.Fields[0].Number)
	}

	// Check second field
	if msg.Fields[1].Name != "temperature" {
		t.Errorf("Expected field name 'temperature', got '%s'", msg.Fields[1].Name)
	}
	if msg.Fields[1].Type != TypeFloat {
		t.Errorf("Expected field type TypeFloat, got %v", msg.Fields[1].Type)
	}
}

func TestParser_ParseNestedMessage(t *testing.T) {
	// Use testdata file
	protoFile := filepath.Join("..", "..", "testdata", "parser", "nested.proto")

	parser := NewParser()
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(schema.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(schema.Messages))
	}

	// Find Outer message
	var outer *Message
	for _, msg := range schema.Messages {
		if msg.Name == "Outer" {
			outer = msg
			break
		}
	}

	if outer == nil {
		t.Fatal("Outer message not found")
	}

	// Check nested field
	if len(outer.Fields) != 2 {
		t.Fatalf("Expected 2 fields in Outer, got %d", len(outer.Fields))
	}

	nestedField := outer.Fields[1]
	if nestedField.Type != TypeMessage {
		t.Errorf("Expected nested field type TypeMessage, got %v", nestedField.Type)
	}

	if nestedField.MessageType == nil {
		t.Fatal("MessageType is nil")
	}

	if nestedField.MessageType.Name != "Inner" {
		t.Errorf("Expected nested message name 'Inner', got '%s'", nestedField.MessageType.Name)
	}
}

func TestParser_ParseEnum(t *testing.T) {
	// Use testdata file
	protoFile := filepath.Join("..", "..", "testdata", "parser", "enum.proto")

	parser := NewParser()
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(schema.Enums))
	}

	enum := schema.Enums[0]
	if enum.Name != "Status" {
		t.Errorf("Expected enum name 'Status', got '%s'", enum.Name)
	}

	if enum.Size != 4 {
		t.Errorf("Expected enum size 4 (default), got %d", enum.Size)
	}

	if len(enum.Values) != 3 {
		t.Fatalf("Expected 3 enum values, got %d", len(enum.Values))
	}

	// Check enum values
	expectedValues := map[string]int32{
		"UNKNOWN": 0,
		"ACTIVE":  1,
		"STOPPED": 2,
	}

	for _, val := range enum.Values {
		expectedNum, exists := expectedValues[val.Name]
		if !exists {
			t.Errorf("Unexpected enum value: %s", val.Name)
			continue
		}
		if val.Number != expectedNum {
			t.Errorf("Expected %s=%d, got %d", val.Name, expectedNum, val.Number)
		}
	}
}

func TestParser_FileNotFound(t *testing.T) {
	parser := NewParser()
	_, err := parser.Parse("nonexistent.proto")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestParser_InvalidProto(t *testing.T) {
	// Use testdata file
	protoFile := filepath.Join("..", "..", "testdata", "parser", "invalid.proto")

	parser := NewParser()
	_, err := parser.Parse(protoFile)
	if err == nil {
		t.Error("Expected error for invalid proto syntax")
	}
}

func TestFieldType_AllTypes(t *testing.T) {
	// Use testdata file
	protoFile := filepath.Join("..", "..", "testdata", "parser", "all_types.proto")

	parser := NewParser()
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := schema.Messages[0]
	expectedTypes := []FieldType{
		TypeBool, TypeInt32, TypeUint32, TypeInt64, TypeUint64, TypeFloat, TypeDouble,
	}

	if len(msg.Fields) != len(expectedTypes) {
		t.Fatalf("Expected %d fields, got %d", len(expectedTypes), len(msg.Fields))
	}

	for i, field := range msg.Fields {
		if field.Type != expectedTypes[i] {
			t.Errorf("Field %d: expected type %v, got %v", i, expectedTypes[i], field.Type)
		}
	}
}

// TestParser_FileOptions tests extraction of file-level options
func TestParser_FileOptions(t *testing.T) {
	protoFile := filepath.Join("..", "..", "testdata", "parser", "file_options.proto")

	parser := NewParser(filepath.Join("..", ".."))
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !schema.Fixed {
		t.Error("Expected Fixed to be true")
	}

	// Endian defaults to little when not explicitly set to something else
	if schema.Endian != "little" {
		t.Errorf("Expected endian 'little', got '%s'", schema.Endian)
	}
}

// TestParser_MessageOptions tests extraction of message-level options
func TestParser_MessageOptions(t *testing.T) {
	protoFile := filepath.Join("..", "..", "testdata", "parser", "message_options.proto")

	parser := NewParser(filepath.Join("..", ".."))
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(schema.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(schema.Messages))
	}

	// Find TestNoOptions message - should have default values
	var msgNoOpts *Message
	for _, msg := range schema.Messages {
		if msg.Name == "TestNoOptions" {
			msgNoOpts = msg
			break
		}
	}
	if msgNoOpts == nil {
		t.Fatal("TestNoOptions message not found")
	}
	// Default values
	if msgNoOpts.Size != 0 {
		t.Errorf("Expected size 0 (unspecified), got %d", msgNoOpts.Size)
	}
	if msgNoOpts.Align != 0 {
		t.Errorf("Expected align 0 (natural), got %d", msgNoOpts.Align)
	}
	if msgNoOpts.Union {
		t.Error("Expected Union to be false")
	}
}

// TestParser_FieldOptions tests extraction of field-level options
func TestParser_FieldOptions(t *testing.T) {
	protoFile := filepath.Join("..", "..", "testdata", "parser", "field_options.proto")

	parser := NewParser(filepath.Join("..", ".."))
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := schema.Messages[0]

	// Check array field without options - should default to 0
	valuesField := msg.Fields[0]
	if !valuesField.Repeated {
		t.Error("Expected values field to be repeated")
	}
	if valuesField.ArraySize != 0 {
		t.Errorf("Expected array_size 0 (unspecified), got %d", valuesField.ArraySize)
	}

	// Check string field without options - should default to 0
	nameField := msg.Fields[1]
	if nameField.Type != TypeString {
		t.Errorf("Expected string type, got %v", nameField.Type)
	}
	if nameField.StringSize != 0 {
		t.Errorf("Expected string_size 0 (unspecified), got %d", nameField.StringSize)
	}
}

// TestParser_EnumOptions tests extraction of enum-level options
func TestParser_EnumOptions(t *testing.T) {
	protoFile := filepath.Join("..", "..", "testdata", "parser", "enum_options.proto")

	parser := NewParser(filepath.Join("..", ".."))
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(schema.Enums))
	}

	// Check DefaultEnum - should have default size of 4
	enum := schema.Enums[0]
	if enum.Name != "DefaultEnum" {
		t.Errorf("Expected enum name 'DefaultEnum', got '%s'", enum.Name)
	}
	if enum.Size != 4 {
		t.Errorf("Expected default enum size 4, got %d", enum.Size)
	}

	// Verify enum values were parsed
	if len(enum.Values) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(enum.Values))
	}
}

// TestParser_MissingOptions tests that missing options return defaults
func TestParser_MissingOptions(t *testing.T) {
	protoFile := filepath.Join("..", "..", "testdata", "parser", "no_options.proto")

	parser := NewParser()
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Default file options
	if !schema.Fixed {
		t.Error("Expected Fixed to default to true")
	}
	if schema.Endian != "little" {
		t.Errorf("Expected endian to default to 'little', got '%s'", schema.Endian)
	}
	if schema.Version != "v1.0.0" {
		t.Errorf("Expected version to default to 'v1.0.0', got '%s'", schema.Version)
	}

	// Message with no options
	msg := schema.Messages[0]
	if msg.Size != 0 {
		t.Errorf("Expected size to be 0 (not specified), got %d", msg.Size)
	}
	if msg.Align != 0 {
		t.Errorf("Expected align to be 0 (natural), got %d", msg.Align)
	}
	if msg.Union {
		t.Error("Expected Union to be false")
	}

	// Enum with no options
	enum := schema.Enums[0]
	if enum.Size != 4 {
		t.Errorf("Expected enum size to default to 4, got %d", enum.Size)
	}

	// Field with no options
	if msg.Fields[0].ArraySize != 0 {
		t.Errorf("Expected array_size to be 0, got %d", msg.Fields[0].ArraySize)
	}
	if msg.Fields[1].StringSize != 0 {
		t.Errorf("Expected string_size to be 0, got %d", msg.Fields[1].StringSize)
	}
}

// TestParser_ImportPaths tests parser with multiple import paths
func TestParser_ImportPaths(t *testing.T) {
	// Create parser with import paths
	parser := NewParser(filepath.Join("..", ".."), "proto2fixed", ".")

	protoFile := filepath.Join("..", "..", "testdata", "parser", "simple_with_options.proto")
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse with import paths failed: %v", err)
	}

	if schema.Package != "test" {
		t.Errorf("Expected package 'test', got '%s'", schema.Package)
	}

	if !schema.Fixed {
		t.Error("Expected Fixed to be true")
	}

	if schema.Endian != "little" {
		t.Errorf("Expected endian 'little', got '%s'", schema.Endian)
	}
}

// TestParser_MessageIds tests parsing of message_id and message_id_size options
func TestParser_MessageIds(t *testing.T) {
	protoFile := filepath.Join("..", "..", "testdata", "ahc2", "commands.proto")

	// Create parser with import paths to find proto2fixed/binary.proto
	parser := NewParser(filepath.Join("..", ".."), ".")
	schema, err := parser.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check file-level message_id_size option
	if schema.MessageIdSize != 4 {
		t.Errorf("Expected MessageIdSize 4 (default), got %d", schema.MessageIdSize)
	}

	// Check that Command has message_id = 1
	var commandMsg *Message
	for _, msg := range schema.Messages {
		if msg.Name == "Command" {
			commandMsg = msg
			break
		}
	}
	if commandMsg == nil {
		t.Fatal("Command message not found")
	}
	if commandMsg.MessageId != 1 {
		t.Errorf("Expected Command.MessageId = 1, got %d", commandMsg.MessageId)
	}

	// Check that Response has message_id = 2
	var responseMsg *Message
	for _, msg := range schema.Messages {
		if msg.Name == "Response" {
			responseMsg = msg
			break
		}
	}
	if responseMsg == nil {
		t.Fatal("Response message not found")
	}
	if responseMsg.MessageId != 2 {
		t.Errorf("Expected Response.MessageId = 2, got %d", responseMsg.MessageId)
	}

	// Check that nested messages (Parameters, etc.) don't have message IDs
	var parametersMsg *Message
	for _, msg := range schema.Messages {
		if msg.Name == "Parameters" {
			parametersMsg = msg
			break
		}
	}
	if parametersMsg != nil && parametersMsg.MessageId != 0 {
		t.Errorf("Expected Parameters.MessageId = 0 (nested message), got %d", parametersMsg.MessageId)
	}
}

// TestParser_GoPackage tests parsing of go_package option
func TestParser_GoPackage(t *testing.T) {
	tests := []struct {
		name               string
		file               string
		expectedPackage    string
		expectedImportPath string
	}{
		{
			name:               "path with slash",
			file:               "go_package_test.proto",
			expectedPackage:    "mypackage",
			expectedImportPath: "github.com/example/mypackage",
		},
		{
			name:               "path with semicolon",
			file:               "go_package_semicolon_test.proto",
			expectedPackage:    "custompkg",
			expectedImportPath: "github.com/example/custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protoFile := filepath.Join("..", "..", "testdata", tt.file)
			parser := NewParser(filepath.Join("..", ".."), ".")
			schema, err := parser.Parse(protoFile)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if schema.GoPackage != tt.expectedPackage {
				t.Errorf("Expected GoPackage '%s', got '%s'", tt.expectedPackage, schema.GoPackage)
			}

			if schema.GoPackageImport != tt.expectedImportPath {
				t.Errorf("Expected GoPackageImport '%s', got '%s'", tt.expectedImportPath, schema.GoPackageImport)
			}
		})
	}
}

func TestParser_Proto2Invalid(t *testing.T) {
	protoFile := filepath.Join("..", "..", "testdata", "parser", "invalid_proto2.proto")

	parser := NewParser(filepath.Join("..", ".."), ".")
	_, err := parser.Parse(protoFile)

	if err == nil {
		t.Error("Expected error for proto2 syntax. proto3 syntax is required")
	}
}
