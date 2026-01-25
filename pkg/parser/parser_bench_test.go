package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkParser_ParseSimpleMessage(b *testing.B) {
	tmpDir := b.TempDir()
	protoFile := filepath.Join(tmpDir, "bench.proto")
	protoContent := `syntax = "proto3";

package bench;

message Simple {
  uint32 id = 1;
  float value = 2;
  bool active = 3;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	parser := NewParser()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(protoFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseComplexMessage(b *testing.B) {
	tmpDir := b.TempDir()
	protoFile := filepath.Join(tmpDir, "bench.proto")
	protoContent := `syntax = "proto3";

package bench;

message Complex {
  uint32 timestamp = 1;
  float temperature = 2;
  repeated float values = 3;
  Nested nested = 4;
  Status status = 5;
  repeated Sensor sensors = 6;
}

message Nested {
  int32 x = 1;
  int32 y = 2;
  int32 z = 3;
}

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
  ERROR = 3;
}

message Sensor {
  uint32 id = 1;
  float value = 2;
  bool enabled = 3;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	parser := NewParser()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(protoFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseNestedMessages(b *testing.B) {
	tmpDir := b.TempDir()
	protoFile := filepath.Join(tmpDir, "bench.proto")
	protoContent := `syntax = "proto3";

package bench;

message Level1 {
  Level2 nested = 1;
  uint32 id = 2;
}

message Level2 {
  Level3 nested = 1;
  float value = 2;
}

message Level3 {
  Level4 nested = 1;
  bool flag = 2;
}

message Level4 {
  int32 data = 1;
  double precision = 2;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	parser := NewParser()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(protoFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseEnums(b *testing.B) {
	tmpDir := b.TempDir()
	protoFile := filepath.Join(tmpDir, "bench.proto")
	protoContent := `syntax = "proto3";

package bench;

enum Status {
  UNKNOWN = 0;
  PENDING = 1;
  ACTIVE = 2;
  INACTIVE = 3;
  COMPLETED = 4;
  FAILED = 5;
  CANCELLED = 6;
  TIMEOUT = 7;
}

enum Priority {
  LOW = 0;
  MEDIUM = 1;
  HIGH = 2;
  CRITICAL = 3;
}

message Task {
  Status status = 1;
  Priority priority = 2;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	parser := NewParser()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(protoFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseLargeSchema(b *testing.B) {
	tmpDir := b.TempDir()
	protoFile := filepath.Join(tmpDir, "bench.proto")

	// Generate a large schema with many fields
	protoContent := `syntax = "proto3";

package bench;

message LargeMessage {
  uint32 field1 = 1;
  uint32 field2 = 2;
  uint32 field3 = 3;
  uint32 field4 = 4;
  uint32 field5 = 5;
  uint32 field6 = 6;
  uint32 field7 = 7;
  uint32 field8 = 8;
  uint32 field9 = 9;
  uint32 field10 = 10;
  float field11 = 11;
  float field12 = 12;
  float field13 = 13;
  float field14 = 14;
  float field15 = 15;
  float field16 = 16;
  float field17 = 17;
  float field18 = 18;
  float field19 = 19;
  float field20 = 20;
  bool field21 = 21;
  bool field22 = 22;
  bool field23 = 23;
  bool field24 = 24;
  bool field25 = 25;
  int64 field26 = 26;
  int64 field27 = 27;
  int64 field28 = 28;
  int64 field29 = 29;
  int64 field30 = 30;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	parser := NewParser()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(protoFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseOneof(b *testing.B) {
	tmpDir := b.TempDir()
	protoFile := filepath.Join(tmpDir, "bench.proto")
	protoContent := `syntax = "proto3";

package bench;

message WithOneof {
  uint32 id = 1;

  oneof value {
    int32 int_value = 2;
    float float_value = 3;
    bool bool_value = 4;
    int64 long_value = 5;
    double double_value = 6;
  }

  bool active = 7;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	parser := NewParser()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(protoFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseMultipleOneofs(b *testing.B) {
	tmpDir := b.TempDir()
	protoFile := filepath.Join(tmpDir, "bench.proto")
	protoContent := `syntax = "proto3";

package bench;

message Notification {
  uint32 id = 1;

  oneof payload {
    SystemNotification system = 2;
    UserNotification user = 3;
    ErrorNotification error = 4;
  }

  oneof metadata {
    int32 priority = 5;
    float urgency = 6;
  }

  uint32 timestamp = 7;
}

message SystemNotification {
  uint32 code = 1;
  float value = 2;
}

message UserNotification {
  uint32 user_id = 1;
  uint32 message_code = 2;
}

message ErrorNotification {
  uint32 error_code = 1;
  uint32 line_number = 2;
  float severity = 3;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	parser := NewParser()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(protoFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseWithUnion(b *testing.B) {
	tmpDir := b.TempDir()
	protoFile := filepath.Join(tmpDir, "bench.proto")

	// Write options file
	optionsFile := filepath.Join(tmpDir, "proto2fixed", "binary.proto")
	if err := os.MkdirAll(filepath.Dir(optionsFile), 0755); err != nil {
		b.Fatalf("Failed to create directory: %v", err)
	}

	optionsContent := `syntax = "proto3";
package proto2fixed;
import "google/protobuf/descriptor.proto";

extend google.protobuf.MessageOptions {
  bool union = 50000;
}
extend google.protobuf.FieldOptions {
  uint32 array_size = 50020;
  uint32 string_size = 50021;
}
`
	if err := os.WriteFile(optionsFile, []byte(optionsContent), 0644); err != nil {
		b.Fatalf("Failed to write options file: %v", err)
	}

	protoContent := `syntax = "proto3";

package bench;

import "proto2fixed/binary.proto";

message UnionMessage {
  option (proto2fixed.union) = true;
  int32 int_value = 1;
  float float_value = 2;
  bool bool_value = 3;
  int64 long_value = 4;
}

message WithArray {
  uint32 id = 1;
  repeated float values = 2 [(proto2fixed.array_size) = 10];
  string name = 3 [(proto2fixed.string_size) = 32];
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	parser := NewParser(tmpDir)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(protoFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseBytesArray(b *testing.B) {
	tmpDir := b.TempDir()
	protoFile := filepath.Join(tmpDir, "bench.proto")

	// Write options file
	optionsFile := filepath.Join(tmpDir, "proto2fixed", "binary.proto")
	if err := os.MkdirAll(filepath.Dir(optionsFile), 0755); err != nil {
		b.Fatalf("Failed to create directory: %v", err)
	}

	optionsContent := `syntax = "proto3";
package proto2fixed;
import "google/protobuf/descriptor.proto";

extend google.protobuf.FieldOptions {
  uint32 array_size = 50020;
  uint32 string_size = 50021;
}
`
	if err := os.WriteFile(optionsFile, []byte(optionsContent), 0644); err != nil {
		b.Fatalf("Failed to write options file: %v", err)
	}

	protoContent := `syntax = "proto3";

package bench;

import "proto2fixed/binary.proto";

message ImageData {
  uint32 width = 1;
  uint32 height = 2;
  bytes frame = 3 [(proto2fixed.array_size) = 1024];
  bytes thumbnail = 4 [(proto2fixed.array_size) = 256];
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		b.Fatalf("Failed to write proto file: %v", err)
	}

	parser := NewParser(tmpDir)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(protoFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseRealWorldAHC2(b *testing.B) {
	parser := NewParser("../..")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse("../../testdata/ahc2/commands.proto")
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParser_ParseRealWorldAHSR(b *testing.B) {
	parser := NewParser("../..")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse("../../testdata/ahsr/status.proto")
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}
