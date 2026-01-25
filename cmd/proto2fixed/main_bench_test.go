package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const protoTemplate = `syntax = "proto3";

package test;

import "proto2fixed/binary.proto";

option (binary.fixed) = true;
option (binary.endian) = "little";
option (binary.message_id_size) = 4;
`

func BenchmarkCLI_BuildBinary(b *testing.B) {
	tmpDir := b.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("go", "build", "-o", binaryPath, ".")
		if err := cmd.Run(); err != nil {
			b.Fatalf("Failed to build binary: %v", err)
		}
	}
}

func BenchmarkCLI_Validate(b *testing.B) {
	tmpDir := b.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-bench")

	// Build binary once
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}

	// Create proto2fixed directory and copy binary.proto
	protoDir := filepath.Join(tmpDir, "proto2fixed")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		b.Fatalf("Failed to create proto2fixed directory: %v", err)
	}

	// Copy binary.proto from project root
	binaryProtoSrc := filepath.Join("..", "..", "proto2fixed", "binary.proto")
	binaryProtoDst := filepath.Join(protoDir, "binary.proto")
	srcData, err := os.ReadFile(binaryProtoSrc)
	if err != nil {
		b.Fatalf("Failed to read binary.proto: %v", err)
	}
	if err := os.WriteFile(binaryProtoDst, srcData, 0644); err != nil {
		b.Fatalf("Failed to write binary.proto: %v", err)
	}

	// Create test proto file
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";

package test;

import "proto2fixed/binary.proto";

option (binary.fixed) = true;
option (binary.endian) = "little";
option (binary.message_id_size) = 4;

message Simple {
  option (binary.message_id) = 1;
  uint32 value = 1;
  float temperature = 2;
  bool active = 3;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "--validate", protoFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			b.Fatalf("Validation failed: %v\nOutput: %s", err, string(output))
		}
	}
}

func BenchmarkCLI_GenerateJSON(b *testing.B) {
	tmpDir := b.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-bench")

	// Build binary once
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}

	// Create proto2fixed directory and copy binary.proto
	protoDir := filepath.Join(tmpDir, "proto2fixed")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		b.Fatalf("Failed to create proto2fixed directory: %v", err)
	}

	binaryProtoSrc := filepath.Join("..", "..", "proto2fixed", "binary.proto")
	binaryProtoDst := filepath.Join(protoDir, "binary.proto")
	srcData, err := os.ReadFile(binaryProtoSrc)
	if err != nil {
		b.Fatalf("Failed to read binary.proto: %v", err)
	}
	if err := os.WriteFile(binaryProtoDst, srcData, 0644); err != nil {
		b.Fatalf("Failed to write binary.proto: %v", err)
	}

	// Create test proto file
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";

package test;

import "proto2fixed/binary.proto";

option (binary.fixed) = true;
option (binary.endian) = "little";
option (binary.message_id_size) = 4;

message Simple {
  option (binary.message_id) = 1;
  uint32 value = 1;
  float temperature = 2;
  bool active = 3;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "--lang=json", "--output="+tmpDir, protoFile)
		if err := cmd.Run(); err != nil {
			b.Fatalf("JSON generation failed: %v", err)
		}
	}
}

func BenchmarkCLI_GenerateArduino(b *testing.B) {
	tmpDir := b.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-bench")

	// Build binary once
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}

	// Create proto2fixed directory and copy binary.proto
	protoDir := filepath.Join(tmpDir, "proto2fixed")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		b.Fatalf("Failed to create proto2fixed directory: %v", err)
	}

	binaryProtoSrc := filepath.Join("..", "..", "proto2fixed", "binary.proto")
	binaryProtoDst := filepath.Join(protoDir, "binary.proto")
	srcData, err := os.ReadFile(binaryProtoSrc)
	if err != nil {
		b.Fatalf("Failed to read binary.proto: %v", err)
	}
	if err := os.WriteFile(binaryProtoDst, srcData, 0644); err != nil {
		b.Fatalf("Failed to write binary.proto: %v", err)
	}

	// Create test proto file
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := protoTemplate + `
message Simple {
  option (binary.message_id) = 1;
  uint32 value = 1;
  float temperature = 2;
  bool active = 3;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "--lang=arduino", "--output="+tmpDir, protoFile)
		if err := cmd.Run(); err != nil {
			b.Fatalf("Arduino generation failed: %v", err)
		}
	}
}

func BenchmarkCLI_GenerateGo(b *testing.B) {
	tmpDir := b.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-bench")

	// Build binary once
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}

	// Create proto2fixed directory and copy binary.proto
	protoDir := filepath.Join(tmpDir, "proto2fixed")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		b.Fatalf("Failed to create proto2fixed directory: %v", err)
	}

	binaryProtoSrc := filepath.Join("..", "..", "proto2fixed", "binary.proto")
	binaryProtoDst := filepath.Join(protoDir, "binary.proto")
	srcData, err := os.ReadFile(binaryProtoSrc)
	if err != nil {
		b.Fatalf("Failed to read binary.proto: %v", err)
	}
	if err := os.WriteFile(binaryProtoDst, srcData, 0644); err != nil {
		b.Fatalf("Failed to write binary.proto: %v", err)
	}

	// Create test proto file
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := protoTemplate + `
message Simple {
  option (binary.message_id) = 1;
  uint32 value = 1;
  float temperature = 2;
  bool active = 3;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "--lang=go", "--output="+tmpDir, protoFile)
		if err := cmd.Run(); err != nil {
			b.Fatalf("Go generation failed: %v", err)
		}
	}
}

