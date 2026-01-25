package testdata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/smoxy-io/proto2fixed/pkg/analyzer"
	"github.com/smoxy-io/proto2fixed/pkg/codecs/dynamic"
	"github.com/smoxy-io/proto2fixed/pkg/generator"
	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

// TestAHC2_Commands_Parse tests parsing the ahc2 commands.proto file
func TestAHC2_Commands_Parse(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahc2/commands.proto")
	if err != nil {
		t.Fatalf("Failed to parse commands.proto: %v", err)
	}

	if schema == nil {
		t.Fatal("Schema is nil")
	}

	if schema.Package != "ahc2" {
		t.Errorf("Expected package 'ahc2', got '%s'", schema.Package)
	}

	// Verify messages exist
	expectedMessages := []string{"Command", "Parameters", "ServoCommand", "ConfigCommand", "Response"}
	messageMap := make(map[string]*parser.Message)
	for _, msg := range schema.Messages {
		messageMap[msg.Name] = msg
	}

	for _, name := range expectedMessages {
		if _, exists := messageMap[name]; !exists {
			t.Errorf("Expected message '%s' not found", name)
		}
	}

	// Verify Action enum exists
	if len(schema.Enums) == 0 {
		t.Fatal("Expected Action enum not found")
	}

	var actionEnum *parser.Enum
	for _, e := range schema.Enums {
		if e.Name == "Action" {
			actionEnum = e
			break
		}
	}

	if actionEnum == nil {
		t.Fatal("Action enum not found")
	}

	// Verify enum has custom size option
	// Note: enum_size option may not be fully parsed yet
	if actionEnum.Size == 0 {
		t.Error("Enum size should not be 0")
	}

	// Verify enum values
	expectedEnumValues := []string{"servo", "multiServo", "config"}
	for _, name := range expectedEnumValues {
		found := false
		for _, val := range actionEnum.Values {
			if val.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected enum value '%s' not found", name)
		}
	}
}

// TestAHC2_Commands_Analyze tests analyzing the ahc2 commands.proto schema
func TestAHC2_Commands_Analyze(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahc2/commands.proto")
	if err != nil {
		t.Fatalf("Failed to parse commands.proto: %v", err)
	}

	// Analyze layout
	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// Verify layouts were created
	layouts := layoutAnalyzer.GetAllLayouts()
	if len(layouts) == 0 {
		t.Fatal("No layouts created")
	}

	// Verify Command message layout
	commandLayout, exists := layouts["Command"]
	if !exists {
		t.Fatal("Command layout not found")
	}

	// Verify fields are properly laid out
	if len(commandLayout.Fields) != 3 {
		t.Errorf("Expected 3 fields in Command, got %d", len(commandLayout.Fields))
	}

	// Verify Parameters layout exists (union type)
	paramsLayout, exists := layouts["Parameters"]
	if !exists {
		t.Fatal("Parameters layout not found")
	}

	// Verify Parameters is a union
	if !paramsLayout.Message.Union {
		t.Error("Parameters message should be marked as union")
	}
}

// TestAHC2_Commands_GenerateArduino tests Arduino code generation
func TestAHC2_Commands_GenerateArduino(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahc2/commands.proto")
	if err != nil {
		t.Fatalf("Failed to parse commands.proto: %v", err)
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	gen := generator.NewArduinoGenerator()
	code, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Failed to generate Arduino code: %v", err)
	}

	if len(code) == 0 {
		t.Error("Generated Arduino code is empty")
	}

	// Verify version is present in header comment
	if !containsString(code, "Schema Version:") {
		t.Error("Expected 'Schema Version:' in generated Arduino code")
	}

	if !containsString(code, "Endianness:") {
		t.Error("Expected 'Endianness:' in generated Arduino code")
	}

	// Verify key structures are present
	expectedElements := []string{
		"Command",
		"ServoCommand",
		"ConfigCommand",
		"Response",
		"Action",
		"typedef enum",
		"typedef struct",
	}

	for _, elem := range expectedElements {
		if !containsString(code, elem) {
			t.Errorf("Expected '%s' in generated Arduino code", elem)
		}
	}
}

