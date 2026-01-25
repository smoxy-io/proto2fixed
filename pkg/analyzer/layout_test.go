package analyzer

import (
	"testing"

	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

func TestLayoutAnalyzer_SimpleMessage(t *testing.T) {
	msg := &parser.Message{
		Name: "Simple",
		Fields: []*parser.Field{
			{Name: "value", Number: 1, Type: parser.TypeUint32},
			{Name: "temp", Number: 2, Type: parser.TypeFloat},
		},
	}

	analyzer := NewLayoutAnalyzer()
	layout, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// uint32 (4) + float (4) = 8 bytes
	if layout.TotalSize != 8 {
		t.Errorf("Expected total size 8, got %d", layout.TotalSize)
	}

	if len(layout.Fields) != 2 {
		t.Fatalf("Expected 2 field layouts, got %d", len(layout.Fields))
	}

	// Check field 1 offset
	if layout.Fields[0].Offset != 0 {
		t.Errorf("Expected field 1 offset 0, got %d", layout.Fields[0].Offset)
	}
	if layout.Fields[0].Size != 4 {
		t.Errorf("Expected field 1 size 4, got %d", layout.Fields[0].Size)
	}

	// Check field 2 offset
	if layout.Fields[1].Offset != 4 {
		t.Errorf("Expected field 2 offset 4, got %d", layout.Fields[1].Offset)
	}
	if layout.Fields[1].Size != 4 {
		t.Errorf("Expected field 2 size 4, got %d", layout.Fields[1].Size)
	}
}

func TestLayoutAnalyzer_FieldOrdering(t *testing.T) {
	// Fields should be ordered by field number, not declaration order
	msg := &parser.Message{
		Name: "Ordered",
		Fields: []*parser.Field{
			{Name: "field3", Number: 3, Type: parser.TypeUint32},
			{Name: "field1", Number: 1, Type: parser.TypeUint32},
			{Name: "field2", Number: 2, Type: parser.TypeUint32},
		},
	}

	analyzer := NewLayoutAnalyzer()
	layout, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// Check fields are in number order
	if layout.Fields[0].Field.Number != 1 {
		t.Errorf("First field should be number 1, got %d", layout.Fields[0].Field.Number)
	}
	if layout.Fields[1].Field.Number != 2 {
		t.Errorf("Second field should be number 2, got %d", layout.Fields[1].Field.Number)
	}
	if layout.Fields[2].Field.Number != 3 {
		t.Errorf("Third field should be number 3, got %d", layout.Fields[2].Field.Number)
	}

	// Check offsets
	if layout.Fields[0].Offset != 0 {
		t.Errorf("Field 1 offset should be 0, got %d", layout.Fields[0].Offset)
	}
	if layout.Fields[1].Offset != 4 {
		t.Errorf("Field 2 offset should be 4, got %d", layout.Fields[1].Offset)
	}
	if layout.Fields[2].Offset != 8 {
		t.Errorf("Field 3 offset should be 8, got %d", layout.Fields[2].Offset)
	}
}

func TestLayoutAnalyzer_Alignment(t *testing.T) {
	msg := &parser.Message{
		Name: "Aligned",
		Fields: []*parser.Field{
			{Name: "flag", Number: 1, Type: parser.TypeBool},    // 1 byte at offset 0
			{Name: "value", Number: 2, Type: parser.TypeUint32}, // 4 bytes, needs alignment
		},
	}

	analyzer := NewLayoutAnalyzer()
	layout, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// bool (1) + padding (3) + uint32 (4) = 8 bytes
	if layout.TotalSize != 8 {
		t.Errorf("Expected total size 8, got %d", layout.TotalSize)
	}

	// Check padding was inserted
	if len(layout.PaddingBytes) != 1 {
		t.Errorf("Expected 1 padding entry, got %d", len(layout.PaddingBytes))
	}

	// uint32 should be at offset 4 (aligned)
	if layout.Fields[1].Offset != 4 {
		t.Errorf("Expected uint32 at offset 4, got %d", layout.Fields[1].Offset)
	}
}

func TestLayoutAnalyzer_UnionMessage(t *testing.T) {
	msg := &parser.Message{
		Name:  "Union",
		Union: true,
		Fields: []*parser.Field{
			{Name: "small", Number: 1, Type: parser.TypeUint32},  // 4 bytes
			{Name: "large", Number: 2, Type: parser.TypeUint64},  // 8 bytes
			{Name: "tiny", Number: 3, Type: parser.TypeBool},     // 1 byte
		},
	}

	analyzer := NewLayoutAnalyzer()
	layout, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// Union size should be discriminator (1 byte) + largest field (8 bytes) = 9 bytes
	if layout.TotalSize != 9 {
		t.Errorf("Expected union size 9 (discriminator + largest field), got %d", layout.TotalSize)
	}

	// Verify discriminator
	if !layout.HasDiscriminator {
		t.Error("Union should have discriminator")
	}
	if layout.DiscriminatorOffset != 0 {
		t.Errorf("Discriminator should be at offset 0, got %d", layout.DiscriminatorOffset)
	}
	if layout.DiscriminatorSize != 1 {
		t.Errorf("Discriminator should be 1 byte, got %d", layout.DiscriminatorSize)
	}

	// All fields should start at offset 1 (after discriminator)
	for i, fieldLayout := range layout.Fields {
		if fieldLayout.Offset != 1 {
			t.Errorf("Union field %d should be at offset 1, got %d", i, fieldLayout.Offset)
		}
	}
}

func TestLayoutAnalyzer_NestedMessage(t *testing.T) {
	inner := &parser.Message{
		Name: "Inner",
		Fields: []*parser.Field{
			{Name: "value", Number: 1, Type: parser.TypeFloat}, // 4 bytes
		},
	}

	outer := &parser.Message{
		Name: "Outer",
		Fields: []*parser.Field{
			{Name: "id", Number: 1, Type: parser.TypeUint32},
			{Name: "nested", Number: 2, Type: parser.TypeMessage, MessageType: inner},
		},
	}

	analyzer := NewLayoutAnalyzer()

	// Analyze inner first
	_, err := analyzer.analyzeMessage(inner, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Inner analysis failed: %v", err)
	}

	// Analyze outer
	layout, err := analyzer.analyzeMessage(outer, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Outer analysis failed: %v", err)
	}

	// uint32 (4) + Inner (4) = 8 bytes
	if layout.TotalSize != 8 {
		t.Errorf("Expected total size 8, got %d", layout.TotalSize)
	}

	// Check nested field size
	nestedField := layout.Fields[1]
	if nestedField.Size != 4 {
		t.Errorf("Expected nested field size 4, got %d", nestedField.Size)
	}
}

func TestLayoutAnalyzer_ArrayField(t *testing.T) {
	msg := &parser.Message{
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
	}

	analyzer := NewLayoutAnalyzer()
	layout, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// float (4) * 10 = 40 bytes
	if layout.TotalSize != 40 {
		t.Errorf("Expected total size 40, got %d", layout.TotalSize)
	}

	if layout.Fields[0].Size != 40 {
		t.Errorf("Expected array field size 40, got %d", layout.Fields[0].Size)
	}
}

func TestLayoutAnalyzer_StringField(t *testing.T) {
	msg := &parser.Message{
		Name: "WithString",
		Fields: []*parser.Field{
			{
				Name:       "name",
				Number:     1,
				Type:       parser.TypeString,
				StringSize: 32,
			},
		},
	}

	analyzer := NewLayoutAnalyzer()
	layout, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// string size is fixed at 32 bytes
	if layout.TotalSize != 32 {
		t.Errorf("Expected total size 32, got %d", layout.TotalSize)
	}
}

func TestLayoutAnalyzer_EnumField(t *testing.T) {
	enum := &parser.Enum{
		Name: "Status",
		Size: 1, // 1-byte enum
	}

	msg := &parser.Message{
		Name: "WithEnum",
		Fields: []*parser.Field{
			{
				Name:     "status",
				Number:   1,
				Type:     parser.TypeEnum,
				EnumType: enum,
			},
		},
	}

	analyzer := NewLayoutAnalyzer()
	layout, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// 1-byte enum
	if layout.TotalSize != 1 {
		t.Errorf("Expected total size 1, got %d", layout.TotalSize)
	}
}

func TestLayoutAnalyzer_MissingArraySize(t *testing.T) {
	msg := &parser.Message{
		Name: "BadArray",
		Fields: []*parser.Field{
			{
				Name:      "values",
				Number:    1,
				Type:      parser.TypeFloat,
				Repeated:  true,
				ArraySize: 0, // Missing!
			},
		},
	}

	analyzer := NewLayoutAnalyzer()
	_, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err == nil {
		t.Error("Expected error for missing array size")
	}
}

func TestLayoutAnalyzer_MissingStringSize(t *testing.T) {
	msg := &parser.Message{
		Name: "BadString",
		Fields: []*parser.Field{
			{
				Name:       "name",
				Number:     1,
				Type:       parser.TypeString,
				StringSize: 0, // Missing!
			},
		},
	}

	analyzer := NewLayoutAnalyzer()
	_, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err == nil {
		t.Error("Expected error for missing string size")
	}
}

func TestLayoutAnalyzer_MessageAlignment(t *testing.T) {
	msg := &parser.Message{
		Name:  "AlignedMessage",
		Align: 8, // 8-byte alignment
		Fields: []*parser.Field{
			{Name: "value", Number: 1, Type: parser.TypeUint32}, // 4 bytes
		},
	}

	analyzer := NewLayoutAnalyzer()
	layout, err := analyzer.analyzeMessage(msg, &parser.Schema{MessageIdSize: 4})
	if err != nil {
		t.Fatalf("Layout analysis failed: %v", err)
	}

	// 4 bytes + 4 bytes padding to align to 8 bytes
	if layout.TotalSize != 8 {
		t.Errorf("Expected total size 8 (aligned), got %d", layout.TotalSize)
	}

	// Should have padding
	if len(layout.PaddingBytes) != 1 {
		t.Errorf("Expected 1 padding entry, got %d", len(layout.PaddingBytes))
	}
}
