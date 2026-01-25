package analyzer

import (
	"fmt"
	"strings"

	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

// ValidationError represents a validation error with source location
type ValidationError struct {
	Message   string
	SourcePos string
}

func (e *ValidationError) Error() string {
	if e.SourcePos != "" {
		return fmt.Sprintf("Error: %s\n    %s", e.SourcePos, e.Message)
	}
	return fmt.Sprintf("Error: %s", e.Message)
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Message   string
	SourcePos string
}

func (w *ValidationWarning) String() string {
	if w.SourcePos != "" {
		return fmt.Sprintf("Warning: %s\n    %s", w.SourcePos, w.Message)
	}
	return fmt.Sprintf("Warning: %s", w.Message)
}

// ValidationResult contains validation errors and warnings
type ValidationResult struct {
	Errors   []*ValidationError
	Warnings []*ValidationWarning
}

// HasErrors returns true if there are any errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) != 0
}

func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) != 0
}

func (r *ValidationResult) HasErrorsOrWarnings() bool {
	return r.HasErrors() || r.HasWarnings()
}

// AddError adds a validation error
func (r *ValidationResult) AddError(sourcePos, message string, args ...any) {
	r.Errors = append(r.Errors, &ValidationError{
		Message:   fmt.Sprintf(message, args...),
		SourcePos: sourcePos,
	})
}

// AddWarning adds a validation warning
func (r *ValidationResult) AddWarning(sourcePos, message string, args ...any) {
	r.Warnings = append(r.Warnings, &ValidationWarning{
		Message:   fmt.Sprintf(message, args...),
		SourcePos: sourcePos,
	})
}

