package parser

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Schema represents a parsed proto file with custom options
type Schema struct {
	FileName        string
	Package         string
	GoPackage       string // Go package name extracted from protobuf package or go_package option
	GoPackageImport string // Full go_package import path (e.g., "github.com/example/mypackage")
	Fixed           bool
	Endian          string // "little" or "big"
	Version         string
	MessageIdSize   uint32 // Message ID encoding size (1,2,4,8), default: 4
	Messages        []*Message
	Enums           []*Enum
}

func (s *Schema) HasType(kinds ...FieldType) bool {
	if len(kinds) == 0 || len(s.Messages) == 0 {
		return false
	}

	for _, kind := range kinds {
		if !kind.Valid() {
			return false
		}
	}

	for _, msg := range s.Messages {
		if msg.HasType(kinds...) {
			return true
		}
	}

	return false
}

func (s *Schema) BaseFilenameWithoutExt(capitalize bool) string {
	if s.FileName == "" {
		return ""
	}

	filename := strings.TrimSuffix(filepath.Base(s.FileName), filepath.Ext(s.FileName))

	if !capitalize {
		return filename
	}

	return strings.ToUpper(filename[0:1]) + filename[1:]
}

// Message represents a proto message definition
type Message struct {
	Name      string
	Size      uint32 // Optional, for validation (0 = not specified)
	Align     uint32 // Alignment requirement (0 = natural alignment)
	Union     bool   // If true, all fields overlay at offset 0
	MessageId uint32 // Message identifier (0 = not specified)
	Fields    []*Field
	Oneofs    []*Oneof
	SourcePos string // For error reporting
}

func (m *Message) HasType(kinds ...FieldType) bool {
	if len(kinds) == 0 || (len(m.Fields) == 0 && len(m.Oneofs) == 0) {
		return false
	}

	for _, field := range m.Fields {
		if slices.Contains(kinds, field.Type) {
			return true
		}
	}

	for _, oneof := range m.Oneofs {
		if oneof.HasType(kinds...) {
			return true
		}
	}

	return false
}

// Oneof represents a oneof group
type Oneof struct {
	Name      string
	Fields    []*Field
	SourcePos string
}

func (o *Oneof) HasType(kinds ...FieldType) bool {
	if len(kinds) == 0 || len(o.Fields) == 0 {
		return false
	}

	for _, field := range o.Fields {
		if slices.Contains(kinds, field.Type) {
			return true
		}
	}

	return false
}

// Field represents a message field
type Field struct {
	Name        string
	Number      int32
	Type        FieldType
	Repeated    bool
	ArraySize   uint32 // For repeated fields (0 = not specified)
	StringSize  uint32 // For string fields (0 = not specified)
	MessageType *Message
	EnumType    *Enum
	OneofIndex  int    // Index of oneof this field belongs to (-1 if not in oneof)
	SourcePos   string // For error reporting
}

// FieldType represents the field's data type
type FieldType int

const (
	TypeUnknown FieldType = iota
	TypeBool
	TypeInt32
	TypeUint32
	TypeInt64
	TypeUint64
	TypeFloat
	TypeDouble
	TypeString
	TypeBytes
	TypeMessage
	TypeEnum
)

func (f FieldType) Valid() bool {
	switch f {
	case TypeBool, TypeInt32, TypeUint32, TypeInt64, TypeUint64, TypeFloat, TypeDouble, TypeString, TypeBytes, TypeMessage, TypeEnum:
		return true
	default:
		return false
	}
}

// Enum represents an enum definition
type Enum struct {
	Name      string
	Size      uint32 // 1, 2, or 4 bytes (0 = default 4)
	Values    []*EnumValue
	SourcePos string
}

// EnumValue represents an enum constant
type EnumValue struct {
	Name   string
	Number int32
}

// Parser parses proto files and extracts fixed binary schema information
type Parser struct {
	importPaths []string
}

// NewParser creates a new parser with optional import paths
func NewParser(importPaths ...string) *Parser {
	return &Parser{
		importPaths: importPaths,
	}
}

