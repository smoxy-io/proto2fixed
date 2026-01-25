package dynamic

import (
	"encoding/json"
	"testing"

	"github.com/smoxy-io/proto2fixed/pkg/generator"
)

func createSimpleSchema() generator.JSONSchema {
	return generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Simple": {
				TotalSize:        9,
				MessageId:        1,
				MessageTotalSize: 13, // 4 (message ID) + 9 (body)
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
					{Name: "value", Type: "float", Offset: 4, Size: 4},
					{Name: "active", Type: "bool", Offset: 8, Size: 1},
				},
			},
		},
	}
}

func createComplexSchema() generator.JSONSchema {
	return generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Complex": {
				TotalSize:        117,
				MessageId:        2,
				MessageTotalSize: 121, // 4 (message ID) + 117 (body)
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
					{Name: "timestamp", Type: "uint32", Offset: 4, Size: 4},
					{Name: "name", Type: "string", Offset: 8, Size: 32},
					{Name: "description", Type: "string", Offset: 40, Size: 64},
					{Name: "temperature", Type: "float", Offset: 104, Size: 4},
					{Name: "pressure", Type: "double", Offset: 108, Size: 8},
					{Name: "active", Type: "bool", Offset: 116, Size: 1},
				},
			},
		},
	}
}

func createUnionSchema() generator.JSONSchema {
	return generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Union": {
				TotalSize:           9, // 1 (discriminator) + 8 (largest field)
				MessageId:           3,
				MessageTotalSize:    13, // 4 (message ID) + 9 (body)
				Union:               true,
				HasDiscriminator:    true,
				DiscriminatorOffset: 0,
				DiscriminatorSize:   1,
				Structure: []*generator.JSONField{
					{Name: "int_value", FieldNumber: 1, Type: "int32", Offset: 1, Size: 4},
					{Name: "float_value", FieldNumber: 2, Type: "float", Offset: 1, Size: 4},
					{Name: "long_value", FieldNumber: 3, Type: "int64", Offset: 1, Size: 8},
					{Name: "bool_value", FieldNumber: 4, Type: "bool", Offset: 1, Size: 1},
				},
			},
		},
	}
}

func createOneofSchema() generator.JSONSchema {
	return generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"WithOneof": {
				TotalSize:        14, // 4 (id) + 4 (timestamp) + 5 (oneof with disc) + 1 (active)
				MessageId:        4,
				MessageTotalSize: 18, // 4 (message ID) + 14 (body)
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
					{Name: "timestamp", Type: "uint32", Offset: 4, Size: 4},
					{Name: "active", Type: "bool", Offset: 13, Size: 1},
				},
				Oneofs: []*generator.JSONOneof{
					{
						Name:                "value",
						Offset:              8,
						Size:                5, // 1 (discriminator) + 4 (largest variant)
						DiscriminatorOffset: 8,
						DiscriminatorSize:   1,
						Variants: []*generator.JSONField{
							{Name: "int_value", FieldNumber: 10, Type: "int32", Offset: 9, Size: 4},
							{Name: "float_value", FieldNumber: 11, Type: "float", Offset: 9, Size: 4},
						},
					},
				},
			},
		},
	}
}

func createNestedSchema() generator.JSONSchema {
	return generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Nested": {
				TotalSize: 12,
				Structure: []*generator.JSONField{
					{Name: "x", Type: "int32", Offset: 0, Size: 4},
					{Name: "y", Type: "int32", Offset: 4, Size: 4},
					{Name: "z", Type: "int32", Offset: 8, Size: 4},
				},
			},
			"WithNested": {
				TotalSize:        20,
				MessageId:        5,
				MessageTotalSize: 24, // 4 (message ID) + 20 (body)
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
					{
						Name:   "nested",
						Type:   "message",
						Offset: 4,
						Size:   12,
						Structure: []*generator.JSONField{
							{Name: "x", Type: "int32", Offset: 0, Size: 4},
							{Name: "y", Type: "int32", Offset: 4, Size: 4},
							{Name: "z", Type: "int32", Offset: 8, Size: 4},
						},
					},
					{Name: "active", Type: "bool", Offset: 16, Size: 1},
				},
			},
		},
	}
}

