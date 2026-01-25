package analyzer

import (
	"testing"

	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

func BenchmarkLayoutAnalyzer_SimpleMessage(b *testing.B) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "Simple",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "value", Number: 2, Type: parser.TypeFloat},
					{Name: "active", Number: 3, Type: parser.TypeBool},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_UnionMessage(b *testing.B) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name:  "Union",
				Union: true,
				Fields: []*parser.Field{
					{Name: "int_value", Number: 1, Type: parser.TypeInt32},
					{Name: "float_value", Number: 2, Type: parser.TypeFloat},
					{Name: "bool_value", Number: 3, Type: parser.TypeBool},
					{Name: "long_value", Number: 4, Type: parser.TypeInt64},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_LargeArrays(b *testing.B) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "LargeArrays",
				Fields: []*parser.Field{
					{Name: "floats", Number: 1, Type: parser.TypeFloat, Repeated: true, ArraySize: 100},
					{Name: "ints", Number: 2, Type: parser.TypeUint32, Repeated: true, ArraySize: 50},
					{Name: "bytes", Number: 3, Type: parser.TypeBool, Repeated: true, ArraySize: 200},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_ManyFields(b *testing.B) {
	fields := make([]*parser.Field, 50)
	for i := 0; i < 50; i++ {
		fields[i] = &parser.Field{
			Name:   "field",
			Number: int32(i + 1),
			Type:   parser.TypeUint32,
		}
	}

	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name:   "ManyFields",
				Fields: fields,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_AlignmentCalculation(b *testing.B) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "Alignment",
				Fields: []*parser.Field{
					{Name: "b1", Number: 1, Type: parser.TypeBool},
					{Name: "i1", Number: 2, Type: parser.TypeUint32},
					{Name: "b2", Number: 3, Type: parser.TypeBool},
					{Name: "d1", Number: 4, Type: parser.TypeDouble},
					{Name: "b3", Number: 5, Type: parser.TypeBool},
					{Name: "i2", Number: 6, Type: parser.TypeUint32},
					{Name: "b4", Number: 7, Type: parser.TypeBool},
					{Name: "f1", Number: 8, Type: parser.TypeFloat},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkValidator_SimpleSchema(b *testing.B) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "Simple",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "value", Number: 2, Type: parser.TypeFloat},
					{Name: "active", Number: 3, Type: parser.TypeBool},
				},
			},
		},
	}

	validator := NewValidator()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := validator.Validate(schema)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

func BenchmarkValidator_StringAndArrayFields(b *testing.B) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "Data",
				Fields: []*parser.Field{
					{Name: "name", Number: 1, Type: parser.TypeString, StringSize: 64},
					{Name: "description", Number: 2, Type: parser.TypeString, StringSize: 128},
					{Name: "values", Number: 3, Type: parser.TypeFloat, Repeated: true, ArraySize: 20},
					{Name: "flags", Number: 4, Type: parser.TypeBool, Repeated: true, ArraySize: 16},
				},
			},
		},
	}

	validator := NewValidator()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := validator.Validate(schema)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