// Parse parses a proto file and returns a Schema
func (p *Parser) Parse(filename string) (*Schema, error) {
	// Resolve absolute path
	absPath, apErr := filepath.Abs(filename)

	if apErr != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", apErr)
	}

	// Check file exists
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	// Setup parser with import paths
	// Add the directory containing the proto file and the current working directory
	importPaths := append(p.importPaths, filepath.Dir(absPath))

	if cwd, err := os.Getwd(); err == nil {
		importPaths = append(importPaths, cwd)
	}

	parser := protoparse.Parser{
		ImportPaths:           importPaths,
		IncludeSourceCodeInfo: true,
	}

	// Parse the file
	fds, pErr := parser.ParseFiles(filepath.Base(absPath))

	if pErr != nil {
		return nil, fmt.Errorf("failed to parse proto file: %w", pErr)
	}

	if len(fds) == 0 {
		return nil, fmt.Errorf("no file descriptors parsed")
	}

	fd := fds[0]

	// check for proto3 syntax
	if !fd.IsProto3() {
		return nil, fmt.Errorf("only proto3 syntax is supported")
	}

	// Extract file-level options
	goPackage, goPackageImport := getGoPackageInfo(fd)

	schema := &Schema{
		FileName:        filename,
		Package:         fd.GetPackage(),
		GoPackage:       goPackage,
		GoPackageImport: goPackageImport,
		Fixed:           getFileOptionBool(fd, "binary.fixed"),
		Endian:          getFileOptionString(fd, "binary.endian"),
		Version:         getFileOptionString(fd, "binary.version"),
		MessageIdSize:   getFileOptionUint32(fd, "binary.message_id_size"),
	}

	// proto file must define a package
	if schema.Package == "" {
		return nil, fmt.Errorf("no package defined in proto file")
	}

	// Default endian to "little" if not specified
	if schema.Endian == "" {
		schema.Endian = "little"
	}

	// Default version to "v1.0.0" if not specified
	if schema.Version == "" {
		schema.Version = "v1.0.0"
	}

	// Default message_id_size to 4 if not specified
	if schema.MessageIdSize == 0 {
		schema.MessageIdSize = 4
	}

	// Default fixed to true (assume all proto2fixed files are fixed binary)
	if !schema.Fixed {
		schema.Fixed = true
	}

	// Parse enums
	enumMap := make(map[string]*Enum)
	for _, enumDesc := range fd.GetEnumTypes() {
		enum := p.parseEnum(enumDesc)
		schema.Enums = append(schema.Enums, enum)
		enumMap[enum.Name] = enum
	}

	// Parse messages (first pass to build structure)
	msgMap := make(map[string]*Message)
	for _, msgDesc := range fd.GetMessageTypes() {
		msg := p.parseMessage(msgDesc, msgMap, enumMap)
		schema.Messages = append(schema.Messages, msg)
		msgMap[msg.Name] = msg
	}

	return schema, nil
}

func (p *Parser) parseMessage(md *desc.MessageDescriptor, msgMap map[string]*Message, enumMap map[string]*Enum) *Message {
	msg := &Message{
		Name:      md.GetName(),
		Size:      getMessageOptionUint32(md, "binary.size"),
		Align:     getMessageOptionUint32(md, "binary.align"),
		Union:     getMessageOptionBool(md, "binary.union"),
		MessageId: getMessageOptionUint32(md, "binary.message_id"),
		SourcePos: getSourcePos(md),
	}

	// Parse nested enums
	for _, enumDesc := range md.GetNestedEnumTypes() {
		enum := p.parseEnum(enumDesc)
		enumMap[enum.Name] = enum
	}

	// Parse nested messages
	for _, nestedDesc := range md.GetNestedMessageTypes() {
		nestedMsg := p.parseMessage(nestedDesc, msgMap, enumMap)
		msgMap[nestedMsg.Name] = nestedMsg
	}

	// Parse oneof groups
	oneofDescs := md.GetOneOfs()
	for _, oneofDesc := range oneofDescs {
		oneof := &Oneof{
			Name:      oneofDesc.GetName(),
			SourcePos: getSourcePos(oneofDesc),
		}
		msg.Oneofs = append(msg.Oneofs, oneof)
	}

	// Parse fields
	for _, fd := range md.GetFields() {
		field := p.parseField(fd, msgMap, enumMap)

		// Check if field is part of a oneof
		if oneofDesc := fd.GetOneOf(); oneofDesc != nil {
			// Find the oneof index
			for i, oneof := range msg.Oneofs {
				if oneof.Name == oneofDesc.GetName() {
					field.OneofIndex = i
					msg.Oneofs[i].Fields = append(msg.Oneofs[i].Fields, field)
					break
				}
			}
		} else {
			field.OneofIndex = -1
		}

		msg.Fields = append(msg.Fields, field)
	}

	return msg
}