func createLargeArraySchema() generator.JSONSchema {
	return generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"LargeArrays": {
				TotalSize:        1624,
				MessageId:        6,
				MessageTotalSize: 1628, // 4 (message ID) + 1624 (body)
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
					{Name: "frame", Type: "bytes", Offset: 4, Size: 1024},
					{Name: "floats", Type: "float", Offset: 1028, Size: 400}, // 100 floats
					{Name: "ints", Type: "uint32", Offset: 1428, Size: 200},  // 50 ints
				},
			},
		},
	}
}

// Benchmark codec creation
func BenchmarkNew_Simple(b *testing.B) {
	schema := createSimpleSchema()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New(schema)
		if err != nil {
			b.Fatalf("Failed to create codec: %v", err)
		}
	}
}

func BenchmarkNew_Complex(b *testing.B) {
	schema := createComplexSchema()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New(schema)
		if err != nil {
			b.Fatalf("Failed to create codec: %v", err)
		}
	}
}

func BenchmarkNew_Union(b *testing.B) {
	schema := createUnionSchema()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New(schema)
		if err != nil {
			b.Fatalf("Failed to create codec: %v", err)
		}
	}
}

func BenchmarkNew_WithOneof(b *testing.B) {
	schema := createOneofSchema()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New(schema)
		if err != nil {
			b.Fatalf("Failed to create codec: %v", err)
		}
	}
}

func BenchmarkNew_Nested(b *testing.B) {
	schema := createNestedSchema()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New(schema)
		if err != nil {
			b.Fatalf("Failed to create codec: %v", err)
		}
	}
}

// Benchmark encoding
func BenchmarkEncode_Simple(b *testing.B) {
	schema := createSimpleSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Simple": map[string]any{
			"id":     float64(42),
			"value":  float64(3.14),
			"active": true,
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
	}
}

func BenchmarkEncode_Complex(b *testing.B) {
	schema := createComplexSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Complex": map[string]any{
			"id":          float64(123),
			"timestamp":   float64(1234567890),
			"name":        "Test Device",
			"description": "This is a test device for benchmarking purposes",
			"temperature": float64(25.5),
			"pressure":    float64(1013.25),
			"active":      true,
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
	}
}

func BenchmarkEncode_Union(b *testing.B) {
	schema := createUnionSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Union": map[string]any{
			"long_value": float64(9876543210),
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
	}
}

func BenchmarkEncode_WithOneof(b *testing.B) {
	schema := createOneofSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"WithOneof": map[string]any{
			"id":        float64(42),
			"timestamp": float64(1234567890),
			"value": map[string]any{
				"float_value": float64(3.14),
			},
			"active": true,
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
	}
}

func BenchmarkEncode_Nested(b *testing.B) {
	schema := createNestedSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"WithNested": map[string]any{
			"id": float64(42),
			"nested": map[string]any{
				"x": float64(10),
				"y": float64(20),
				"z": float64(30),
			},
			"active": true,
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
	}
}

