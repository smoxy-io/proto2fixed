package mage

import (
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"io"
	OS "os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/magefile/mage/sh"
)

const (
	SHA256_CHECKSUM_FILE_PERMS     = 0644
	SHA256_CHECKSUM_FILE_EXTENSION = ".sha256.checksum"

	BUILD_OS_ARCHES_ARM_PATTERN = "^arm(64)?v\\d+$"
)

const (
	EnvVarBuildLocal = "BUILD_LOCAL"
)

const (
	binName = "proto2fixed"
)

var (
	//go:embed VERSION
	releaseVersion string

	buildOsArchs []string = []string{
		"linux/arm64",
		"linux/amd64",
		"darwin/arm64",
		"darwin/amd64",
		"windows/amd64",
	}

	// builds that require a specific version of GOARM to be set
	// (use "v" to separate the arch from the GOARM value)
	buildOsArchsArm []string = []string{
		//"linux/armv5",
		//"linux/armv6",
	}

	buildRoot string = filepath.Join("build", "bin")
)

func buildBinary(target string, version string, debug bool) error {
	fmt.Println("building binary: " + target)

	// ensure that the build root exists
	if err := OS.MkdirAll(buildRoot, 0755); err != nil {
		return err
	}

	osArchs := append([]string{}, buildOsArchs...)
	osArchsArm := append([]string{}, buildOsArchsArm...)

	if buildLocal() {
		osArchs = []string{runtime.GOOS + "/" + runtime.GOARCH}
		osArchsArm = []string{}
	}

	// build the golang binary
	cmdArgs := []string{
		"-output", filepath.Join(buildRoot, "{{.OS}}_{{.Arch}}_"+version, "{{.Dir}}"),
		"-ldflags", "-X 'main.VERSION=" + version + "'",
		"-osarch", strings.Join(osArchs, " "),
	}

	if debug {
		cmdArgs = append(cmdArgs, "-gcflags", "all=-N -l")
	}

	cmdArgs = append(cmdArgs, "."+string(OS.PathSeparator)+filepath.Join("cmd", target))

	out, err := sh.OutCmd("gox", cmdArgs...)()

	fmt.Println(out)

	if err != nil {
		return err
	}

	if len(osArchsArm) == 0 {
		return nil
	}

	fmt.Println("building GOARM specific arm binaries: " + target)

	for i := 0; i < len(osArchsArm); i++ {
		os, arch := splitOsArch(osArchsArm[i])

		if os == "" || arch == "" {
			fmt.Println("skipping " + osArchsArm[i])
			continue
		}

		if match, e := regexp.Match(BUILD_OS_ARCHES_ARM_PATTERN, []byte(arch)); !match || e != nil {
			fmt.Println("skipping " + osArchsArm[i])
			continue
		}

		parts := strings.Split(arch, "v")

		if len(parts) != 2 {
			fmt.Println("skipping " + osArchsArm[i])
			continue
		}

		fmt.Println("building " + osArchsArm[i])

		args := []string{
			"-output", filepath.Join(buildRoot, "{{.OS}}_{{.Arch}}v"+parts[1]+"_"+version, "{{.Dir}}"),
			"-ldflags", "-X 'main.VERSION=" + version + "'",
			"-osarch", strings.Join([]string{os, parts[0]}, "/"),
		}

		if debug {
			args = append(args, "-gcflags", "all=-N -l")
		}

		args = append(args, "."+string(OS.PathSeparator)+filepath.Join("cmd", target))

		env := make(map[string]string)

		env["GOARM"] = parts[1]

		out, err = sh.OutputWith(env, "gox", args...)

		fmt.Println(out)

		if err != nil {
			return err
		}
	}

	return nil
}

