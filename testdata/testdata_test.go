package proto2fixed_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smoxy-io/proto2fixed/pkg/analyzer"
	"github.com/smoxy-io/proto2fixed/pkg/generator"
	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

// TestData_AllProtoFiles tests that all .proto files in testdata can be parsed, validated, and generated
func TestData_AllProtoFiles(t *testing.T) {
	testdataDir := "testdata"

	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only test .proto files, skip invalid.proto, example directories, validation test files, and test-specific files
		if !strings.HasSuffix(path, ".proto") ||
			strings.Contains(path, "invalid.proto") ||
			strings.Contains(path, "testdata/ahc2") ||
			strings.Contains(path, "testdata/ahsr") ||
			strings.Contains(path, "testdata/generator") ||
			strings.Contains(path, "testdata/parser") ||
			strings.Contains(path, "testdata/validation") ||
			strings.Contains(path, "no_options.proto") ||
			strings.Contains(path, "file_options.proto") ||
			strings.Contains(path, "message_options.proto") ||
			strings.Contains(path, "field_options.proto") ||
			strings.Contains(path, "enum_options.proto") {
			return nil
		}

		t.Run(path, func(t *testing.T) {
			// Parse
			p := parser.NewParser(".", "proto2fixed")
			schema, err := p.Parse(path)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Validate
			v := analyzer.NewValidator()
			result, err := v.Validate(schema)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}
			if result.HasErrors() {
				for _, e := range result.Errors {
					t.Errorf("Validation error: %s", e.Error())
				}
			}

			// Get layouts
			layouts := v.GetAnalyzer().GetAllLayouts()

			// Generate JSON
			jsonGen := generator.NewJSONGenerator()
			jsonCode, err := jsonGen.Generate(schema, layouts)
			if err != nil {
				t.Errorf("JSON generation failed: %v", err)
			}
			if !strings.Contains(jsonCode, "fixed-binary") {
				t.Error("JSON output should contain 'fixed-binary'")
			}

			// Generate Arduino
			arduinoGen := generator.NewArduinoGenerator()
			arduinoCode, err := arduinoGen.Generate(schema, layouts)
			if err != nil {
				t.Errorf("Arduino generation failed: %v", err)
			}
			if !strings.Contains(arduinoCode, "#ifndef") {
				t.Error("Arduino output should contain header guards")
			}

			// Generate Go
			goGen := generator.NewGoGenerator()
			goCode, err := goGen.Generate(schema, layouts)
			if err != nil {
				t.Errorf("Go generation failed: %v", err)
			}
			// Check for package declaration - if go_package is set, it overrides the generator package
			expectedPkg := "test"
			if schema.GoPackage != "" {
				expectedPkg = schema.GoPackage
			}
			if !strings.Contains(goCode, "package "+expectedPkg) {
				t.Errorf("Go output should contain 'package %s'", expectedPkg)
			}
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk testdata directory: %v", err)
	}
}

// TestData_InvalidProto tests that invalid.proto correctly fails parsing
func TestData_InvalidProto(t *testing.T) {
	protoFile := filepath.Join("testdata", "parser", "invalid.proto")

	p := parser.NewParser()
	_, err := p.Parse(protoFile)
	if err == nil {
		t.Error("Expected error when parsing invalid proto")
	}
}

// TestData_CLI_AllFormats tests CLI generation for a representative proto file
func TestData_CLI_AllFormats(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-test")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/proto2fixed")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	protoFile := filepath.Join("testdata", "cli", "simple_generate.proto")

	// Test JSON generation
	t.Run("JSON", func(t *testing.T) {
		outputFile := filepath.Join(tmpDir, "test", "simple_generate.json")
		cmd := exec.Command(binaryPath, "--lang=json", "--output="+tmpDir, protoFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("JSON generation failed: %v\nOutput: %s", err, string(output))
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read output: %v", err)
		}
		if !strings.Contains(string(data), "fixed-binary") {
			t.Error("JSON output should contain 'fixed-binary'")
		}
	})

	// Test Arduino generation
	t.Run("Arduino", func(t *testing.T) {
		outputFile := filepath.Join(tmpDir, "test", "simple_generate.h")
		cmd := exec.Command(binaryPath, "--lang=arduino", "--output="+tmpDir, protoFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Arduino generation failed: %v\nOutput: %s", err, string(output))
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read output: %v", err)
		}
		if !strings.Contains(string(data), "typedef struct") {
			t.Error("Arduino output should contain struct definition")
		}
	})

	// Test Go generation
	t.Run("Go", func(t *testing.T) {
		outputFile := filepath.Join(tmpDir, "test", "simple_generate.fbpb.go")
		cmd := exec.Command(binaryPath, "--lang=go", "--output="+tmpDir, protoFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Go generation failed: %v\nOutput: %s", err, string(output))
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read output: %v", err)
		}
		if !strings.Contains(string(data), "package test") {
			t.Error("Go output should contain package declaration")
		}
	})

	// Test validation
	t.Run("Validate", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--validate", protoFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Validation failed: %v\nOutput: %s", err, string(output))
		}
		if !strings.Contains(string(output), "Schema validation passed") {
			t.Error("Expected validation success message")
		}
	})
}

