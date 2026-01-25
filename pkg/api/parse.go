package api

import "github.com/smoxy-io/proto2fixed/pkg/parser"

func Parse(protoFile string, importPaths ...string) (*parser.Schema, error) {
	p := parser.NewParser(importPaths...)

	return p.Parse(protoFile)
}