func (p *Parser) parseField(fd *desc.FieldDescriptor, msgMap map[string]*Message, enumMap map[string]*Enum) *Field {
	field := &Field{
		Name:       fd.GetName(),
		Number:     fd.GetNumber(),
		Repeated:   fd.IsRepeated(),
		ArraySize:  getFieldOptionUint32(fd, "binary.array_size"),
		StringSize: getFieldOptionUint32(fd, "binary.string_size"),
		SourcePos:  getSourcePos(fd),
	}

	// Determine field type
	switch fd.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		field.Type = TypeBool
	case descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_SINT32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		field.Type = TypeInt32
	case descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		field.Type = TypeUint32
	case descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_SINT64, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		field.Type = TypeInt64
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		field.Type = TypeUint64
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		field.Type = TypeFloat
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		field.Type = TypeDouble
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		field.Type = TypeString
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		field.Type = TypeBytes
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		field.Type = TypeMessage
		// Reference to nested or external message
		msgType := fd.GetMessageType()
		if msg, exists := msgMap[msgType.GetName()]; exists {
			field.MessageType = msg
		} else {
			// Parse nested message on-demand
			field.MessageType = p.parseMessage(msgType, msgMap, enumMap)
			msgMap[msgType.GetName()] = field.MessageType
		}
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		field.Type = TypeEnum
		enumType := fd.GetEnumType()
		if enum, exists := enumMap[enumType.GetName()]; exists {
			field.EnumType = enum
		} else {
			// Parse nested enum on-demand
			field.EnumType = p.parseEnum(enumType)
			enumMap[enumType.GetName()] = field.EnumType
		}
	default:
		field.Type = TypeUnknown
	}

	return field
}

func (p *Parser) parseEnum(ed *desc.EnumDescriptor) *Enum {
	enum := &Enum{
		Name:      ed.GetName(),
		Size:      getEnumOptionUint32(ed, "binary.enum_size"),
		SourcePos: getSourcePos(ed),
	}

	// Default enum size to 4 bytes (int32)
	if enum.Size == 0 {
		enum.Size = 4
	}

	for _, vd := range ed.GetValues() {
		enum.Values = append(enum.Values, &EnumValue{
			Name:   vd.GetName(),
			Number: vd.GetNumber(),
		})
	}

	return enum
}

// Helper functions to extract custom built-in options
// These look for uninterpreted options matching binary.* pattern
// This allows proto2fixed to work without requiring an binary.proto import

func getFileOptionBool(fd *desc.FileDescriptor, optionName string) bool {
	opts := fd.GetFileOptions()
	if opts == nil {
		return false
	}

	// Check uninterpreted options for both "binary.xxx" and "(binary.xxx)" syntax
	for _, opt := range opts.UninterpretedOption {
		if matchesOptionName(opt, optionName) {
			if opt.IdentifierValue != nil && *opt.IdentifierValue == "true" {
				return true
			}
		}
	}
	return false
}

func getGoPackageInfo(fd *desc.FileDescriptor) (packageName string, importPath string) {
	// defaults to using the package name as defined in the proto file
	packageParts := strings.Split(fd.GetPackage(), ".")

	if len(packageParts) != 0 {
		packageName = packageParts[len(packageParts)-1]
		importPath = strings.Join(packageParts, "/")
	}

	// the go_package option can be used to override the package name and import path
	opts := fd.GetFileOptions()

	if opts == nil || opts.GoPackage == nil {
		return packageName, importPath
	}

	goPackage := *opts.GoPackage

	// go_package can be "path" or "path;package"
	// If it contains a semicolon, use the part after it as the package name
	if idx := strings.Index(goPackage, ";"); idx != -1 {
		importPath = goPackage[:idx]
		packageName = goPackage[idx+1:]

		return packageName, importPath
	}

	// Otherwise, use the last component of the path as the package name
	importPath = goPackage
	packageName = goPackage

	if idx := strings.LastIndex(goPackage, "/"); idx != -1 {
		packageName = goPackage[idx+1:]
	}

	return packageName, importPath
}

func getFileOptionString(fd *desc.FileDescriptor, optionName string) string {
	opts := fd.GetFileOptions()
	if opts == nil {
		return ""
	}

	// Check uninterpreted options
	for _, opt := range opts.UninterpretedOption {
		if matchesOptionName(opt, optionName) {
			if opt.StringValue != nil {
				return string(opt.StringValue)
			}
		}
	}
	return ""
}

func getFileOptionUint32(fd *desc.FileDescriptor, optionName string) uint32 {
	opts := fd.GetFileOptions()
	if opts == nil {
		return 0
	}

	// Try reflection first (for compiled/resolved extensions)
	if val := getExtensionUint32(opts, optionName); val > 0 {
		return val
	}

	// Fallback to uninterpreted options (for unresolved extensions)
	for _, opt := range opts.UninterpretedOption {
		if matchesOptionName(opt, optionName) {
			if opt.PositiveIntValue != nil {
				return uint32(*opt.PositiveIntValue)
			}
		}
	}
	return 0
}