// TestData_OneofFiles tests oneof-specific proto files
func TestData_OneofFiles(t *testing.T) {
	oneofFiles := []string{
		filepath.Join("testdata", "oneof", "simple_oneof.proto"),
		filepath.Join("testdata", "oneof", "complex_oneof.proto"),
	}

	for _, protoFile := range oneofFiles {
		t.Run(protoFile, func(t *testing.T) {
			// Parse
			p := parser.NewParser()
			schema, err := p.Parse(protoFile)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Check that oneofs were parsed
			hasOneof := false
			for _, msg := range schema.Messages {
				if len(msg.Oneofs) > 0 {
					hasOneof = true

					// Verify oneof fields are associated correctly
					for _, oneof := range msg.Oneofs {
						if len(oneof.Fields) == 0 {
							t.Errorf("Oneof %s has no fields", oneof.Name)
						}

						// Verify fields have correct OneofIndex
						for _, field := range oneof.Fields {
							if field.OneofIndex == -1 {
								t.Errorf("Field %s in oneof %s has invalid OneofIndex", field.Name, oneof.Name)
							}
						}
					}
				}
			}

			if !hasOneof {
				t.Error("Expected at least one message with oneof")
			}

			// Validate and generate
			v := analyzer.NewValidator()
			result, err := v.Validate(schema)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}
			if result.HasErrors() {
				for _, e := range result.Errors {
					t.Errorf("Validation error: %s", e.Error())
				}
			}

			layouts := v.GetAnalyzer().GetAllLayouts()

			// Check that oneof layouts were generated
			hasOneofLayout := false
			for _, layout := range layouts {
				if len(layout.Oneofs) > 0 {
					hasOneofLayout = true

					// Verify oneof layout properties
					for _, oneofLayout := range layout.Oneofs {
						if oneofLayout.Size == 0 {
							t.Errorf("Oneof %s has zero size", oneofLayout.Oneof.Name)
						}

						// All variant fields should have same offset (union behavior)
						if len(oneofLayout.Fields) > 1 {
							firstOffset := oneofLayout.Fields[0].Offset
							for _, fieldLayout := range oneofLayout.Fields[1:] {
								if fieldLayout.Offset != firstOffset {
									t.Errorf("Oneof %s: variant offsets should be same (got %d and %d)",
										oneofLayout.Oneof.Name, firstOffset, fieldLayout.Offset)
								}
							}
						}
					}
				}
			}

			if !hasOneofLayout {
				t.Error("Expected at least one message layout with oneof")
			}

			// Test all generators
			jsonGen := generator.NewJSONGenerator()
			jsonCode, err := jsonGen.Generate(schema, layouts)
			if err != nil {
				t.Errorf("JSON generation failed: %v", err)
			}
			if !strings.Contains(jsonCode, "oneofs") {
				t.Error("JSON output should contain 'oneofs' field")
			}

			arduinoGen := generator.NewArduinoGenerator()
			arduinoCode, err := arduinoGen.Generate(schema, layouts)
			if err != nil {
				t.Errorf("Arduino generation failed: %v", err)
			}
			if !strings.Contains(arduinoCode, "union") {
				t.Error("Arduino output should contain union for oneof")
			}

			goGen := generator.NewGoGenerator()
			goCode, err := goGen.Generate(schema, layouts)
			if err != nil {
				t.Errorf("Go generation failed: %v", err)
			}
			if !strings.Contains(goCode, "Oneof") {
				t.Error("Go output should contain Oneof comment")
			}
		})
	}
}

