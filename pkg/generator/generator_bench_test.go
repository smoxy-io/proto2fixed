package generator

import (
	"testing"

	"github.com/smoxy-io/proto2fixed/pkg/analyzer"
	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

func createSimpleSchemaAndLayouts() (*parser.Schema, map[string]*analyzer.MessageLayout) {
	schema := &parser.Schema{
		FileName:      "simple.proto",
		Fixed:         true,
		Endian:        "little",
		Package:       "bench",
		GoPackage:     "bench",
		MessageIdSize: 1,
		Messages: []*parser.Message{
			{
				Name:      "Simple",
				MessageId: 1,
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32},
					{Name: "value", Number: 2, Type: parser.TypeFloat},
					{Name: "active", Number: 3, Type: parser.TypeBool},
				},
			},
		},
	}

	// Run analyzer to get real layouts
	la := analyzer.NewLayoutAnalyzer()
	if err := la.Analyze(schema); err != nil {
		panic(err)
	}

	return schema, la.GetAllLayouts()
}

func createLargeSchemaAndLayouts() (*parser.Schema, map[string]*analyzer.MessageLayout) {
	messages := make([]*parser.Message, 10)

	// Create 10 messages with 10 fields each
	for i := 0; i < 10; i++ {
		fields := make([]*parser.Field, 10)

		for j := 0; j < 10; j++ {
			fields[j] = &parser.Field{
				Name:   "field" + string(rune('0'+j)),
				Number: int32(j + 1),
				Type:   parser.TypeUint32,
			}
		}

		messages[i] = &parser.Message{
			Name:      "Message" + string(rune('A'+i)),
			MessageId: uint32(i),
			Fields:    fields,
		}
	}

	schema := &parser.Schema{
		FileName:  "large.proto",
		Fixed:     true,
		Endian:    "little",
		Package:   "bench",
		GoPackage: "bench",
		Messages:  messages,
	}

	la := analyzer.NewLayoutAnalyzer()
	if err := la.Analyze(schema); err != nil {
		panic(err)
	}

	return schema, la.GetAllLayouts()
}

