package mage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/smoxy-io/proto2fixed/pkg/api"
	"github.com/smoxy-io/proto2fixed/pkg/generator"
)

// Examples namespace for schema generation tasks
type Examples mg.Namespace

// Generate all schema outputs (JSON, Arduino, Go)
func (Examples) Generate() error {
	mg.Deps(Examples.JSON, Examples.Arduino, Examples.Go)

	return nil
}

// JSON generates JSON examples for firmware
func (Examples) JSON() error {
	protoFiles, pfErr := getExamplesProtoFiles()

	if pfErr != nil {
		return pfErr
	}

	outDir := getExamplesOutputDir()

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	opts := []api.Option{
		api.WithLanguage(generator.LangJSON),
		api.WithOutputDir(filepath.Join(outDir, "json")),
		api.WithDefaultImportPaths(),
	}

	fmt.Println("Generating JSON examples from examples/*.proto files...")

	return api.Process(protoFiles, opts...)

	//for _, protoFile := range protoFiles {
	//	fmt.Printf("Processing %s...\n", protoFile)
	//
	//	p := parser.NewParser(".", filepath.Join(".", "examples"))
	//
	//	schema, pErr := p.Parse(protoFile)
	//
	//	if pErr != nil {
	//		fmt.Println("Parse failed")
	//		return pErr
	//	}
	//
	//	validator := analyzer.NewValidator()
	//
	//	vResult, vErr := validator.Validate(schema)
	//
	//	if vErr != nil {
	//		fmt.Println("Validation error", vErr)
	//		return vErr
	//	}
	//
	//	// Print validation warnings
	//	if len(vResult.Warnings) > 0 {
	//		for _, warning := range vResult.Warnings {
	//			fmt.Println(warning.String())
	//		}
	//	}
	//
	//	// Print validation errors and exit if any
	//	if vResult.HasErrors() {
	//		for _, err := range vResult.Errors {
	//			fmt.Println(err.Error())
	//		}
	//
	//		return errors.New("validation failed")
	//	}
	//
	//	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	//
	//	if err := layoutAnalyzer.Analyze(schema); err != nil {
	//		fmt.Println("Layout analysis failed")
	//		return err
	//	}
	//
	//	layouts := layoutAnalyzer.GetAllLayouts()
	//
	//	output := generator.OutputFile(generator.LangJSON, schema, getExamplesOutputDirParts()...)
	//
	//	fmt.Printf("Generating %s...\n", output)
	//
	//	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
	//		return err
	//	}
	//
	//	gen := generator.NewJSONGenerator()
	//
	//	code, gErr := gen.Generate(schema, layouts)
	//
	//	if gErr != nil {
	//		fmt.Println("Failed to generate code")
	//		return gErr
	//	}
	//
	//	fmt.Printf("Generated code length: %d\n", len(code))
	//	fmt.Printf("First 500 chars:\n%s\n", code[:min(500, len(code))])
	//
	//	if err := os.WriteFile(output, []byte(code), 0644); err != nil {
	//		return err
	//	}
	//}
	//
	//return nil
}

// Arduino generates C++ headers for ESP32
func (Examples) Arduino() error {
	protoFiles, pfErr := getExamplesProtoFiles()

	if pfErr != nil {
		return pfErr
	}

	outDir := getExamplesOutputDir()

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	opts := []api.Option{
		api.WithLanguage(generator.LangArduino),
		api.WithOutputDir(filepath.Join(outDir, "arduino")),
		api.WithDefaultImportPaths(),
	}

	fmt.Println("Generating Arduino examples from examples/*.proto files...")

	return api.Process(protoFiles, opts...)

	//for _, protoFile := range protoFiles {
	//	fmt.Printf("Processing %s...\n", protoFile)
	//
	//	p := parser.NewParser(".", filepath.Join(".", "examples"))
	//
	//	schema, pErr := p.Parse(protoFile)
	//
	//	if pErr != nil {
	//		fmt.Println("Parse failed")
	//		return pErr
	//	}
	//
	//	validator := analyzer.NewValidator()
	//
	//	vResult, vErr := validator.Validate(schema)
	//
	//	if vErr != nil {
	//		fmt.Println("Validation error", vErr)
	//		return vErr
	//	}
	//
	//	// Print validation warnings
	//	if len(vResult.Warnings) > 0 {
	//		for _, warning := range vResult.Warnings {
	//			fmt.Println(warning.String())
	//		}
	//	}
	//
	//	// Print validation errors and exit if any
	//	if vResult.HasErrors() {
	//		for _, err := range vResult.Errors {
	//			fmt.Println(err.Error())
	//		}
	//
	//		return errors.New("validation failed")
	//	}
	//
	//	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	//
	//	if err := layoutAnalyzer.Analyze(schema); err != nil {
	//		fmt.Println("Layout analysis failed")
	//		return err
	//	}
	//
	//	layouts := layoutAnalyzer.GetAllLayouts()
	//
	//	output := generator.OutputFile(generator.LangArduino, schema, outputDirParts...)
	//
	//	fmt.Printf("Generating %s...\n", output)
	//
	//	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
	//		return err
	//	}
	//
	//	gen := generator.NewArduinoGenerator()
	//
	//	code, gErr := gen.Generate(schema, layouts)
	//
	//	if gErr != nil {
	//		fmt.Println("Failed to generate code")
	//		return gErr
	//	}
	//
	//	fmt.Printf("Generated code length: %d\n", len(code))
	//	fmt.Printf("First 500 chars:\n%s\n", code[:min(500, len(code))])
	//
	//	if err := os.WriteFile(output, []byte(code), 0644); err != nil {
	//		return err
	//	}
	//}
	//
	//return nil
}

