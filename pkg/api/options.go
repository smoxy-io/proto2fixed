package api

import "github.com/smoxy-io/proto2fixed/pkg/generator"

var (
	defaultImportPaths = []string{".", "proto2fixed"}
)

type Options struct {
	Lang         generator.Language
	OutDir       string
	ValidateOnly bool
	ImportPaths  []string
}

func NewOptions(opts ...Option) *Options {
	o := &Options{}

	copy(o.ImportPaths, defaultImportPaths)

	if len(opts) == 0 {
		return o
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

type Option = func(*Options)

func WithLanguage(lang generator.Language) Option {
	return func(opts *Options) {
		opts.Lang = lang
	}
}

func WithOutputDir(dir string) Option {
	return func(opts *Options) {
		opts.OutDir = dir
	}
}

func WithValidateOnly(validateOnly bool) Option {
	return func(opts *Options) {
		opts.ValidateOnly = validateOnly
	}
}

func WithImportPaths(path ...string) Option {
	return func(opts *Options) {
		opts.ImportPaths = path
	}
}

func WithDefaultImportPaths() Option {
	return WithImportPaths(defaultImportPaths...)
}

func AppendImportPaths(path ...string) Option {
	return func(opts *Options) {
		opts.ImportPaths = append(opts.ImportPaths, path...)
	}
}
