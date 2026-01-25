package api

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/smoxy-io/proto2fixed/pkg/generator"
)

type processProtoFileResp struct {
	Err           error
	ProtoFile     string
	OutFile       string
	Code          string
	ValidationMsg string
}

type processProtoFileReq struct {
	Lang         generator.Language
	ProtoFile    string
	OutDir       string
	ValidateOnly bool
	ImportPaths  []string
}

// Process analyzes and generates code for a set of proto files (uses concurrency)
func Process(protoFiles []string, opts ...Option) error {
	if len(protoFiles) == 0 {
		return fmt.Errorf("no proto files specified")
	}

	options := NewOptions(opts...)

	if !options.ValidateOnly && !options.Lang.IsValid() {
		return fmt.Errorf("invalid language: %s", options.Lang)
	}

	concurrent := int(math.Min(float64(runtime.NumCPU()), float64(len(protoFiles))))

	wg := &sync.WaitGroup{}
	workerWg := &sync.WaitGroup{}
	outChan := make(chan *processProtoFileResp, concurrent)
	inChan := make(chan *processProtoFileReq, concurrent)
	mErr := &multierror.Error{}

	// processing function
	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, protoFile := range protoFiles {
			req := &processProtoFileReq{
				Lang:         options.Lang,
				ProtoFile:    protoFile,
				OutDir:       options.OutDir,
				ValidateOnly: options.ValidateOnly,
				ImportPaths:  options.ImportPaths,
			}

			// enforcing concurrency
			inChan <- req

			// spawn worker
			workerWg.Add(1)
			go func() {
				defer workerWg.Done()

				processProtoFile(req, outChan)

				// allow another request to be processed
				<-inChan
			}()
		}

		// no more requests
		close(inChan)

		// wait for all workers to finish
		workerWg.Wait()

		// no more responses
		close(outChan)
	}()

	// reporting function
	wg.Add(1)
	go func() {
		defer wg.Done()

		for resp := range outChan {
			if resp.Err != nil {
				mErr = multierror.Append(mErr, fmt.Errorf("Error processing %s: %v\n    %s", resp.ProtoFile, resp.Err, resp.ValidationMsg))

				continue
			}

			if options.ValidateOnly {
				fmt.Printf("✓ Schema validation passed for %s\n", resp.ProtoFile)
			} else {
				fmt.Printf("✓ %s -> %s\n", resp.ProtoFile, resp.OutFile)
			}

			if resp.ValidationMsg != "" {
				_, _ = fmt.Fprintf(os.Stderr, "%s", resp.ValidationMsg)
			}

			if options.OutDir != "" || options.ValidateOnly {
				continue
			}

			// print to stdout
			fmt.Println(resp.Code)
		}
	}()

	// wait for processing to finish
	wg.Wait()

	return mErr.ErrorOrNil()
}

func processProtoFile(req *processProtoFileReq, out chan<- *processProtoFileResp) {
	resp := &processProtoFileResp{
		ProtoFile: req.ProtoFile,
	}

	schema, sErr := Parse(req.ProtoFile, req.ImportPaths...)

	if sErr != nil {
		resp.Err = fmt.Errorf("parse error: %v", sErr)
		out <- resp
		return
	}

	result, analyzer, rErr := Validate(schema)

	if rErr != nil {
		resp.Err = fmt.Errorf("validation error: %v", rErr)
		out <- resp
		return
	}

	if result.HasWarnings() {
		for _, warning := range result.Warnings {
			resp.ValidationMsg += fmt.Sprintf("%s\n", warning.String())
		}
	}

	if result.HasErrors() {
		for _, err := range result.Errors {
			resp.ValidationMsg += fmt.Sprintf("%s\n", err.Error())
		}

		resp.Err = fmt.Errorf("validation failed")
		out <- resp

		return
	}

	if req.ValidateOnly {
		out <- resp
		return
	}

	code, cErr := GenerateLang(req.Lang, schema, analyzer.GetAllLayouts())

	if cErr != nil {
		resp.Err = fmt.Errorf("generation error: %v", cErr)
		out <- resp
		return
	}

	resp.Code = code

	if req.OutDir == "" {
		resp.OutFile = "stdout"
		out <- resp
		return
	}

	resp.OutFile = generator.OutputFile(req.Lang.String(), schema, req.OutDir)

	if err := os.MkdirAll(filepath.Dir(resp.OutFile), 0755); err != nil {
		resp.Err = err
		out <- resp
		return
	}

	if err := os.WriteFile(resp.OutFile, []byte(code), 0644); err != nil {
		resp.Err = err
		out <- resp
		return
	}

	out <- resp
}