// TestData_ParserSpecific tests parser-specific test files
func TestData_ParserSpecific(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		check func(*testing.T, *parser.Schema)
	}{
		{
			name: "SimpleWithOptions",
			file: filepath.Join("testdata", "parser", "simple_with_options.proto"),
			check: func(t *testing.T, schema *parser.Schema) {
				if !schema.Fixed {
					t.Error("Expected Fixed option to be true")
				}
				if schema.Endian != "little" {
					t.Errorf("Expected endian 'little', got '%s'", schema.Endian)
				}
			},
		},
		{
			name: "Nested",
			file: filepath.Join("testdata", "parser", "nested.proto"),
			check: func(t *testing.T, schema *parser.Schema) {
				if len(schema.Messages) != 2 {
					t.Errorf("Expected 2 messages, got %d", len(schema.Messages))
				}

				var outer *parser.Message
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
				var nestedField *parser.Field
				for _, field := range outer.Fields {
					if field.Type == parser.TypeMessage {
						nestedField = field
						break
					}
				}

				if nestedField == nil {
					t.Fatal("Nested message field not found")
				}

				if nestedField.MessageType == nil {
					t.Error("MessageType should not be nil")
				}
				if nestedField.MessageType.Name != "Inner" {
					t.Errorf("Expected nested message 'Inner', got '%s'", nestedField.MessageType.Name)
				}
			},
		},
		{
			name: "Enum",
			file: filepath.Join("testdata", "parser", "enum.proto"),
			check: func(t *testing.T, schema *parser.Schema) {
				if len(schema.Enums) != 1 {
					t.Fatalf("Expected 1 enum, got %d", len(schema.Enums))
				}

				enum := schema.Enums[0]
				if enum.Name != "Status" {
					t.Errorf("Expected enum 'Status', got '%s'", enum.Name)
				}

				if len(enum.Values) != 3 {
					t.Errorf("Expected 3 enum values, got %d", len(enum.Values))
				}

				// Check a message uses the enum
				hasEnumField := false
				for _, msg := range schema.Messages {
					for _, field := range msg.Fields {
						if field.Type == parser.TypeEnum && field.EnumType != nil {
							hasEnumField = true
							if field.EnumType.Name != "Status" {
								t.Errorf("Expected enum type 'Status', got '%s'", field.EnumType.Name)
							}
						}
					}
				}

				if !hasEnumField {
					t.Error("Expected at least one field using the enum")
				}
			},
		},
		{
			name: "AllTypes",
			file: filepath.Join("testdata", "parser", "all_types.proto"),
			check: func(t *testing.T, schema *parser.Schema) {
				if len(schema.Messages) != 1 {
					t.Fatalf("Expected 1 message, got %d", len(schema.Messages))
				}

				msg := schema.Messages[0]
				expectedTypes := []parser.FieldType{
					parser.TypeBool,
					parser.TypeInt32,
					parser.TypeUint32,
					parser.TypeInt64,
					parser.TypeUint64,
					parser.TypeFloat,
					parser.TypeDouble,
				}

				if len(msg.Fields) != len(expectedTypes) {
					t.Fatalf("Expected %d fields, got %d", len(expectedTypes), len(msg.Fields))
				}

				for i, field := range msg.Fields {
					if field.Type != expectedTypes[i] {
						t.Errorf("Field %d: expected type %v, got %v", i, expectedTypes[i], field.Type)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.NewParser(".", "proto2fixed")
			schema, err := p.Parse(tt.file)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			tt.check(t, schema)
		})
	}
}

// TestData_BytesFields tests end-to-end for messages with bytes fields
func TestData_BytesFields(t *testing.T) {
	// Create schema programmatically since option extraction doesn't work properly
	schema := &parser.Schema{
		FileName:  "test.proto",
		Fixed:     true,
		Endian:    "little",
		Package:   "testpkg",
		GoPackage: "testpkg",
		Messages: []*parser.Message{
			{
				Name:      "BytesMessage",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "data", Number: 2, Type: parser.TypeBytes, ArraySize: 64},
					{Name: "checksum", Number: 3, Type: parser.TypeUint32},
				},
			},
		},
	}

	// Validate
	v := analyzer.NewValidator()
	result, err := v.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("Validation errors: %v", result.Errors)
	}

	// Generate
	layouts := v.GetAnalyzer().GetAllLayouts()
	jsonGen := generator.NewJSONGenerator()
	jsonCode, err := jsonGen.Generate(schema, layouts)
	if err != nil {
		t.Errorf("JSON generation failed: %v", err)
	}
	if !strings.Contains(jsonCode, "bytes") {
		t.Error("JSON output should contain 'bytes' type")
	}

	arduinoGen := generator.NewArduinoGenerator()
	arduinoCode, err := arduinoGen.Generate(schema, layouts)
	if err != nil {
		t.Errorf("Arduino generation failed: %v", err)
	}
	if !strings.Contains(arduinoCode, "uint8_t") {
		t.Error("Arduino output should contain uint8_t for bytes")
	}

	goGen := generator.NewGoGenerator()
	goCode, err := goGen.Generate(schema, layouts)
	if err != nil {
		t.Errorf("Go generation failed: %v", err)
	}
	if !strings.Contains(goCode, "[]byte") {
		t.Error("Go output should contain []byte for bytes field")
	}
}