func BenchmarkEncode_LargeArrays(b *testing.B) {
	schema := createLargeArraySchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	// Create a 1024 byte string for the frame
	frame := make([]byte, 1024)
	for i := range frame {
		frame[i] = byte(i % 256)
	}

	input := map[string]any{
		"LargeArrays": map[string]any{
			"id":    float64(1),
			"frame": string(frame),
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
	}
}

// Benchmark decoding
func BenchmarkDecode_Simple(b *testing.B) {
	schema := createSimpleSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Simple": map[string]any{
			"id":     float64(42),
			"value":  float64(3.14),
			"active": true,
		},
	}
	inputJSON, _ := json.Marshal(input)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		b.Fatalf("Encode failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

func BenchmarkDecode_Complex(b *testing.B) {
	schema := createComplexSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Complex": map[string]any{
			"id":          float64(123),
			"timestamp":   float64(1234567890),
			"name":        "Test Device",
			"description": "This is a test device for benchmarking purposes",
			"temperature": float64(25.5),
			"pressure":    float64(1013.25),
			"active":      true,
		},
	}
	inputJSON, _ := json.Marshal(input)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		b.Fatalf("Encode failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

func BenchmarkDecode_Union(b *testing.B) {
	schema := createUnionSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Union": map[string]any{
			"long_value": float64(9876543210),
		},
	}
	inputJSON, _ := json.Marshal(input)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		b.Fatalf("Encode failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

func BenchmarkDecode_WithOneof(b *testing.B) {
	schema := createOneofSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"WithOneof": map[string]any{
			"id":        float64(42),
			"timestamp": float64(1234567890),
			"value": map[string]any{
				"float_value": float64(3.14),
			},
			"active": true,
		},
	}
	inputJSON, _ := json.Marshal(input)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		b.Fatalf("Encode failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

func BenchmarkDecode_Nested(b *testing.B) {
	schema := createNestedSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"WithNested": map[string]any{
			"id": float64(42),
			"nested": map[string]any{
				"x": float64(10),
				"y": float64(20),
				"z": float64(30),
			},
			"active": true,
		},
	}
	inputJSON, _ := json.Marshal(input)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		b.Fatalf("Encode failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

func BenchmarkDecode_LargeArrays(b *testing.B) {
	schema := createLargeArraySchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	frame := make([]byte, 1024)
	for i := range frame {
		frame[i] = byte(i % 256)
	}

	input := map[string]any{
		"LargeArrays": map[string]any{
			"id":    float64(1),
			"frame": string(frame),
		},
	}
	inputJSON, _ := json.Marshal(input)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		b.Fatalf("Encode failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

// Benchmark round-trip (encode + decode)
func BenchmarkRoundTrip_Simple(b *testing.B) {
	schema := createSimpleSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Simple": map[string]any{
			"id":     float64(42),
			"value":  float64(3.14),
			"active": true,
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binary, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
		_, err = codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

func BenchmarkRoundTrip_Complex(b *testing.B) {
	schema := createComplexSchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Complex": map[string]any{
			"id":          float64(123),
			"timestamp":   float64(1234567890),
			"name":        "Test Device",
			"description": "This is a test device for benchmarking purposes",
			"temperature": float64(25.5),
			"pressure":    float64(1013.25),
			"active":      true,
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binary, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
		_, err = codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

func BenchmarkRoundTrip_LargeArrays(b *testing.B) {
	schema := createLargeArraySchema()
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	frame := make([]byte, 1024)
	for i := range frame {
		frame[i] = byte(i % 256)
	}

	input := map[string]any{
		"LargeArrays": map[string]any{
			"id":    float64(1),
			"frame": string(frame),
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binary, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
		_, err = codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

// Benchmark different endianness
func BenchmarkEncode_BigEndian(b *testing.B) {
	schema := createSimpleSchema()
	schema.Endian = "big"
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Simple": map[string]any{
			"id":     float64(42),
			"value":  float64(3.14),
			"active": true,
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(inputJSON)
		if err != nil {
			b.Fatalf("Encode failed: %v", err)
		}
	}
}

func BenchmarkDecode_BigEndian(b *testing.B) {
	schema := createSimpleSchema()
	schema.Endian = "big"
	codec, err := New(schema)
	if err != nil {
		b.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Simple": map[string]any{
			"id":     float64(42),
			"value":  float64(3.14),
			"active": true,
		},
	}
	inputJSON, _ := json.Marshal(input)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		b.Fatalf("Encode failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(binary)
		if err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}