func BenchmarkValidator_ManyMessages(b *testing.B) {
	messages := make([]*parser.Message, 20)
	for i := 0; i < 20; i++ {
		messages[i] = &parser.Message{
			Name: "Message" + string(rune('A'+i)),
			Fields: []*parser.Field{
				{Name: "field1", Number: 1, Type: parser.TypeUint32},
				{Name: "field2", Number: 2, Type: parser.TypeFloat},
				{Name: "field3", Number: 3, Type: parser.TypeBool},
			},
		}
	}

	schema := &parser.Schema{
		Fixed:    true,
		Endian:   "little",
		Messages: messages,
	}

	validator := NewValidator()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := validator.Validate(schema)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_OneofMessage(b *testing.B) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "WithOneof",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32, OneofIndex: -1},
					{Name: "int_value", Number: 2, Type: parser.TypeInt32, OneofIndex: 0},
					{Name: "float_value", Number: 3, Type: parser.TypeFloat, OneofIndex: 0},
					{Name: "bool_value", Number: 4, Type: parser.TypeBool, OneofIndex: 0},
					{Name: "active", Number: 5, Type: parser.TypeBool, OneofIndex: -1},
				},
				Oneofs: []*parser.Oneof{
					{
						Name: "value",
						Fields: []*parser.Field{
							{Name: "int_value", Number: 2, Type: parser.TypeInt32, OneofIndex: 0},
							{Name: "float_value", Number: 3, Type: parser.TypeFloat, OneofIndex: 0},
							{Name: "bool_value", Number: 4, Type: parser.TypeBool, OneofIndex: 0},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_MultipleOneofs(b *testing.B) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "MultiOneof",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32, OneofIndex: -1},
					{Name: "int_val", Number: 2, Type: parser.TypeInt32, OneofIndex: 0},
					{Name: "float_val", Number: 3, Type: parser.TypeFloat, OneofIndex: 0},
					{Name: "priority", Number: 4, Type: parser.TypeInt32, OneofIndex: 1},
					{Name: "urgency", Number: 5, Type: parser.TypeFloat, OneofIndex: 1},
					{Name: "timestamp", Number: 6, Type: parser.TypeUint32, OneofIndex: -1},
				},
				Oneofs: []*parser.Oneof{
					{
						Name: "value",
						Fields: []*parser.Field{
							{Name: "int_val", Number: 2, Type: parser.TypeInt32, OneofIndex: 0},
							{Name: "float_val", Number: 3, Type: parser.TypeFloat, OneofIndex: 0},
						},
					},
					{
						Name: "metadata",
						Fields: []*parser.Field{
							{Name: "priority", Number: 4, Type: parser.TypeInt32, OneofIndex: 1},
							{Name: "urgency", Number: 5, Type: parser.TypeFloat, OneofIndex: 1},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_OneofWithNestedMessages(b *testing.B) {
	nestedMsg1 := &parser.Message{
		Name: "Nested1",
		Fields: []*parser.Field{
			{Name: "x", Number: 1, Type: parser.TypeInt32, OneofIndex: -1},
			{Name: "y", Number: 2, Type: parser.TypeInt32, OneofIndex: -1},
		},
	}

	nestedMsg2 := &parser.Message{
		Name: "Nested2",
		Fields: []*parser.Field{
			{Name: "a", Number: 1, Type: parser.TypeFloat, OneofIndex: -1},
			{Name: "b", Number: 2, Type: parser.TypeFloat, OneofIndex: -1},
		},
	}

	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			nestedMsg1,
			nestedMsg2,
			{
				Name: "WithNestedOneof",
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32, OneofIndex: -1},
					{Name: "nested1", Number: 2, Type: parser.TypeMessage, MessageType: nestedMsg1, OneofIndex: 0},
					{Name: "nested2", Number: 3, Type: parser.TypeMessage, MessageType: nestedMsg2, OneofIndex: 0},
					{Name: "active", Number: 4, Type: parser.TypeBool, OneofIndex: -1},
				},
				Oneofs: []*parser.Oneof{
					{
						Name: "payload",
						Fields: []*parser.Field{
							{Name: "nested1", Number: 2, Type: parser.TypeMessage, MessageType: nestedMsg1, OneofIndex: 0},
							{Name: "nested2", Number: 3, Type: parser.TypeMessage, MessageType: nestedMsg2, OneofIndex: 0},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_BytesArrays(b *testing.B) {
	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			{
				Name: "ImageData",
				Fields: []*parser.Field{
					{Name: "width", Number: 1, Type: parser.TypeUint32},
					{Name: "height", Number: 2, Type: parser.TypeUint32},
					{Name: "frame", Number: 3, Type: parser.TypeBytes, ArraySize: 1024},
					{Name: "thumbnail", Number: 4, Type: parser.TypeBytes, ArraySize: 256},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_UnionWithNestedMessages(b *testing.B) {
	nestedMsg := &parser.Message{
		Name: "Nested",
		Fields: []*parser.Field{
			{Name: "x", Number: 1, Type: parser.TypeInt32, OneofIndex: -1},
			{Name: "y", Number: 2, Type: parser.TypeInt32, OneofIndex: -1},
		},
	}

	schema := &parser.Schema{
		Fixed:  true,
		Endian: "little",
		Messages: []*parser.Message{
			nestedMsg,
			{
				Name:  "UnionWithNested",
				Union: true,
				Fields: []*parser.Field{
					{Name: "int_value", Number: 1, Type: parser.TypeInt32, OneofIndex: -1},
					{Name: "nested_value", Number: 2, Type: parser.TypeMessage, MessageType: nestedMsg, OneofIndex: -1},
					{Name: "float_value", Number: 3, Type: parser.TypeFloat, OneofIndex: -1},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_RealWorldAHC2(b *testing.B) {
	p := parser.NewParser("../..")
	schema, err := p.Parse("../../testdata/ahc2/commands.proto")
	if err != nil {
		b.Fatalf("Parse failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkLayoutAnalyzer_RealWorldAHSR(b *testing.B) {
	p := parser.NewParser("../..")
	schema, err := p.Parse("../../testdata/ahsr/status.proto")
	if err != nil {
		b.Fatalf("Parse failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewLayoutAnalyzer()
		err := analyzer.Analyze(schema)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkValidator_RealWorldAHC2(b *testing.B) {
	p := parser.NewParser("../..")
	schema, err := p.Parse("../../testdata/ahc2/commands.proto")
	if err != nil {
		b.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := validator.Validate(schema)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

func BenchmarkValidator_RealWorldAHSR(b *testing.B) {
	p := parser.NewParser("../..")
	schema, err := p.Parse("../../testdata/ahsr/status.proto")
	if err != nil {
		b.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := validator.Validate(schema)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}
