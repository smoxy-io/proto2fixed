package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_Version(t *testing.T) {
	// Build the binary first
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Run with --version
	cmd = exec.Command(binaryPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run --version: %v", err)
	}

	if !strings.Contains(string(output), "proto2fixed version") {
		t.Errorf("Expected version output, got: %s", string(output))
	}
}

func TestCLI_Help(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	cmd = exec.Command(binaryPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run --help: %v", err)
	}

	requiredStrings := []string{
		"proto2fixed",
		"Usage:",
		"--lang",
		"--output",
		"--validate",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(string(output), required) {
			t.Errorf("Help output missing: %s", required)
		}
	}
}

func TestCLI_Validate(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-test")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Use testdata file
	protoFile := filepath.Join("testdata", "cli", "simple_validate.proto")

	// Run validation from project root
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	cmd = exec.Command(binaryPath, "--validate", protoFile)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Validation failed: %v\nOutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "Schema validation passed") {
		t.Errorf("Expected validation success message, got: %s", string(output))
	}
}

func TestCLI_GenerateJSON(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-test")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Use testdata file
	protoFile := filepath.Join("testdata", "cli", "simple_generate.proto")

	// Run from project root
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Generate JSON
	outputFile := filepath.Join(tmpDir, "test", "simple_generate.json")
	cmd = exec.Command(binaryPath, "--lang=json", "--output="+tmpDir, protoFile)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("JSON generation failed: %v\nOutput: %s", err, string(output))
	}

	// Check output file exists
	if _, err := os.Stat(outputFile); err != nil {
		t.Fatalf("Output file not created: %v", err)
	}

	// Read and verify JSON
	jsonData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	if !strings.Contains(string(jsonData), "fixed-binary") {
		t.Errorf("Expected JSON to contain 'fixed-binary', got: %s", string(jsonData))
	}
}

func TestCLI_NoInput(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Run without input file
	cmd = exec.Command(binaryPath, "--lang=json")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error when no input file provided")
	}

	if !strings.Contains(string(output), "no input file") {
		t.Errorf("Expected 'no input file' error, got: %s", string(output))
	}
}

func TestCLI_InvalidLang(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Use testdata file
	protoFile := filepath.Join("testdata", "cli", "simple_validate.proto")

	// Run from project root
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Use invalid language
	cmd = exec.Command(binaryPath, "--lang=invalid", protoFile)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for invalid language")
	}

	if !strings.Contains(string(output), "invalid language") {
		t.Errorf("Expected 'invalid language' error, got: %s", string(output))
	}
}

func TestCLI_GenerateArduino(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Use testdata file
	protoFile := filepath.Join("testdata", "cli", "simple_validate.proto")

	// Run from project root
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "test", "simple_validate.h")
	cmd = exec.Command(binaryPath, "--lang=arduino", "--output="+tmpDir, protoFile)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Arduino generation failed: %v\nOutput: %s", err, string(output))
	}

	// Verify output
	headerData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	if !strings.Contains(string(headerData), "#ifndef") {
		t.Error("Arduino output should contain header guards")
	}
	if !strings.Contains(string(headerData), "typedef struct") {
		t.Error("Arduino output should contain struct definition")
	}
}

func TestCLI_GenerateGo(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Use testdata file
	protoFile := filepath.Join("testdata", "cli", "simple_validate.proto")

	// Run from project root
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "test", "simple_validate.fbpb.go")
	cmd = exec.Command(binaryPath, "--lang=go", "--output="+tmpDir, protoFile)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Go generation failed: %v\nOutput: %s", err, string(output))
	}

	// Verify output
	goData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	if !strings.Contains(string(goData), "package test") {
		t.Error("Go output should contain package declaration")
	}
	if !strings.Contains(string(goData), "Decoder struct {") {
		t.Error("Go output should contain Decoder type")
	}
	if !strings.Contains(string(goData), "Encoder struct {") {
		t.Error("Go output should contain Encoder type")
	}
}
