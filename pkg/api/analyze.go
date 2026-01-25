package api

import (
	"github.com/smoxy-io/proto2fixed/pkg/analyzer"
	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

func Validate(schema *parser.Schema) (*analyzer.ValidationResult, *analyzer.LayoutAnalyzer, error) {
	validator := analyzer.NewValidator()

	result, rErr := validator.Validate(schema)

	if rErr != nil {
		return nil, nil, rErr
	}

	return result, validator.GetAnalyzer(), nil
}

func Analyze(schema *parser.Schema) (*analyzer.LayoutAnalyzer, error) {
	layoutAnalyzer := analyzer.NewLayoutAnalyzer()

	if err := layoutAnalyzer.Analyze(schema); err != nil {
		return nil, err
	}

	return layoutAnalyzer, nil
}
