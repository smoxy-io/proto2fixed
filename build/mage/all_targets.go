package mage

import (
	"fmt"
	OS "os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/sh"
)

// Default target runs build
var Default = Build

func Lint() error {
	fmt.Println("running go lint checks")

	out, err := sh.OutCmd("go", "fmt", "./...")()

	if err == nil {
		fmt.Println(out)
	}

	return err
}

func Fmt() error {
	return Lint()
}

func Test() error {
	fmt.Println("running go test with coverage")

	out, err := sh.OutCmd("go", "test", "./...", "-cover")()

	fmt.Println(out)

	return err
}

// Bench runs all benchmarks
func Bench() error {
	fmt.Println("running benchmarks")

	out, err := sh.OutCmd("go", "test", "-bench=.", "-benchmem", "-run=^$", "./pkg/...", "./cmd/...")()

	fmt.Println(out)

	return err
}

// BenchShort runs quick benchmarks (100ms benchtime)
func BenchShort() error {
	fmt.Println("running quick benchmarks")

	out, err := sh.OutCmd("go", "test", "-bench=.", "-benchmem", "-benchtime=100ms", "-run=^$", "./pkg/...", "./cmd/...")()

	fmt.Println(out)

	return err
}

// BenchRealWorld runs only real-world benchmarks
func BenchRealWorld() error {
	fmt.Println("running real-world benchmarks")

	out, err := sh.OutCmd("go", "test", "-bench=RealWorld", "-benchmem", "-run=^$", "./pkg/...")()

	fmt.Println(out)

	return err
}

func Tidy() error {
	fmt.Println("running go mod tidy")

	out, err := sh.OutCmd("go", "mod", "tidy")()

	fmt.Println(out)

	return err
}

// Build builds the binary
func Build() error {
	var debug bool

	version := getVersion()

	if version == "dev" {
		debug = true
	}

	return buildBinary(binName, version, debug)
}

// Compress create zip archives for each platform
func Compress() error {
	version := getVersion()

	return compressBinary(binName, version)
}

// ShaSum create SHA256 sums
func ShaSum() error {
	version := getVersion()

	return getBinarySha256Sum(binName, version)
}

// Release build the binary and create all release artifacts
func Release() error {
	fmt.Println("building release artifacts")

	if buildLocal() {
		fmt.Println("WARNING: only releasing the current OS/arch")
	}

	// ensure that Build() uses the release version
	OS.Setenv("VERSION", "release")

	// build the binary
	if err := Build(); err != nil {
		return err
	}

	// create SHA256 sums
	if err := ShaSum(); err != nil {
		return err
	}

	// create zip archives
	if err := Compress(); err != nil {
		return err
	}

	return nil
}

func Clean() error {
	fmt.Println("cleaning up any prior builds")

	out, err := sh.OutCmd("rm", "-rf", "build/bin")()

	fmt.Println(out)

	return err
}

// Install installs the proto2fixed binary to $GOPATH/bin
func Install() error {
	fmt.Println("Installing proto2fixed...")

	if err := BuildLocal(); err != nil {
		return err
	}

	binPath := filepath.Join(buildRoot, strings.Join([]string{runtime.GOOS, runtime.GOARCH, getVersion()}, "_"), binName)

	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	return sh.Copy(filepath.Join(OS.Getenv("GOPATH"), "bin", binName), binPath)
}

// BuildLocal builds for the current OS/arch
func BuildLocal() error {
	fmt.Println("Building binary for local OS/arch...")

	// ensure that Build() uses the current OS/arch
	OS.Setenv(EnvVarBuildLocal, "true")

	return Build()
}