// String returns formatted error and warning messages
func (r *ValidationResult) String() string {
	var sb strings.Builder
	for _, err := range r.Errors {
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}
	for _, warn := range r.Warnings {
		sb.WriteString(warn.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

// Validator validates proto schemas for fixed binary encoding
type Validator struct {
	analyzer *LayoutAnalyzer
	result   *ValidationResult
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		analyzer: NewLayoutAnalyzer(),
		result:   &ValidationResult{},
	}
}

// Validate validates a schema and returns validation results
func (v *Validator) Validate(schema *parser.Schema) (*ValidationResult, error) {
	v.result = &ValidationResult{}

	// Validate file-level options
	v.validateFileOptions(schema)

	// Analyze layouts
	if err := v.analyzer.Analyze(schema); err != nil {
		return nil, err
	}

	// Validate each message
	for _, msg := range schema.Messages {
		v.validateMessage(msg, schema)
	}

	// Validate enums
	for _, enum := range schema.Enums {
		v.validateEnum(enum)
	}

	// Validate message IDs
	v.validateMessageIds(schema)

	return v.result, nil
}

// GetAnalyzer returns the layout analyzer with computed layouts
func (v *Validator) GetAnalyzer() *LayoutAnalyzer {
	return v.analyzer
}

func (v *Validator) validateFileOptions(schema *parser.Schema) {
	if !schema.Fixed {
		v.result.AddError("", "File must have option (binary.fixed) = true")
	}

	if schema.Endian != "little" && schema.Endian != "big" {
		v.result.AddError("", "Invalid endian option: %s (must be 'little' or 'big')", schema.Endian)
	}
}

func (v *Validator) validateMessage(msg *parser.Message, schema *parser.Schema) {
	layout, exists := v.analyzer.GetLayout(msg.Name)
	if !exists {
		v.result.AddError(msg.SourcePos, "Message %s: layout not computed", msg.Name)
		return
	}

	// Validate declared size matches calculated size (if specified)
	if msg.Size > 0 && msg.Size != layout.TotalSize {
		v.result.AddError(msg.SourcePos,
			"Message '%s' declared size (%d) != calculated size (%d)\n"+
				"Calculated: %d bytes\n"+
				"Either remove (binary.size) option or adjust to match calculated size",
			msg.Name, msg.Size, layout.TotalSize, layout.TotalSize)
	}

	// Validate alignment is power of 2
	if msg.Align > 0 && !isPowerOfTwo(msg.Align) {
		v.result.AddError(msg.SourcePos,
			"Message '%s' alignment (%d) must be a power of 2",
			msg.Name, msg.Align)
	}

	// Check for field number gaps
	if !msg.Union {
		v.checkFieldNumberGaps(msg)
	}

	// Validate fields
	for _, field := range msg.Fields {
		v.validateField(field, msg)
	}

	// Check for misaligned fields
	for _, fieldLayout := range layout.Fields {
		alignment := v.analyzer.getFieldAlignment(fieldLayout.Field)
		if alignment > 1 && fieldLayout.Offset%alignment != 0 {
			v.result.AddWarning(fieldLayout.Field.SourcePos,
				"Field '%s.%s' (field number %d) at offset %d is not %d-byte aligned",
				msg.Name, fieldLayout.Field.Name, fieldLayout.Field.Number, fieldLayout.Offset, alignment)
		}
	}

	// Validate discriminator field numbers
	v.validateDiscriminatorFieldNumbers(msg)
}

func (v *Validator) validateField(field *parser.Field, msg *parser.Message) {
	// Validate repeated fields have array size
	if field.Repeated && field.ArraySize == 0 {
		v.result.AddError(field.SourcePos,
			"Field '%s.%s' is repeated but missing (binary.array_size) option\n"+
				"Add: repeated %s %s = %d [(binary.array_size) = <count>];",
			msg.Name, field.Name, fieldTypeName(field), field.Name, field.Number)
	}

	// Validate string fields have string size
	if field.Type == parser.TypeString && field.StringSize == 0 {
		v.result.AddError(field.SourcePos,
			"Field '%s.%s' is type string but missing (binary.string_size) option\n"+
				"Add: string %s = %d [(binary.string_size) = <size>];",
			msg.Name, field.Name, field.Name, field.Number)
	}

	// Validate bytes fields (treated as fixed-size arrays)
	if field.Type == parser.TypeBytes && !field.Repeated && field.ArraySize == 0 {
		v.result.AddError(field.SourcePos,
			"Field '%s.%s' is type bytes but missing (binary.array_size) option\n"+
				"Fixed binary mode requires explicit size for bytes fields",
			msg.Name, field.Name)
	}

	// Validate string size is reasonable
	if field.Type == parser.TypeString && field.StringSize > 1024 {
		v.result.AddWarning(field.SourcePos,
			"Field '%s.%s' string size (%d) is very large (>1KB). Consider if this is intended.",
			msg.Name, field.Name, field.StringSize)
	}

	// Validate nested message types are resolved
	if field.Type == parser.TypeMessage && field.MessageType == nil {
		v.result.AddError(field.SourcePos,
			"Field '%s.%s' message type not resolved",
			msg.Name, field.Name)
	}

	// Validate enum types are resolved
	if field.Type == parser.TypeEnum && field.EnumType == nil {
		v.result.AddError(field.SourcePos,
			"Field '%s.%s' enum type not resolved",
			msg.Name, field.Name)
	}
}

func (v *Validator) validateEnum(enum *parser.Enum) {
	// Validate enum size
	if enum.Size != 1 && enum.Size != 2 && enum.Size != 4 {
		v.result.AddError(enum.SourcePos,
			"Enum '%s' size (%d) must be 1, 2, or 4 bytes",
			enum.Name, enum.Size)
	}

	// Validate enum has values
	if len(enum.Values) == 0 {
		v.result.AddError(enum.SourcePos,
			"Enum '%s' has no values defined",
			enum.Name)
	}

	// Validate enum values fit in specified size
	for _, val := range enum.Values {
		if !enumValueFitsInSize(val.Number, enum.Size) {
			v.result.AddError(enum.SourcePos,
				"Enum '%s' value '%s' (%d) does not fit in %d byte(s)",
				enum.Name, val.Name, val.Number, enum.Size)
		}
	}
}

func (v *Validator) checkFieldNumberGaps(msg *parser.Message) {
	if len(msg.Fields) == 0 {
		return
	}

	// Sort field numbers
	numbers := make([]int32, len(msg.Fields))
	for i, field := range msg.Fields {
		numbers[i] = field.Number
	}

	// Find gaps
	var gaps []int32
	for i := 0; i < len(numbers)-1; i++ {
		diff := numbers[i+1] - numbers[i]
		if diff > 1 {
			for j := numbers[i] + 1; j < numbers[i+1]; j++ {
				gaps = append(gaps, j)
			}
		}
	}

	if len(gaps) > 0 {
		gapStr := make([]string, len(gaps))
		for i, g := range gaps {
			gapStr[i] = fmt.Sprintf("%d", g)
		}
		v.result.AddWarning(msg.SourcePos,
			"Message '%s' has gaps in field numbers: %s\n"+
				"This may create suboptimal padding. Consider renumbering fields sequentially or using reserved fields with (binary.reserved_size).",
			msg.Name, strings.Join(gapStr, ", "))
	}
}

func isPowerOfTwo(n uint32) bool {
	return n > 0 && (n&(n-1)) == 0
}

func enumValueFitsInSize(value int32, size uint32) bool {
	switch size {
	case 1:
		return value >= -128 && value <= 127
	case 2:
		return value >= -32768 && value <= 32767
	case 4:
		return true // int32 always fits
	default:
		return false
	}
}

func fieldTypeName(field *parser.Field) string {
	switch field.Type {
	case parser.TypeBool:
		return "bool"
	case parser.TypeInt32:
		return "int32"
	case parser.TypeUint32:
		return "uint32"
	case parser.TypeInt64:
		return "int64"
	case parser.TypeUint64:
		return "uint64"
	case parser.TypeFloat:
		return "float"
	case parser.TypeDouble:
		return "double"
	case parser.TypeString:
		return "string"
	case parser.TypeBytes:
		return "bytes"
	case parser.TypeMessage:
		if field.MessageType != nil {
			return field.MessageType.Name
		}
		return "message"
	case parser.TypeEnum:
		if field.EnumType != nil {
			return field.EnumType.Name
		}
		return "enum"
	default:
		return "unknown"
	}
}

// validateMessageIds validates message ID constraints
func (v *Validator) validateMessageIds(schema *parser.Schema) {
	// If MessageIdSize is 0, message IDs are not being used - skip validation
	// (The parser sets a default of 4, but manually created schemas in tests may have 0)
	if schema.MessageIdSize == 0 {
		return
	}

	// Validate file-level message_id_size
	if schema.MessageIdSize != 1 && schema.MessageIdSize != 2 &&
		schema.MessageIdSize != 4 && schema.MessageIdSize != 8 {
		v.result.AddError("",
			"File option message_id_size (%d) must be 1, 2, 4, or 8",
			schema.MessageIdSize)
	}

	messageIdMap := make(map[uint32]string) // Maps ID -> message name
	messagesWithIds := 0
	topLevelNames := make(map[string]bool)
	nestedNames := make(map[string]bool)

	// Build top-level message set
	for _, msg := range schema.Messages {
		topLevelNames[msg.Name] = true
	}

	// Identify nested messages (messages used as field types)
	for _, msg := range schema.Messages {
		for _, field := range msg.Fields {
			if field.Type == parser.TypeMessage && field.MessageType != nil {
				nestedNames[field.MessageType.Name] = true
			}
		}
	}

	// Count top-level messages
	topLevelCount := 0
	for _, msg := range schema.Messages {
		isTopLevel := topLevelNames[msg.Name] && !nestedNames[msg.Name]
		if isTopLevel {
			topLevelCount++
		}
	}

	// Validate each message
	for _, msg := range schema.Messages {
		isTopLevel := topLevelNames[msg.Name] && !nestedNames[msg.Name]

		if msg.MessageId > 0 {
			// Check if nested message has message_id (warning)
			if !isTopLevel {
				v.result.AddWarning(msg.SourcePos,
					"Message '%s' is nested and should not have message_id option (will be ignored)",
					msg.Name)
				continue // Skip further validation for this message's ID
			}

			// Check for maximum value based on size
			maxValue := uint64(1)<<(schema.MessageIdSize*8) - 1
			if uint64(msg.MessageId) > maxValue {
				v.result.AddError(msg.SourcePos,
					"Message '%s' message_id (%d) exceeds maximum for size %d bytes (max: %d)",
					msg.Name, msg.MessageId, schema.MessageIdSize, maxValue)
			}

			// Check for duplicate message IDs
			if existingMsg, exists := messageIdMap[msg.MessageId]; exists {
				v.result.AddError(msg.SourcePos,
					"Message '%s' has duplicate message_id %d (already used by '%s')",
					msg.Name, msg.MessageId, existingMsg)
			} else {
				messageIdMap[msg.MessageId] = msg.Name
				messagesWithIds++
			}
		} else if isTopLevel {
			// Top-level message without message_id (warning)
			v.result.AddWarning(msg.SourcePos,
				"Top-level message '%s' does not have a message_id option",
				msg.Name)
		}
	}

	// Error if no top-level messages have message IDs
	if topLevelCount > 0 && messagesWithIds == 0 {
		v.result.AddError("",
			"Schema must have at least one top-level message with a message_id option")
	}
}

// validateDiscriminatorFieldNumbers validates field numbers for union messages and oneof fields
func (v *Validator) validateDiscriminatorFieldNumbers(msg *parser.Message) {
	// Union messages - validate all field numbers are <= 255
	if msg.Union {
		for _, field := range msg.Fields {
			if field.OneofIndex == -1 && field.Number > 255 {
				v.result.AddError(field.SourcePos,
					"Union message '%s' field '%s' number (%d) exceeds 255. "+
						"Discriminator is uint8, maximum field number is 255.",
					msg.Name, field.Name, field.Number)
			}
		}
	}

	// Oneof variants - validate all variant numbers are <= 255
	for _, oneof := range msg.Oneofs {
		for _, field := range oneof.Fields {
			if field.Number > 255 {
				v.result.AddError(field.SourcePos,
					"Oneof '%s.%s' variant '%s' number (%d) exceeds 255. "+
						"Discriminator is uint8, maximum field number is 255.",
					msg.Name, oneof.Name, field.Name, field.Number)
			}
		}
	}
}