func getMessageOptionUint32(md *desc.MessageDescriptor, optionName string) uint32 {
	opts := md.GetMessageOptions()
	if opts == nil {
		return 0
	}

	// First try to get via reflection (for compiled/resolved extensions)
	if val := getExtensionUint32(opts, optionName); val > 0 {
		return val
	}

	// Fallback to uninterpreted options (for unresolved extensions)
	for _, opt := range opts.UninterpretedOption {
		if matchesOptionName(opt, optionName) {
			if opt.PositiveIntValue != nil {
				return uint32(*opt.PositiveIntValue)
			}
		}
	}
	return 0
}

func getMessageOptionBool(md *desc.MessageDescriptor, optionName string) bool {
	opts := md.GetMessageOptions()
	if opts == nil {
		return false
	}

	// First try to get via reflection (for compiled/resolved extensions)
	if val := getExtensionBool(opts, optionName); val {
		return true
	}

	// Fallback to uninterpreted options (for unresolved extensions)
	for _, opt := range opts.UninterpretedOption {
		if matchesOptionName(opt, optionName) {
			if opt.IdentifierValue != nil && *opt.IdentifierValue == "true" {
				return true
			}
		}
	}
	return false
}

func getFieldOptionUint32(fd *desc.FieldDescriptor, optionName string) uint32 {
	opts := fd.GetFieldOptions()
	if opts == nil {
		return 0
	}

	// First try to get via reflection (for compiled/resolved extensions)
	if val := getExtensionUint32(opts, optionName); val != 0 {
		return val
	}

	// Fallback to uninterpreted options (for unresolved extensions)
	for _, opt := range opts.UninterpretedOption {
		if matchesOptionName(opt, optionName) {
			if opt.PositiveIntValue != nil {
				return uint32(*opt.PositiveIntValue)
			}
		}
	}
	return 0
}

func getEnumOptionUint32(ed *desc.EnumDescriptor, optionName string) uint32 {
	opts := ed.GetEnumOptions()
	if opts == nil {
		return 0
	}

	// First try to get via reflection (for compiled/resolved extensions)
	if val := getExtensionUint32(opts, optionName); val != 0 {
		return val
	}

	// Fallback to uninterpreted options (for unresolved extensions)
	for _, opt := range opts.UninterpretedOption {
		if matchesOptionName(opt, optionName) {
			if opt.PositiveIntValue != nil {
				return uint32(*opt.PositiveIntValue)
			}
		}
	}
	return 0
}

// getExtensionUint32 extracts a uint32 extension value from a proto message using reflection
func getExtensionUint32(msg proto.Message, optionName string) uint32 {
	if msg == nil {
		return 0
	}

	// Map extension names to field numbers (from proto2fixed/binary.proto)
	extensionNumbers := map[string]int32{
		"binary.array_size":      50020,
		"binary.string_size":     50021,
		"binary.reserved_size":   50022,
		"binary.enum_size":       50030,
		"binary.size":            50010,
		"binary.align":           50011,
		"binary.message_id":      50013,
		"binary.message_id_size": 50004,
		"array_size":             50020,
		"string_size":            50021,
		"reserved_size":          50022,
		"enum_size":              50030,
		"size":                   50010,
		"align":                  50011,
		"message_id":             50013,
		"message_id_size":        50004,
	}

	fieldNum, ok := extensionNumbers[optionName]
	if !ok {
		return 0
	}

	// Use protobuf reflection to access extension fields
	ref := msg.ProtoReflect()

	// Iterate over all fields including extensions
	var result uint32
	ref.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if int32(fd.Number()) == fieldNum {
			// Found the extension field
			if fd.Kind() == protoreflect.Uint32Kind {
				result = uint32(v.Uint())
				return false // stop iteration
			}
		}
		return true // continue iteration
	})

	if result != 0 {
		return result
	}

	// Try using unknown fields as fallback
	unknownFields := ref.GetUnknown()
	if len(unknownFields) > 0 {
		// Parse unknown fields to find our extension
		return parseUnknownFieldUint32(unknownFields, fieldNum)
	}

	return 0
}

