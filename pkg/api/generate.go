package api

import (
	"fmt"

	"github.com/smoxy-io/proto2fixed/pkg/analyzer"
	"github.com/smoxy-io/proto2fixed/pkg/generator"
	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

func Generate(gen generator.Generator, schema *parser.Schema, layouts map[string]*analyzer.MessageLayout) (string, error) {
	return gen.Generate(schema, layouts)
}

func GenerateJSON(schema *parser.Schema, layouts map[string]*analyzer.MessageLayout) (string, error) {
	return Generate(generator.NewJSONGenerator(), schema, layouts)
}

func GenerateArduino(schema *parser.Schema, layouts map[string]*analyzer.MessageLayout) (string, error) {
	return Generate(generator.NewArduinoGenerator(), schema, layouts)
}

func GenerateGo(schema *parser.Schema, layouts map[string]*analyzer.MessageLayout) (string, error) {
	return Generate(generator.NewGoGenerator(), schema, layouts)
}

func GenerateLang(lang generator.Language, schema *parser.Schema, layouts map[string]*analyzer.MessageLayout, opts ...generator.GeneratorOption) (string, error) {
	gen := generator.NewGenerator(lang, opts...)

	if gen == nil {
		return "", fmt.Errorf("invalid language: %s", lang)
	}

	return Generate(gen, schema, layouts)
}
