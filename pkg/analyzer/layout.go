package analyzer

import (
	"fmt"
	"sort"

	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

// FieldLayout represents the binary layout information for a field
type FieldLayout struct {
	Field  *parser.Field
	Offset uint32
	Size   uint32
}

// OneofLayout represents the binary layout for a oneof group
type OneofLayout struct {
	Oneof               *parser.Oneof
	Offset              uint32
	Size                uint32
	Fields              []*FieldLayout
	DiscriminatorOffset uint32 // Always at Offset
	DiscriminatorSize   uint32 // Always 1
}

// MessageLayout represents the binary layout for a message
type MessageLayout struct {
	Message             *parser.Message
	Fields              []*FieldLayout
	Oneofs              []*OneofLayout
	TotalSize           uint32
	PaddingBytes        []PaddingInfo
	HasMessageId        bool   // True if this message has a message ID
	MessageId           uint32 // The message ID value
	MessageIdSize       uint32 // The message ID encoding size (from schema)
	MessageTotalSize    uint32 // TotalSize + MessageIdSize (with ID header)
	HasDiscriminator    bool   // True if Union=true
	DiscriminatorOffset uint32 // Always 0 for unions
	DiscriminatorSize   uint32 // Always 1 for unions
}

// PaddingInfo represents padding inserted for alignment
type PaddingInfo struct {
	Offset uint32
	Size   uint32
	Reason string
}

// LayoutAnalyzer calculates binary layouts for messages
type LayoutAnalyzer struct {
	layouts map[string]*MessageLayout
	enums   map[string]*parser.Enum
	schema  *parser.Schema // Stored schema for accessing file-level options
}

// NewLayoutAnalyzer creates a new layout analyzer
func NewLayoutAnalyzer() *LayoutAnalyzer {
	return &LayoutAnalyzer{
		layouts: make(map[string]*MessageLayout),
		enums:   make(map[string]*parser.Enum),
	}
}

// Analyze calculates layouts for all messages in the schema
func (a *LayoutAnalyzer) Analyze(schema *parser.Schema) error {
	// Store schema for use by helper functions
	a.schema = schema

	// Build enum map
	for _, enum := range schema.Enums {
		a.enums[enum.Name] = enum
	}

	// Analyze each message
	for _, msg := range schema.Messages {
		if _, err := a.analyzeMessage(msg, schema); err != nil {
			return err
		}
	}

	return nil
}

// GetLayout returns the layout for a message by name
func (a *LayoutAnalyzer) GetLayout(messageName string) (*MessageLayout, bool) {
	layout, ok := a.layouts[messageName]
	return layout, ok
}

// GetAllLayouts returns all computed layouts
func (a *LayoutAnalyzer) GetAllLayouts() map[string]*MessageLayout {
	return a.layouts
}

// analyzeMessage calculates the binary layout for a message
func (a *LayoutAnalyzer) analyzeMessage(msg *parser.Message, schema *parser.Schema) (*MessageLayout, error) {
	// Check if already analyzed
	if layout, exists := a.layouts[msg.Name]; exists {
		return layout, nil
	}

	layout := &MessageLayout{
		Message:       msg,
		Fields:        make([]*FieldLayout, 0),
		Oneofs:        make([]*OneofLayout, 0),
		HasMessageId:  msg.MessageId > 0,
		MessageId:     msg.MessageId,
		MessageIdSize: schema.MessageIdSize,
	}

	// Separate fields into regular fields and oneof fields
	regularFields := make([]*parser.Field, 0)
	if len(msg.Oneofs) == 0 {
		// No oneofs, all fields are regular fields
		regularFields = msg.Fields
	} else {
		// Filter out fields that are part of oneofs
		for _, field := range msg.Fields {
			if field.OneofIndex == -1 {
				regularFields = append(regularFields, field)
			}
		}
	}

	// Sort regular fields by field number (ascending)
	sortedFields := make([]*parser.Field, len(regularFields))
	copy(sortedFields, regularFields)
	sort.Slice(sortedFields, func(i, j int) bool {
		return sortedFields[i].Number < sortedFields[j].Number
	})

	// Pre-calculate oneof layouts
	oneofLayouts := make([]*OneofLayout, len(msg.Oneofs))
	for i, oneof := range msg.Oneofs {
		oneofLayout := &OneofLayout{
			Oneof:               oneof,
			Fields:              make([]*FieldLayout, 0),
			DiscriminatorOffset: 0, // Relative to oneof start
			DiscriminatorSize:   1, // Always 1 byte
		}

		// All oneof fields overlay at offset 1 (after discriminator, within the oneof)
		// Size = discriminator (1 byte) + max of all variant sizes
		maxSize := uint32(0)
		for _, field := range oneof.Fields {
			fieldSize, err := a.calculateFieldSize(field)
			if err != nil {
				return nil, fmt.Errorf("oneof %s.%s field %s: %w", msg.Name, oneof.Name, field.Name, err)
			}

			oneofLayout.Fields = append(oneofLayout.Fields, &FieldLayout{
				Field:  field,
				Offset: 1, // Relative to oneof start, after discriminator
				Size:   fieldSize,
			})

			if fieldSize > maxSize {
				maxSize = fieldSize
			}
		}
		oneofLayout.Size = 1 + maxSize // Discriminator + max variant size
		oneofLayouts[i] = oneofLayout
	}

	if msg.Union {
		// Union message: discriminator at offset 0, all fields at offset 1
		// Size = discriminator (1 byte) + largest field
		layout.HasDiscriminator = true
		layout.DiscriminatorOffset = 0
		layout.DiscriminatorSize = 1

		maxSize := uint32(0)
		for _, field := range sortedFields {
			fieldSize, err := a.calculateFieldSize(field)
			if err != nil {
				return nil, fmt.Errorf("field %s.%s: %w", msg.Name, field.Name, err)
			}

			layout.Fields = append(layout.Fields, &FieldLayout{
				Field:  field,
				Offset: 1, // After discriminator
				Size:   fieldSize,
			})

			if fieldSize > maxSize {
				maxSize = fieldSize
			}
		}
		layout.TotalSize = 1 + maxSize // Discriminator + max field size
	} else {
		// Normal message: sequential layout with alignment
		// Need to interleave oneofs and regular fields based on field numbers
		currentOffset := uint32(0)

		fieldIdx := 0
		oneofIdx := 0

		for fieldIdx < len(sortedFields) || oneofIdx < len(oneofLayouts) {
			// Determine whether to place a regular field or oneof next
			placeField := true
			if fieldIdx >= len(sortedFields) {
				placeField = false
			} else if oneofIdx < len(oneofLayouts) {
				// Compare lowest field number in oneof with current field
				oneofMinNumber := getMinFieldNumber(msg.Oneofs[oneofIdx])
				if oneofMinNumber < sortedFields[fieldIdx].Number {
					placeField = false
				}
			}

			if placeField {
				// Place regular field
				field := sortedFields[fieldIdx]
				fieldSize, err := a.calculateFieldSize(field)
				if err != nil {
					return nil, fmt.Errorf("field %s.%s: %w", msg.Name, field.Name, err)
				}

				// Calculate alignment requirement
				alignment := a.getFieldAlignment(field)

				// Add padding if needed for alignment
				if alignment > 1 && currentOffset%alignment != 0 {
					paddingSize := alignment - (currentOffset % alignment)
					layout.PaddingBytes = append(layout.PaddingBytes, PaddingInfo{
						Offset: currentOffset,
						Size:   paddingSize,
						Reason: fmt.Sprintf("align field %d (%s) to %d-byte boundary", field.Number, field.Name, alignment),
					})
					currentOffset += paddingSize
				}

				layout.Fields = append(layout.Fields, &FieldLayout{
					Field:  field,
					Offset: currentOffset,
					Size:   fieldSize,
				})

				currentOffset += fieldSize
				fieldIdx++
			} else {
				// Place oneof
				oneofLayout := oneofLayouts[oneofIdx]

				// Calculate alignment for oneof (max alignment of all variants)
				alignment := a.getOneofAlignment(oneofLayout)

				// Add padding if needed for alignment
				if alignment > 1 && currentOffset%alignment != 0 {
					paddingSize := alignment - (currentOffset % alignment)
					layout.PaddingBytes = append(layout.PaddingBytes, PaddingInfo{
						Offset: currentOffset,
						Size:   paddingSize,
						Reason: fmt.Sprintf("align oneof %s to %d-byte boundary", oneofLayout.Oneof.Name, alignment),
					})
					currentOffset += paddingSize
				}

				// Set oneof offset
				oneofLayout.Offset = currentOffset
				oneofLayout.DiscriminatorOffset = currentOffset

				// Update field offsets within oneof (absolute offsets)
				for _, fieldLayout := range oneofLayout.Fields {
					fieldLayout.Offset = currentOffset + 1 // After discriminator
				}

				layout.Oneofs = append(layout.Oneofs, oneofLayout)
				currentOffset += oneofLayout.Size
				oneofIdx++
			}
		}

		// Add final padding for message alignment if specified
		if msg.Align > 0 && currentOffset%msg.Align != 0 {
			paddingSize := msg.Align - (currentOffset % msg.Align)
			layout.PaddingBytes = append(layout.PaddingBytes, PaddingInfo{
				Offset: currentOffset,
				Size:   paddingSize,
				Reason: fmt.Sprintf("align message to %d-byte boundary", msg.Align),
			})
			currentOffset += paddingSize
		}

		layout.TotalSize = currentOffset
	}

	// Calculate total size including message ID header
	if layout.HasMessageId {
		layout.MessageTotalSize = layout.TotalSize + layout.MessageIdSize
	} else {
		layout.MessageTotalSize = layout.TotalSize
	}

	// Store layout
	a.layouts[msg.Name] = layout

	return layout, nil
}

// calculateFieldSize returns the size in bytes for a field
func (a *LayoutAnalyzer) calculateFieldSize(field *parser.Field) (uint32, error) {
	baseSize, err := a.getBaseTypeSize(field)
	if err != nil {
		return 0, err
	}

	// Handle arrays
	if field.Repeated {
		if field.ArraySize == 0 {
			return 0, fmt.Errorf("repeated field missing (binary.array_size) option")
		}
		return baseSize * field.ArraySize, nil
	}

	return baseSize, nil
}

// getBaseTypeSize returns the size of a single element of the field type
func (a *LayoutAnalyzer) getBaseTypeSize(field *parser.Field) (uint32, error) {
	switch field.Type {
	case parser.TypeBool:
		return 1, nil
	case parser.TypeInt32, parser.TypeUint32:
		return 4, nil
	case parser.TypeInt64, parser.TypeUint64:
		return 8, nil
	case parser.TypeFloat:
		return 4, nil
	case parser.TypeDouble:
		return 8, nil
	case parser.TypeString:
		if field.StringSize == 0 {
			return 0, fmt.Errorf("string field missing (binary.string_size) option")
		}
		return field.StringSize, nil
	case parser.TypeBytes:
		if field.ArraySize == 0 {
			return 0, fmt.Errorf("bytes field missing (binary.array_size) option")
		}
		return field.ArraySize, nil
	case parser.TypeMessage:
		if field.MessageType == nil {
			return 0, fmt.Errorf("message type not resolved")
		}
		// Recursively analyze nested message
		nestedLayout, err := a.analyzeMessage(field.MessageType, a.schema)
		if err != nil {
			return 0, err
		}
		return nestedLayout.TotalSize, nil
	case parser.TypeEnum:
		if field.EnumType == nil {
			return 0, fmt.Errorf("enum type not resolved")
		}
		return field.EnumType.Size, nil
	default:
		return 0, fmt.Errorf("unsupported field type: %v", field.Type)
	}
}

// getFieldAlignment returns the natural alignment requirement for a field
func (a *LayoutAnalyzer) getFieldAlignment(field *parser.Field) uint32 {
	switch field.Type {
	case parser.TypeBool:
		return 1
	case parser.TypeInt32, parser.TypeUint32, parser.TypeFloat:
		return 4
	case parser.TypeInt64, parser.TypeUint64, parser.TypeDouble:
		return 8
	case parser.TypeString:
		return 1 // char array, no special alignment
	case parser.TypeBytes:
		return 1
	case parser.TypeMessage:
		// Message alignment depends on its internal fields
		// For now, use 1-byte alignment unless message specifies otherwise
		if field.MessageType != nil && field.MessageType.Align > 0 {
			return field.MessageType.Align
		}
		return 1
	case parser.TypeEnum:
		if field.EnumType != nil {
			return field.EnumType.Size
		}
		return 4 // Default enum alignment
	default:
		return 1
	}
}

// getMinFieldNumber returns the minimum field number in a oneof
func getMinFieldNumber(oneof *parser.Oneof) int32 {
	if len(oneof.Fields) == 0 {
		return 0
	}
	minNum := oneof.Fields[0].Number
	for _, field := range oneof.Fields[1:] {
		if field.Number < minNum {
			minNum = field.Number
		}
	}
	return minNum
}

// getOneofAlignment returns the maximum alignment requirement of all variants in a oneof
func (a *LayoutAnalyzer) getOneofAlignment(oneofLayout *OneofLayout) uint32 {
	maxAlign := uint32(1)
	for _, fieldLayout := range oneofLayout.Fields {
		align := a.getFieldAlignment(fieldLayout.Field)
		if align > maxAlign {
			maxAlign = align
		}
	}
	return maxAlign
}