func BenchmarkCLI_ComplexSchema(b *testing.B) {
	tmpDir := b.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-bench")

	// Build binary once
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}

	// Create proto2fixed directory and copy binary.proto
	protoDir := filepath.Join(tmpDir, "proto2fixed")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		b.Fatalf("Failed to create proto2fixed directory: %v", err)
	}

	binaryProtoSrc := filepath.Join("..", "..", "proto2fixed", "binary.proto")
	binaryProtoDst := filepath.Join(protoDir, "binary.proto")
	srcData, err := os.ReadFile(binaryProtoSrc)
	if err != nil {
		b.Fatalf("Failed to read binary.proto: %v", err)
	}
	if err := os.WriteFile(binaryProtoDst, srcData, 0644); err != nil {
		b.Fatalf("Failed to write binary.proto: %v", err)
	}

	// Create complex proto file
	protoFile := filepath.Join(tmpDir, "complex.proto")
	protoContent := protoTemplate + `
message Complex {
  option (binary.message_id) = 1;
  uint32 timestamp = 1;
  float temperature = 2;
  Nested nested = 3;
  Status status = 4;
}

message Nested {
  int32 x = 1;
  int32 y = 2;
  int32 z = 3;
}

enum Status {
  option (binary.enum_size) = 1;
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
  ERROR = 3;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "--lang=json", "--output="+tmpDir, protoFile)
		if err := cmd.Run(); err != nil {
			b.Fatalf("Complex schema generation failed: %v", err)
		}
	}
}

func BenchmarkCLI_EndToEnd_AllFormats(b *testing.B) {
	tmpDir := b.TempDir()
	binaryPath := filepath.Join(tmpDir, "proto2fixed-bench")

	// Build binary once
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}

	// Create proto2fixed directory and copy binary.proto
	protoDir := filepath.Join(tmpDir, "proto2fixed")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		b.Fatalf("Failed to create proto2fixed directory: %v", err)
	}

	binaryProtoSrc := filepath.Join("..", "..", "proto2fixed", "binary.proto")
	binaryProtoDst := filepath.Join(protoDir, "binary.proto")
	srcData, err := os.ReadFile(binaryProtoSrc)
	if err != nil {
		b.Fatalf("Failed to read binary.proto: %v", err)
	}
	if err := os.WriteFile(binaryProtoDst, srcData, 0644); err != nil {
		b.Fatalf("Failed to write binary.proto: %v", err)
	}

	// Create test proto file
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := protoTemplate + `
message Data {
  option (binary.message_id) = 1;
  uint32 id = 1;
  float value = 2;
  bool active = 3;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate JSON
		cmd := exec.Command(binaryPath, "--lang=json", "--output="+tmpDir, protoFile)
		if err := cmd.Run(); err != nil {
			b.Fatalf("JSON generation failed: %v", err)
		}

		// Generate Arduino
		cmd = exec.Command(binaryPath, "--lang=arduino", "--output="+tmpDir, protoFile)
		if err := cmd.Run(); err != nil {
			b.Fatalf("Arduino generation failed: %v", err)
		}

		// Generate Go
		cmd = exec.Command(binaryPath, "--lang=go", "--output="+tmpDir, protoFile)
		if err := cmd.Run(); err != nil {
			b.Fatalf("Go generation failed: %v", err)
		}
	}
}
