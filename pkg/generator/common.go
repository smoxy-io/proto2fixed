package generator

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/smoxy-io/proto2fixed/pkg/analyzer"
	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

// Language represents a supported code generation language
type Language string

func (l Language) String() string {
	return string(l)
}

func (l Language) IsValid() bool {
	switch l {
	case LangJSON, LangArduino, LangGo:
		return true
	default:
		return false
	}
}

// Generator is the interface for all code generators
type Generator interface {
	Generate(schema *parser.Schema, layouts map[string]*analyzer.MessageLayout) (string, error)
}

// OutputFile returns the output path for the file for a given language's generated code
func OutputFile(lang string, schema *parser.Schema, pathPrefixParts ...string) string {
	if schema.FileName == "" {
		return ""
	}

	pathPrefix := filepath.Join(pathPrefixParts...)
	fileName := strings.TrimSuffix(filepath.Base(schema.FileName), filepath.Ext(schema.FileName))
	//dirParts := strings.Split(filepath.Dir(schema.FileName), string(os.PathSeparator))
	dirParts := strings.Split(schema.Package, ".")

	path := filepath.Join(dirParts...)

	switch Language(lang) {
	case LangArduino:
		return filepath.Join(pathPrefix, path, fileName+".h")

	case LangJSON:
		return filepath.Join(pathPrefix, path, fileName+".json")

	case LangGo:
		base := fileName + ".fbpb.go"

		if schema.GoPackageImport == "" {
			return filepath.Join(pathPrefix, path, base)
		}

		// Use go_package import path as directory structure
		pathParts := strings.Split(schema.GoPackageImport, "/")

		return filepath.Join(pathPrefix, filepath.Join(pathParts...), base)
	}

	return ""
}

// NewGenerator creates a new generator for the given language
func NewGenerator(lang Language, opts ...GeneratorOption) Generator {
	var gen Generator

	switch lang {
	case LangArduino:
		g := NewArduinoGenerator()
		gen = g

	case LangJSON:
		g := NewJSONGenerator()
		gen = g

	case LangGo:
		g := NewGoGenerator()
		gen = g

	default:
		return nil
	}

	if len(opts) == 0 {
		return gen
	}

	for _, opt := range opts {
		opt(gen)
	}

	return gen
}

// Helper functions shared across generators

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// initialsFromCamelCase returns the initials of a camelCase string
func initialsFromCamelCase(s string) string {
	if s == "" {
		return ""
	}

	inits := s[:1]

	prev := rune(s[0])

	for _, r := range s[1:] {
		if (unicode.IsUpper(r) && !unicode.IsUpper(prev)) || (unicode.IsNumber(r) && !unicode.IsNumber(prev)) {
			inits += string(r)
		}

		prev = r
	}

	return strings.ToLower(inits)
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 0; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// getTypeNameC returns the C type name for a field
func getTypeNameC(field *parser.Field) string {
	switch field.Type {
	case parser.TypeBool:
		return "bool"
	case parser.TypeInt32:
		return "int32_t"
	case parser.TypeUint32:
		return "uint32_t"
	case parser.TypeInt64:
		return "int64_t"
	case parser.TypeUint64:
		return "uint64_t"
	case parser.TypeFloat:
		return "float"
	case parser.TypeDouble:
		return "double"
	case parser.TypeString:
		return "char"
	case parser.TypeBytes:
		return "uint8_t"
	case parser.TypeMessage:
		if field.MessageType != nil {
			return field.MessageType.Name
		}
		return "void*"
	case parser.TypeEnum:
		if field.EnumType != nil {
			return field.EnumType.Name
		}
		return "int32_t"
	default:
		return "void*"
	}
}

// getTypeNameGo returns the Go type name for a field
func getTypeNameGo(field *parser.Field) string {
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
		return "float32"
	case parser.TypeDouble:
		return "float64"
	case parser.TypeString:
		return "string"
	case parser.TypeBytes:
		return "[]byte"
	case parser.TypeMessage:
		if field.MessageType != nil {
			return field.MessageType.Name
		}
		return "any"
	case parser.TypeEnum:
		if field.EnumType != nil {
			return field.EnumType.Name
		}
		return "int32"
	default:
		return "any"
	}
}

// getJSONTypeName returns the JSON schema type name
func getJSONTypeName(field *parser.Field) string {
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
		return "float32"
	case parser.TypeDouble:
		return "float64"
	case parser.TypeString:
		return "string"
	case parser.TypeBytes:
		return "bytes"
	case parser.TypeMessage:
		return "struct"
	case parser.TypeEnum:
		return "enum"
	default:
		return "unknown"
	}
}