// Go generates Go decoder/encoder code
func (Examples) Go() error {
	protoFiles, pfErr := getExamplesProtoFiles()

	if pfErr != nil {
		return pfErr
	}

	outDir := getExamplesOutputDir()

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	opts := []api.Option{
		api.WithLanguage(generator.LangGo),
		api.WithOutputDir(filepath.Join(outDir, "go")),
		api.WithDefaultImportPaths(),
	}

	fmt.Println("Generating Go examples from examples/*.proto files...")

	return api.Process(protoFiles, opts...)

	//for _, protoFile := range protoFiles {
	//	fmt.Printf("Parsing %s...\n", protoFile)
	//
	//	p := parser.NewParser(".", filepath.Join(".", "examples"))
	//
	//	schema, pErr := p.Parse(protoFile)
	//
	//	if pErr != nil {
	//		fmt.Println("Parse failed")
	//		return pErr
	//	}
	//
	//	validator := analyzer.NewValidator()
	//
	//	vResult, vErr := validator.Validate(schema)
	//
	//	if vErr != nil {
	//		fmt.Println("Validation error", vErr)
	//		return vErr
	//	}
	//
	//	// Print validation warnings
	//	if len(vResult.Warnings) > 0 {
	//		for _, warning := range vResult.Warnings {
	//			fmt.Println(warning.String())
	//		}
	//	}
	//
	//	// Print validation errors and exit if any
	//	if vResult.HasErrors() {
	//		for _, err := range vResult.Errors {
	//			fmt.Println(err.Error())
	//		}
	//
	//		return errors.New("validation failed")
	//	}
	//
	//	layoutAnalyzer := analyzer.NewLayoutAnalyzer()
	//
	//	if err := layoutAnalyzer.Analyze(schema); err != nil {
	//		fmt.Println("Layout analysis failed")
	//		return err
	//	}
	//
	//	layouts := layoutAnalyzer.GetAllLayouts()
	//
	//	output := generator.OutputFile(generator.LangGo, schema, outputDirParts...)
	//
	//	fmt.Printf("Generating %s...\n", output)
	//
	//	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
	//		return err
	//	}
	//
	//	gen := generator.NewGoGenerator()
	//
	//	code, gErr := gen.Generate(schema, layouts)
	//
	//	if gErr != nil {
	//		fmt.Println("Failed to generate code")
	//		return gErr
	//	}
	//
	//	fmt.Printf("Generated code length: %d\n", len(code))
	//	fmt.Printf("First 500 chars:\n%s\n", code[:min(500, len(code))])
	//
	//	if err := os.WriteFile(output, []byte(code), 0644); err != nil {
	//		return err
	//	}
	//}
	//
	//return nil
}

// Clean removes all generated files
func (Examples) Clean() error {
	fmt.Println("Cleaning generated example schema files...")

	return sh.Rm(getExamplesOutputDir())
}

// Validate validates all proto examples without generating code
func (Examples) Validate() error {
	// Ensure proto2fixed is built
	if err := Build(); err != nil {
		return err
	}

	protoFiles, err := getExamplesProtoFiles()
	if err != nil {
		return err
	}

	hasErrors := false
	for _, protoFile := range protoFiles {
		fmt.Printf("Validating %s...\n", protoFile)
		if err := sh.RunV("./proto2fixed", "--validate", protoFile); err != nil {
			// Extract just the filename for cleaner output
			fileName := filepath.Base(protoFile)
			fmt.Printf("✗ %s failed validation\n", fileName)
			hasErrors = true
		} else {
			fileName := filepath.Base(protoFile)
			fmt.Printf("✓ %s passed validation\n", fileName)
		}
	}

	if hasErrors {
		return fmt.Errorf("some examples failed validation")
	}

	fmt.Println("\n✓ All examples valid!")
	return nil
}

// List lists all proto files in the examples directory
func (Examples) List() error {
	protoFiles, err := getExamplesProtoFiles()
	if err != nil {
		return err
	}

	fmt.Println("Proto schema files:")
	for _, file := range protoFiles {
		// Get file size
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// Count lines
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		lines := strings.Count(string(content), "\n") + 1

		fmt.Printf("  - %s (%d bytes, %d lines)\n", filepath.Base(file), info.Size(), lines)
	}

	return nil
}

func getExamplesProtoFiles() ([]string, error) {
	return filepath.Glob(filepath.Join("examples", "*.proto"))
}

func getExamplesOutputDirParts() []string {
	return []string{"examples", "generated"}
}

func getExamplesOutputDir() string {
	return filepath.Join(getExamplesOutputDirParts()...)
}