// getExtensionBool extracts a bool extension value from a proto message using reflection
func getExtensionBool(msg proto.Message, optionName string) bool {
	if msg == nil {
		return false
	}

	// Map extension names to field numbers (from proto2fixed/binary.proto)
	extensionNumbers := map[string]int32{
		"binary.fixed": 50001,
		"binary.union": 50012,
		"fixed":        50001,
		"union":        50012,
	}

	fieldNum, ok := extensionNumbers[optionName]
	if !ok {
		return false
	}

	// Use protobuf reflection to access extension fields
	ref := msg.ProtoReflect()

	// Iterate over all fields including extensions
	var result bool
	ref.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if int32(fd.Number()) == fieldNum {
			// Found the extension field
			if fd.Kind() == protoreflect.BoolKind {
				result = v.Bool()
				return false // stop iteration
			}
		}
		return true // continue iteration
	})

	if result {
		return true
	}

	// Try using unknown fields as fallback
	unknownFields := ref.GetUnknown()
	if len(unknownFields) > 0 {
		// Parse unknown fields to find our extension
		return parseUnknownFieldBool(unknownFields, fieldNum)
	}

	return false
}

// parseUnknownFieldBool parses unknown fields to extract a bool value by field number
func parseUnknownFieldBool(unknownFields protoreflect.RawFields, fieldNum int32) bool {
	data := []byte(unknownFields)
	offset := 0

	for offset < len(data) {
		// Read tag
		tag, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			break
		}
		offset += n

		fieldNumber := int32(tag >> 3)
		wireType := tag & 0x7

		if fieldNumber == fieldNum && wireType == 0 {
			// Found our field! Read the varint value (0 or 1)
			val, n := binary.Uvarint(data[offset:])
			if n > 0 {
				return val != 0
			}
			return false
		}

		// Skip this field based on wire type
		switch wireType {
		case 0: // Varint
			_, n := binary.Uvarint(data[offset:])
			if n <= 0 {
				return false
			}
			offset += n
		case 1: // 64-bit
			offset += 8
		case 2: // Length-delimited
			length, n := binary.Uvarint(data[offset:])
			if n <= 0 {
				return false
			}
			offset += n + int(length)
		case 5: // 32-bit
			offset += 4
		default:
			return false
		}
	}

	return false
}

// parseUnknownFieldUint32 parses unknown fields to extract a uint32 value by field number
func parseUnknownFieldUint32(unknownFields protoreflect.RawFields, fieldNum int32) uint32 {
	// Parse the unknown fields wire format
	// Format: tag (varint) + value
	// Tag = (field_number << 3) | wire_type
	// For uint32, wire_type = 0 (varint)

	data := []byte(unknownFields)
	offset := 0

	for offset < len(data) {
		// Read tag
		tag, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			break
		}
		offset += n

		fieldNumber := int32(tag >> 3)
		wireType := tag & 0x7

		if fieldNumber == fieldNum && wireType == 0 {
			// Found our field! Read the varint value
			val, n := binary.Uvarint(data[offset:])
			if n > 0 {
				return uint32(val)
			}
			return 0
		}

		// Skip this field based on wire type
		switch wireType {
		case 0: // Varint
			_, n := binary.Uvarint(data[offset:])
			if n <= 0 {
				return 0
			}
			offset += n
		case 1: // 64-bit
			offset += 8
		case 2: // Length-delimited
			length, n := binary.Uvarint(data[offset:])
			if n <= 0 {
				return 0
			}
			offset += n + int(length)
		case 5: // 32-bit
			offset += 4
		default:
			// Unknown wire type
			return 0
		}
	}

	return 0
}

// matchesOptionName checks if an uninterpreted option matches the given name
// Supports various formats: "binary.size", "binary", "size"
func matchesOptionName(opt *descriptorpb.UninterpretedOption, targetName string) bool {
	if len(opt.Name) == 0 {
		return false
	}

	// Build the full option name from the parts
	var fullName string
	for i, part := range opt.Name {
		if i > 0 {
			fullName += "."
		}
		fullName += part.GetNamePart()
	}

	// Match exact name
	if fullName == targetName {
		return true
	}

	// Also try matching just the last part (e.g., "size" matches "binary.size")
	if len(opt.Name) > 0 {
		lastName := opt.Name[len(opt.Name)-1].GetNamePart()
		// Extract the last part of targetName
		targetParts := make([]string, 0)
		for i := len(targetName) - 1; i >= 0; i-- {
			if targetName[i] == '.' {
				targetParts = append([]string{targetName[i+1:]}, targetParts...)
				break
			}
			if i == 0 {
				targetParts = []string{targetName}
			}
		}
		if len(targetParts) > 0 && lastName == targetParts[0] {
			return true
		}
	}

	return false
}

func getSourcePos(d desc.Descriptor) string {
	loc := d.GetSourceInfo()
	if loc == nil {
		return ""
	}
	// Format: filename:line:col
	return fmt.Sprintf("%s:%d:%d", d.GetFile().GetName(), loc.GetSpan()[0]+1, loc.GetSpan()[1]+1)
}