// TestData_DeeplyNestedMessages tests 3+ levels of message nesting
func TestData_DeeplyNestedMessages(t *testing.T) {
	protoFile := filepath.Join("testdata", "generator", "nested_messages.proto")

	p := parser.NewParser(".")
	schema, err := p.Parse(protoFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Validate
	v := analyzer.NewValidator()
	result, err := v.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("Validation errors: %v", result.Errors)
	}

	// Ensure nested message types are properly linked
	var outerMsg *parser.Message
	for _, msg := range schema.Messages {
		if msg.Name == "Outer" {
			outerMsg = msg
			break
		}
	}

	if outerMsg == nil {
		t.Fatal("Outer message not found")
	}

	// Check nested field has MessageType set
	hasNestedMessage := false
	for _, field := range outerMsg.Fields {
		if field.Type == parser.TypeMessage && field.MessageType != nil {
			hasNestedMessage = true
			if field.MessageType.Name != "Inner" {
				t.Errorf("Expected nested message 'Inner', got '%s'", field.MessageType.Name)
			}
		}
	}

	if !hasNestedMessage {
		t.Error("Expected at least one nested message field")
	}

	// Generate all formats
	layouts := v.GetAnalyzer().GetAllLayouts()
	for _, gen := range []struct {
		name string
		g    interface {
			Generate(*parser.Schema, map[string]*analyzer.MessageLayout) (string, error)
		}
	}{
		{"JSON", generator.NewJSONGenerator()},
		{"Arduino", generator.NewArduinoGenerator()},
		{"Go", generator.NewGoGenerator()},
	} {
		_, err := gen.g.Generate(schema, layouts)
		if err != nil {
			t.Errorf("%s generation failed: %v", gen.name, err)
		}
	}
}

// TestData_UnionMessages tests union message generation
func TestData_UnionMessages(t *testing.T) {
	// Create schema programmatically since option extraction doesn't work properly
	schema := &parser.Schema{
		FileName:  "test.proto",
		Fixed:     true,
		Endian:    "little",
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
					{Name: "long_value", Number: 3, Type: parser.TypeUint64},
				},
			},
		},
	}

	// Validate and generate
	v := analyzer.NewValidator()
	result, err := v.Validate(schema)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("Validation errors: %v", result.Errors)
	}

	layouts := v.GetAnalyzer().GetAllLayouts()

	// Arduino should generate typedef union
	arduinoGen := generator.NewArduinoGenerator()
	arduinoCode, err := arduinoGen.Generate(schema, layouts)
	if err != nil {
		t.Errorf("Arduino generation failed: %v", err)
	}
	if !strings.Contains(arduinoCode, "typedef union") {
		t.Error("Arduino output should contain 'typedef union'")
	}

	// Other generators should also handle union
	jsonGen := generator.NewJSONGenerator()
	_, err = jsonGen.Generate(schema, layouts)
	if err != nil {
		t.Errorf("JSON generation failed: %v", err)
	}

	goGen := generator.NewGoGenerator()
	_, err = goGen.Generate(schema, layouts)
	if err != nil {
		t.Errorf("Go generation failed: %v", err)
	}
}