func getBinarySha256Sum(target string, version string) error {
	fmt.Println("calculating Sha256 sums for binary: " + target)

	osArchs := append(buildOsArchs, buildOsArchsArm...)

	waitGroup := &sync.WaitGroup{}
	outputWg := &sync.WaitGroup{}
	outChan := make(chan string, len(osArchs))
	errChan := make(chan error, len(osArchs))
	mErr := &multierror.Error{}

	outputWg.Add(1)
	go func() {
		defer outputWg.Done()

		// exits when errChan is closed
		for err := range errChan {
			if err != nil {
				continue
			}

			mErr = multierror.Append(mErr, err)
		}
	}()

	outputWg.Add(1)
	go func() {
		defer outputWg.Done()

		// exits when outChan is closed
		for out := range outChan {
			fmt.Println(out)
		}
	}()

	for _, osarch := range osArchs {
		os, arch := splitOsArch(osarch)

		if os == "" || arch == "" {
			errChan <- errors.New("Cannot split osarch: " + osarch)
			continue
		}

		binaryName := filepath.Join(buildRoot, strings.Join([]string{os, arch, version}, "_"), target)
		shaFileName := filepath.Join(buildRoot, strings.Join([]string{target, version, os, arch}, "_")+SHA256_CHECKSUM_FILE_EXTENSION)

		if os == "windows" {
			binaryName = binaryName + ".exe"
		}

		waitGroup.Add(1)
		go func(fn string, sfn string) {
			defer waitGroup.Done()

			f, fErr := OS.Open(fn)

			if fErr != nil {
				errChan <- fErr
				return
			}

			defer f.Close()

			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				errChan <- err
				return
			}

			shaSum := fmt.Sprintf("%x", h.Sum(nil))

			if err := OS.WriteFile(sfn, []byte(shaSum), SHA256_CHECKSUM_FILE_PERMS); err != nil {
				errChan <- err
				return
			}

			outChan <- "Sha256 Sum for " + fn + ": " + shaSum

			//fmt.Printf("%x", h.Sum(nil))
		}(binaryName, shaFileName)
	}

	// wait for go routines to finish
	waitGroup.Wait()

	// close channels
	close(outChan)
	close(errChan)

	// wait for output go routine to finish
	outputWg.Wait()

	return mErr.ErrorOrNil()
}

func compressBinary(target string, version string) error {
	fmt.Println("compressing binary: " + target)

	osArchs := append(buildOsArchs, buildOsArchsArm...)

	waitGroup := &sync.WaitGroup{}
	outputWg := &sync.WaitGroup{}
	outChan := make(chan string, len(osArchs))
	errChan := make(chan error, len(osArchs))
	mErr := &multierror.Error{}

	outputWg.Add(1)
	go func() {
		defer outputWg.Done()

		// exits when errChan is closed
		for err := range errChan {
			if err != nil {
				continue
			}

			mErr = multierror.Append(mErr, err)
		}
	}()

	outputWg.Add(1)
	go func() {
		defer outputWg.Done()

		// exits when outChan is closed
		for out := range outChan {
			fmt.Println(out)
		}
	}()

	for _, osarch := range osArchs {
		os, arch := splitOsArch(osarch)

		if os == "" || arch == "" {
			errChan <- errors.New("Cannot split osarch: " + osarch)
			continue
		}

		zipFile := filepath.Join(buildRoot, strings.Join([]string{target, version, os, arch}, "_")+".zip")
		zipContent := filepath.Join(buildRoot, strings.Join([]string{os, arch, version}, "_"), target)

		if os == "windows" {
			zipContent = zipContent + ".exe"
		}

		waitGroup.Add(1)
		go func(zf string, zc string) {
			defer waitGroup.Done()

			cmdArgs := []string{
				"-j",
				zf,
				zc,
			}

			out, zErr := sh.OutCmd("zip", cmdArgs...)()

			if zErr != nil {
				errChan <- zErr
				return
			}

			outChan <- out
		}(zipFile, zipContent)
	}

	// wait for go routines to finish
	waitGroup.Wait()

	// close channels
	close(outChan)
	close(errChan)

	// wait for output go routines to finish
	outputWg.Wait()

	return mErr.ErrorOrNil()
}

func splitOsArch(osarch string) (os string, arch string) {
	osarchParts := strings.Split(osarch, "/")

	return osarchParts[0], osarchParts[1]
}

func getVersion() string {
	version := OS.Getenv("VERSION")

	if version == "" {
		version = "dev"
	} else if version == "release" {
		version = releaseVersion
	}

	return version
}

func buildLocal() bool {
	bl, _ := strconv.ParseBool(OS.Getenv(EnvVarBuildLocal))

	return bl
}
