package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/smoxy-io/proto2fixed/pkg/api"
	"github.com/smoxy-io/proto2fixed/pkg/generator"
)

var (
	// set at build time using -ldflags=" -X 'main.VERSION=$version'"
	//   - edit build/mage/VERSION to set the version when building
	VERSION = "dev"
)

func main() {
	// Define flags
	lang := flag.String("lang", "", "Output language (json|arduino|go)")
	output := flag.String("output", "", "Output directory (default: stdout)")
	validate := flag.Bool("validate", false, "Validate schema only (no code generation)")
	version := flag.Bool("version", false, "Show version information")
	importPathList := flag.String("import-paths", "", "OS specific path-list-separator separated list of import paths (linux: colon-separated, windows: semicolon-separated)")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Printf("proto2fixed version %s\n", VERSION)
		os.Exit(0)
	}

	// Handle help flag
	if *help {
		printHelp()
		os.Exit(0)
	}

	// Get input file
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Error: no input files specified\n\n")
		printHelp()
		os.Exit(1)
	}

	inputFiles := flag.Args()

	// Validate required flags
	if !*validate && *lang == "" {
		fmt.Fprintf(os.Stderr, "Error: --lang flag is required (json|arduino|go)\n\n")
		printHelp()
		os.Exit(1)
	}

	// Process the proto files
	var importPaths []string

	if *importPathList != "" {
		importPaths = strings.Split(*importPathList, string(os.PathListSeparator))
	}

	processOptions := []api.Option{
		api.WithValidateOnly(*validate),
	}

	if *lang != "" {
		processOptions = append(processOptions, api.WithLanguage(generator.Language(*lang)))
	}

	if *output != "" {
		processOptions = append(processOptions, api.WithOutputDir(*output))
	}

	if len(importPaths) != 0 {
		processOptions = append(processOptions, api.WithImportPaths(importPaths...))
	}

	if err := api.Process(inputFiles, processOptions...); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("proto2fixed - Protocol Buffer to Fixed Binary Compiler")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  proto2fixed [flags] <input.proto> [<input.proto>] ...")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --import-paths=<pathList>    OS specific path-list-separator separated")
	fmt.Println("                               list of import paths (linux: colon-separated,")
	fmt.Println("                               windows: semicolon-separated)")
	fmt.Println("  --lang=<target>              Output language (json|arduino|go)")
	fmt.Println("  --output=<dir>               Output directory (default: stdout)")
	fmt.Println("  --validate                   Validate schema only (no code generation)")
	fmt.Println("  --version                    Show version information")
	fmt.Println("  --help                       Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  proto2fixed --lang=json status.proto")
	fmt.Println("  proto2fixed --lang=arduino --output=status.h status.proto")
	fmt.Println("  proto2fixed --lang=go --package=protocol status.proto")
	fmt.Println("  proto2fixed --validate status.proto")
	fmt.Println()
}