func BenchmarkJSONGenerator_SimpleSchema(b *testing.B) {
	schema, layouts := createSimpleSchemaAndLayouts()
	gen := NewJSONGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkJSONGenerator_LargeSchema(b *testing.B) {
	schema, layouts := createLargeSchemaAndLayouts()
	gen := NewJSONGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkArduinoGenerator_SimpleSchema(b *testing.B) {
	schema, layouts := createSimpleSchemaAndLayouts()
	gen := NewArduinoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkArduinoGenerator_LargeSchema(b *testing.B) {
	schema, layouts := createLargeSchemaAndLayouts()
	gen := NewArduinoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkGoGenerator_SimpleSchema(b *testing.B) {
	schema, layouts := createSimpleSchemaAndLayouts()
	gen := NewGoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkGoGenerator_LargeSchema(b *testing.B) {
	schema, layouts := createLargeSchemaAndLayouts()
	gen := NewGoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkGoGenerator_BigEndian(b *testing.B) {
	schema, layouts := createSimpleSchemaAndLayouts()
	schema.Endian = "big"
	gen := NewGoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkAllGenerators_SimpleSchema(b *testing.B) {
	schema, layouts := createSimpleSchemaAndLayouts()

	generators := []Generator{
		NewJSONGenerator(),
		NewArduinoGenerator(),
		NewGoGenerator(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, gen := range generators {
			_, err := gen.Generate(schema, layouts)
			if err != nil {
				b.Fatalf("Generation failed: %v", err)
			}
		}
	}
}

func BenchmarkAllGenerators_LargeSchema(b *testing.B) {
	schema, layouts := createLargeSchemaAndLayouts()

	generators := []Generator{
		NewJSONGenerator(),
		NewArduinoGenerator(),
		NewGoGenerator(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, gen := range generators {
			_, err := gen.Generate(schema, layouts)
			if err != nil {
				b.Fatalf("Generation failed: %v", err)
			}
		}
	}
}

func createOneofSchemaAndLayouts() (*parser.Schema, map[string]*analyzer.MessageLayout) {
	schema := &parser.Schema{
		FileName:  "oneof.proto",
		Fixed:     true,
		Endian:    "little",
		Package:   "bench",
		GoPackage: "bench",
		Messages: []*parser.Message{
			{
				Name:      "WithOneof",
				MessageId: 1,
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

	la := analyzer.NewLayoutAnalyzer()
	if err := la.Analyze(schema); err != nil {
		panic(err)
	}

	return schema, la.GetAllLayouts()
}

func BenchmarkJSONGenerator_Oneof(b *testing.B) {
	schema, layouts := createOneofSchemaAndLayouts()
	gen := NewJSONGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkArduinoGenerator_Oneof(b *testing.B) {
	schema, layouts := createOneofSchemaAndLayouts()
	gen := NewArduinoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkGoGenerator_Oneof(b *testing.B) {
	schema, layouts := createOneofSchemaAndLayouts()
	gen := NewGoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func createComplexOneofSchemaAndLayouts() (*parser.Schema, map[string]*analyzer.MessageLayout) {
	systemNotif := &parser.Message{
		Name:      "SystemNotification",
		MessageId: 1,
		Fields: []*parser.Field{
			{Name: "code", Number: 1, Type: parser.TypeUint32, OneofIndex: -1},
			{Name: "value", Number: 2, Type: parser.TypeFloat, OneofIndex: -1},
		},
	}

	userNotif := &parser.Message{
		Name:      "UserNotification",
		MessageId: 2,
		Fields: []*parser.Field{
			{Name: "user_id", Number: 1, Type: parser.TypeUint32, OneofIndex: -1},
			{Name: "message_code", Number: 2, Type: parser.TypeUint32, OneofIndex: -1},
		},
	}

	schema := &parser.Schema{
		FileName:  "complex_oneof.proto",
		Fixed:     true,
		Endian:    "little",
		Package:   "bench",
		GoPackage: "bench",
		Messages: []*parser.Message{
			systemNotif,
			userNotif,
			{
				Name:      "Notification",
				MessageId: 3,
				Fields: []*parser.Field{
					{Name: "id", Number: 1, Type: parser.TypeUint32, OneofIndex: -1},
					{Name: "system", Number: 2, Type: parser.TypeMessage, MessageType: systemNotif, OneofIndex: 0},
					{Name: "user", Number: 3, Type: parser.TypeMessage, MessageType: userNotif, OneofIndex: 0},
					{Name: "timestamp", Number: 4, Type: parser.TypeUint32, OneofIndex: -1},
				},
				Oneofs: []*parser.Oneof{
					{
						Name: "payload",
						Fields: []*parser.Field{
							{Name: "system", Number: 2, Type: parser.TypeMessage, MessageType: systemNotif, OneofIndex: 0},
							{Name: "user", Number: 3, Type: parser.TypeMessage, MessageType: userNotif, OneofIndex: 0},
						},
					},
				},
			},
		},
	}

	la := analyzer.NewLayoutAnalyzer()
	if err := la.Analyze(schema); err != nil {
		panic(err)
	}

	return schema, la.GetAllLayouts()
}

func BenchmarkJSONGenerator_ComplexOneof(b *testing.B) {
	schema, layouts := createComplexOneofSchemaAndLayouts()
	gen := NewJSONGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkArduinoGenerator_ComplexOneof(b *testing.B) {
	schema, layouts := createComplexOneofSchemaAndLayouts()
	gen := NewArduinoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkGoGenerator_ComplexOneof(b *testing.B) {
	schema, layouts := createComplexOneofSchemaAndLayouts()
	gen := NewGoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

// Benchmark with real-world schemas
func createRealWorldSchemaAndLayouts(protoFile string) (*parser.Schema, map[string]*analyzer.MessageLayout) {
	p := parser.NewParser("../..")
	schema, err := p.Parse(protoFile)
	if err != nil {
		panic(err)
	}

	la := analyzer.NewLayoutAnalyzer()
	if err := la.Analyze(schema); err != nil {
		panic(err)
	}

	return schema, la.GetAllLayouts()
}

func BenchmarkJSONGenerator_RealWorldAHC2(b *testing.B) {
	schema, layouts := createRealWorldSchemaAndLayouts("../../testdata/ahc2/commands.proto")
	gen := NewJSONGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkArduinoGenerator_RealWorldAHC2(b *testing.B) {
	schema, layouts := createRealWorldSchemaAndLayouts("../../testdata/ahc2/commands.proto")
	gen := NewArduinoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkGoGenerator_RealWorldAHC2(b *testing.B) {
	schema, layouts := createRealWorldSchemaAndLayouts("../../testdata/ahc2/commands.proto")
	gen := NewGoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkJSONGenerator_RealWorldAHSR(b *testing.B) {
	schema, layouts := createRealWorldSchemaAndLayouts("../../testdata/ahsr/status.proto")
	gen := NewJSONGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkArduinoGenerator_RealWorldAHSR(b *testing.B) {
	schema, layouts := createRealWorldSchemaAndLayouts("../../testdata/ahsr/status.proto")
	gen := NewArduinoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkGoGenerator_RealWorldAHSR(b *testing.B) {
	schema, layouts := createRealWorldSchemaAndLayouts("../../testdata/ahsr/status.proto")
	gen := NewGoGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(schema, layouts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

func BenchmarkAllGenerators_RealWorldAHC2(b *testing.B) {
	schema, layouts := createRealWorldSchemaAndLayouts("../../testdata/ahc2/commands.proto")

	generators := []Generator{
		NewJSONGenerator(),
		NewArduinoGenerator(),
		NewGoGenerator(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, gen := range generators {
			_, err := gen.Generate(schema, layouts)
			if err != nil {
				b.Fatalf("Generation failed: %v", err)
			}
		}
	}
}

func BenchmarkAllGenerators_RealWorldAHSR(b *testing.B) {
	schema, layouts := createRealWorldSchemaAndLayouts("../../testdata/ahsr/status.proto")

	generators := []Generator{
		NewJSONGenerator(),
		NewArduinoGenerator(),
		NewGoGenerator(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, gen := range generators {
			_, err := gen.Generate(schema, layouts)
			if err != nil {
				b.Fatalf("Generation failed: %v", err)
			}
		}
	}
}