// TestAHC2_Commands_GenerateGo tests Go code generation
func TestAHC2_Commands_GenerateGo(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahc2/commands.proto")
	if err != nil {
		t.Fatalf("Failed to parse commands.proto: %v", err)
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	gen := generator.NewGoGenerator()
	code, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	if len(code) == 0 {
		t.Error("Generated Go code is empty")
	}

	// Verify schema metadata constants are present
	if !containsString(code, "const CommandsSchemaVersion") {
		t.Error("Expected 'const CommandsSchemaVersion' in generated Go code")
	}

	if !containsString(code, "const CommandsSchemaEndian") {
		t.Error("Expected 'const CommandsSchemaEndian' in generated Go code")
	}

	// Verify key structures are present
	expectedElements := []string{
		"package ahc2",
		"type CommandDecoder struct {",
		"type ResponseDecoder struct {",
		"type CommandEncoder struct {",
		"type ResponseEncoder struct {",
		"const CommandSize",
	}

	for _, elem := range expectedElements {
		if !containsString(code, elem) {
			t.Errorf("Expected '%s' in generated Go code", elem)
		}
	}
}

// TestAHC2_Commands_GenerateJSON tests JSON schema generation
func TestAHC2_Commands_GenerateJSON(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahc2/commands.proto")
	if err != nil {
		t.Fatalf("Failed to parse commands.proto: %v", err)
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	gen := generator.NewJSONGenerator()
	jsonData, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Failed to generate JSON schema: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Generated JSON schema is empty")
	}

	// Parse and verify JSON structure
	var jsonSchema generator.JSONSchema
	if err := json.Unmarshal([]byte(jsonData), &jsonSchema); err != nil {
		t.Fatalf("Failed to parse generated JSON: %v", err)
	}

	// Version should always be present
	if jsonSchema.Version == "" {
		t.Error("JSON schema version is empty")
	}

	if jsonSchema.Endian == "" {
		t.Error("JSON schema endian is empty")
	}

	// Verify messages are present
	if len(jsonSchema.Messages) == 0 {
		t.Fatal("No messages in JSON schema")
	}

	expectedMessages := []string{"Command", "ServoCommand", "ConfigCommand", "Response", "Parameters"}
	for _, name := range expectedMessages {
		if _, exists := jsonSchema.Messages[name]; !exists {
			t.Errorf("Expected message '%s' not found in JSON schema", name)
		}
	}

	// Verify Command message has correct structure
	cmd, exists := jsonSchema.Messages["Command"]
	if !exists {
		t.Fatal("Command message not found in JSON schema")
	}

	if cmd.TotalSize == 0 {
		t.Error("Command message TotalSize is 0")
	}

	// Verify Parameters message is a union
	params, exists := jsonSchema.Messages["Parameters"]
	if !exists {
		t.Fatal("Parameters message not found in JSON schema")
	}

	if !params.Union {
		t.Error("Parameters message should be marked as union")
	}
}

// TestAHC2_Commands_DynamicCodec tests dynamic codec encoding/decoding
func TestAHC2_Commands_DynamicCodec(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahc2/commands.proto")
	if err != nil {
		t.Fatalf("Failed to parse commands.proto: %v", err)
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	gen := generator.NewJSONGenerator()
	jsonData, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Failed to generate JSON schema: %v", err)
	}

	var jsonSchema generator.JSONSchema
	if err := json.Unmarshal([]byte(jsonData), &jsonSchema); err != nil {
		t.Fatalf("Failed to parse generated JSON: %v", err)
	}

	// Create dynamic codec
	codec, err := dynamic.New(jsonSchema)
	if err != nil {
		t.Fatalf("Failed to create dynamic codec: %v", err)
	}

	if codec == nil {
		t.Fatal("Codec is nil")
	}

	// Test encoding Response message (simpler than Command)
	responseInput := map[string]any{
		"Response": map[string]any{
			"id":     float64(42),
			"errMsg": "test error",
		},
	}

	inputJSON, _ := json.Marshal(responseInput)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		t.Fatalf("Failed to encode Response: %v", err)
	}

	if len(binary) == 0 {
		t.Error("Encoded binary is empty")
	}

	// Test decoding
	outputJSON, err := codec.Decode(binary)
	if err != nil {
		t.Fatalf("Failed to decode Response: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(outputJSON, &result); err != nil {
		t.Fatalf("Failed to parse decoded JSON: %v", err)
	}

	response, ok := result["Response"].(map[string]any)
	if !ok {
		t.Fatal("Expected Response in decoded result")
	}

	if response["id"].(float64) != 42 {
		t.Errorf("Expected id=42, got %v", response["id"])
	}
}

// TestAHC2_Commands_Integration tests the full pipeline
func TestAHC2_Commands_Integration(t *testing.T) {
	// Parse
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahc2/commands.proto")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Analyze
	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	// Generate all formats
	generators := map[string]generator.Generator{
		"arduino": generator.NewArduinoGenerator(),
		"go":      generator.NewGoGenerator(),
		"json":    generator.NewJSONGenerator(),
	}

	tempDir := t.TempDir()

	for name, gen := range generators {
		code, err := gen.Generate(schema, layouts)
		if err != nil {
			t.Errorf("Failed to generate %s code: %v", name, err)
			continue
		}

		if len(code) == 0 {
			t.Errorf("Generated %s code is empty", name)
			continue
		}

		// Write to temp file
		ext := ".txt"
		if name == "go" {
			ext = ".go"
		} else if name == "json" {
			ext = ".json"
		} else if name == "arduino" {
			ext = ".h"
		}

		filename := filepath.Join(tempDir, "commands_"+name+ext)
		if err := os.WriteFile(filename, []byte(code), 0644); err != nil {
			t.Errorf("Failed to write %s file: %v", name, err)
		}
	}
}

// TestAHSR_Status_Parse tests parsing the ahsr status.proto file
func TestAHSR_Status_Parse(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahsr/status.proto")
	if err != nil {
		t.Fatalf("Failed to parse status.proto: %v", err)
	}

	if schema == nil {
		t.Fatal("Schema is nil")
	}

	if schema.Package != "ahsr" {
		t.Errorf("Expected package 'ahsr', got '%s'", schema.Package)
	}

	// Verify messages exist
	expectedMessages := []string{
		"StatusReport",
		"StatusReportUART1",
		"Devices",
		"DevicesUART1",
		"StatusInterface",
		"LED",
		"Camera",
	}
	messageMap := make(map[string]*parser.Message)
	for _, msg := range schema.Messages {
		messageMap[msg.Name] = msg
	}

	for _, name := range expectedMessages {
		if _, exists := messageMap[name]; !exists {
			t.Errorf("Expected message '%s' not found", name)
		}
	}

	// Verify Camera message has bytes field
	cameraMsg := messageMap["Camera"]
	if cameraMsg == nil {
		t.Fatal("Camera message not found")
	}

	var frameField *parser.Field
	for _, field := range cameraMsg.Fields {
		if field.Name == "frame" {
			frameField = field
			break
		}
	}

	if frameField == nil {
		t.Fatal("frame field not found in Camera message")
	}

	if frameField.Type != parser.TypeBytes {
		t.Errorf("Expected frame field type bytes, got %v", frameField.Type)
	}

	if frameField.ArraySize != 1024 {
		t.Errorf("Expected frame array size 1024, got %d", frameField.ArraySize)
	}
}

// TestAHSR_Status_Analyze tests analyzing the ahsr status.proto schema
func TestAHSR_Status_Analyze(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahsr/status.proto")
	if err != nil {
		t.Fatalf("Failed to parse status.proto: %v", err)
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// Verify layouts were created
	layouts := layoutAnalyzer.GetAllLayouts()
	if len(layouts) == 0 {
		t.Fatal("No layouts created")
	}

	// Verify StatusReport message layout
	statusLayout, exists := layouts["StatusReport"]
	if !exists {
		t.Fatal("StatusReport layout not found")
	}

	// Verify nested message structure
	if len(statusLayout.Fields) != 2 {
		t.Errorf("Expected 2 fields in StatusReport, got %d", len(statusLayout.Fields))
	}
}

// TestAHSR_Status_GenerateArduino tests Arduino code generation
func TestAHSR_Status_GenerateArduino(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahsr/status.proto")
	if err != nil {
		t.Fatalf("Failed to parse status.proto: %v", err)
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	gen := generator.NewArduinoGenerator()
	code, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Failed to generate Arduino code: %v", err)
	}

	if len(code) == 0 {
		t.Error("Generated Arduino code is empty")
	}

	// Verify version is present in header comment
	if !containsString(code, "Schema Version:") {
		t.Error("Expected 'Schema Version:' in generated Arduino code")
	}

	if !containsString(code, "Endianness:") {
		t.Error("Expected 'Endianness:' in generated Arduino code")
	}

	// Verify key structures are present
	expectedElements := []string{
		"StatusReport",
		"Devices",
		"Camera",
		"LED",
		"uint8_t frame[1024]", // bytes array
		"typedef struct",
	}

	for _, elem := range expectedElements {
		if !containsString(code, elem) {
			t.Errorf("Expected '%s' in generated Arduino code", elem)
		}
	}
}

// TestAHSR_Status_GenerateGo tests Go code generation
func TestAHSR_Status_GenerateGo(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahsr/status.proto")
	if err != nil {
		t.Fatalf("Failed to parse status.proto: %v", err)
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	gen := generator.NewGoGenerator()
	code, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	if len(code) == 0 {
		t.Error("Generated Go code is empty")
	}

	// Verify schema metadata constants are present
	if !containsString(code, "const StatusSchemaVersion") {
		t.Error("Expected 'const StatusSchemaVersion' in generated Go code")
	}

	if !containsString(code, "const StatusSchemaEndian") {
		t.Error("Expected 'const StatusSchemaEndian' in generated Go code")
	}

	// Verify key structures are present
	expectedElements := []string{
		"package ahsr",
		"type StatusReportDecoder struct {",
		"type StatusReportEncoder struct {",
		"type StatusReportUART1Decoder struct {",
		"type StatusReportUART1Encoder struct {",
		"const StatusReportSize",
	}

	for _, elem := range expectedElements {
		if !containsString(code, elem) {
			t.Errorf("Expected '%s' in generated Go code", elem)
		}
	}
}

// TestAHSR_Status_GenerateJSON tests JSON schema generation
func TestAHSR_Status_GenerateJSON(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahsr/status.proto")
	if err != nil {
		t.Fatalf("Failed to parse status.proto: %v", err)
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	gen := generator.NewJSONGenerator()
	jsonData, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Failed to generate JSON schema: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Generated JSON schema is empty")
	}

	// Parse and verify JSON structure
	var jsonSchema generator.JSONSchema
	if err := json.Unmarshal([]byte(jsonData), &jsonSchema); err != nil {
		t.Fatalf("Failed to parse generated JSON: %v", err)
	}

	// Version should always be present
	if jsonSchema.Version == "" {
		t.Error("JSON schema version is empty")
	}

	// Verify messages are present
	expectedMessages := []string{
		"StatusReport",
		"StatusReportUART1",
		"Devices",
		"DevicesUART1",
		"Camera",
		"LED",
		"StatusInterface",
	}

	for _, name := range expectedMessages {
		if _, exists := jsonSchema.Messages[name]; !exists {
			t.Errorf("Expected message '%s' not found in JSON schema", name)
		}
	}

	// Verify Camera message has bytes field
	camera, exists := jsonSchema.Messages["Camera"]
	if !exists {
		t.Fatal("Camera message not found in JSON schema")
	}

	var frameField *generator.JSONField
	for _, field := range camera.Structure {
		if field.Name == "frame" {
			frameField = field
			break
		}
	}

	if frameField == nil {
		t.Fatal("frame field not found in Camera message")
	}

	if frameField.Type != "bytes" {
		t.Errorf("Expected frame field type bytes, got %s", frameField.Type)
	}

	if frameField.Size != 1024 {
		t.Errorf("Expected frame field size 1024, got %d", frameField.Size)
	}
}

// TestAHSR_Status_DynamicCodec tests dynamic codec encoding/decoding
func TestAHSR_Status_DynamicCodec(t *testing.T) {
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahsr/status.proto")
	if err != nil {
		t.Fatalf("Failed to parse status.proto: %v", err)
	}

	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	gen := generator.NewJSONGenerator()
	jsonData, err := gen.Generate(schema, layouts)
	if err != nil {
		t.Fatalf("Failed to generate JSON schema: %v", err)
	}

	var jsonSchema generator.JSONSchema
	if err := json.Unmarshal([]byte(jsonData), &jsonSchema); err != nil {
		t.Fatalf("Failed to parse generated JSON: %v", err)
	}

	// Create dynamic codec
	codec, err := dynamic.New(jsonSchema)
	if err != nil {
		t.Fatalf("Failed to create dynamic codec: %v", err)
	}

	if codec == nil {
		t.Fatal("Codec is nil")
	}

	// Test encoding LED message (simplest structure)
	ledInput := map[string]any{
		"LED": map[string]any{
			"on": true,
		},
	}

	inputJSON, _ := json.Marshal(ledInput)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		t.Fatalf("Failed to encode LED: %v", err)
	}

	if len(binary) == 0 {
		t.Error("Encoded binary is empty")
	}

	// Test decoding
	outputJSON, err := codec.Decode(binary)
	if err != nil {
		t.Fatalf("Failed to decode LED: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(outputJSON, &result); err != nil {
		t.Fatalf("Failed to parse decoded JSON: %v", err)
	}

	led, ok := result["LED"].(map[string]any)
	if !ok {
		t.Fatal("Expected LED in decoded result")
	}

	if led["on"].(bool) != true {
		t.Errorf("Expected on=true, got %v", led["on"])
	}
}

// TestAHSR_Status_Integration tests the full pipeline
func TestAHSR_Status_Integration(t *testing.T) {
	// Parse
	p := parser.NewParser(".")
	schema, err := p.Parse("testdata/ahsr/status.proto")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Analyze
	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	if err := layoutAnalyzer.Analyze(schema); err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	layouts := layoutAnalyzer.GetAllLayouts()

	// Generate all formats
	generators := map[string]generator.Generator{
		"arduino": generator.NewArduinoGenerator(),
		"go":      generator.NewGoGenerator(),
		"json":    generator.NewJSONGenerator(),
	}

	tempDir := t.TempDir()

	for name, gen := range generators {
		code, err := gen.Generate(schema, layouts)
		if err != nil {
			t.Errorf("Failed to generate %s code: %v", name, err)
			continue
		}

		if len(code) == 0 {
			t.Errorf("Generated %s code is empty", name)
			continue
		}

		// Write to temp file
		ext := ".txt"
		if name == "go" {
			ext = ".go"
		} else if name == "json" {
			ext = ".json"
		} else if name == "arduino" {
			ext = ".h"
		}

		filename := filepath.Join(tempDir, "status_"+name+ext)
		if err := os.WriteFile(filename, []byte(code), 0644); err != nil {
			t.Errorf("Failed to write %s file: %v", name, err)
		}
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstringHelper(s, substr)
}

func findSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
