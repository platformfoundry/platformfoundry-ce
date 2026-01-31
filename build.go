//go:build ignore
// +build ignore

// Build script for Platform Foundry
// Usage: go run build.go [command]
// Commands: build, install, clean, test, run, snapshot, cross, version, lint, help

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	binaryName = "pf"
	buildDir   = "bin"
	mainPath   = "./cmd/pf"
	modulePath = "github.com/platformfoundry/platformfoundry-ce"
)

// Build targets for cross-compilation
var targets = []struct {
	GOOS   string
	GOARCH string
}{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"windows", "amd64"},
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]

	switch command {
	case "build":
		if err := build(); err != nil {
			log.Fatal(err)
		}
	case "install":
		if err := install(); err != nil {
			log.Fatal(err)
		}
	case "clean":
		if err := clean(); err != nil {
			log.Fatal(err)
		}
	case "test":
		if err := test(); err != nil {
			log.Fatal(err)
		}
	case "run":
		if err := run(); err != nil {
			log.Fatal(err)
		}
	case "snapshot":
		if err := snapshot(); err != nil {
			log.Fatal(err)
		}
	case "cross":
		if err := crossCompile(); err != nil {
			log.Fatal(err)
		}
	case "version":
		showVersion()
	case "lint":
		if err := lint(); err != nil {
			log.Fatal(err)
		}
	case "help":
		printHelp()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printHelp()
		os.Exit(1)
	}
}

// getLdflags returns the ldflags for version injection
func getLdflags() string {
	version := getVersion()
	commit := getCommit()
	date := time.Now().UTC().Format(time.RFC3339)

	versionPkg := modulePath + "/internal/version"
	ldflags := []string{
		"-s", "-w",
		fmt.Sprintf("-X %s.Version=%s", versionPkg, version),
		fmt.Sprintf("-X %s.Commit=%s", versionPkg, commit),
		fmt.Sprintf("-X %s.Date=%s", versionPkg, date),
		fmt.Sprintf("-X %s.BuiltBy=build.go", versionPkg),
	}
	return strings.Join(ldflags, " ")
}

// getVersion returns the version from git tag or "dev"
func getVersion() string {
	cmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	output, err := cmd.Output()
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(output))
}

// getCommit returns the current git commit SHA
func getCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "none"
	}
	return strings.TrimSpace(string(output))
}

func build() error {
	fmt.Println("Building", binaryName+"...")

	// Create build directory
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("error creating build directory: %w", err)
	}

	// Determine binary name with extension for Windows
	outputBinary := filepath.Join(buildDir, binaryName)
	if runtime.GOOS == "windows" {
		outputBinary += ".exe"
	}

	// Build command with ldflags
	ldflags := getLdflags()
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", outputBinary, mainPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Println("Build complete:", outputBinary)
	return nil
}

func install() error {
	fmt.Println("Installing dependencies...")

	// Download dependencies
	cmd := exec.Command("go", "mod", "download")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download dependencies: %w", err)
	}

	// Tidy dependencies
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy dependencies: %w", err)
	}

	fmt.Println("Dependencies installed")
	return nil
}

func clean() error {
	fmt.Println("Cleaning...")

	// Run go clean
	cmd := exec.Command("go", "clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Warning: go clean failed:", err)
	}

	// Remove build directory
	if err := os.RemoveAll(buildDir); err != nil {
		fmt.Println("Warning: Failed to remove build directory:", err)
	}

	// Remove dist directory (goreleaser output)
	if err := os.RemoveAll("dist"); err != nil {
		fmt.Println("Warning: Failed to remove dist directory:", err)
	}

	fmt.Println("Clean complete")
	return nil
}

func test() error {
	fmt.Println("Running tests...")

	cmd := exec.Command("go", "test", "-v", "-race", "-coverprofile=coverage.out", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	fmt.Println("Tests passed")
	return nil
}

func run() error {
	// Build first
	if err := build(); err != nil {
		return err
	}

	fmt.Println("\nRunning", binaryName+"...")

	// Determine binary path
	binaryPath := filepath.Join(buildDir, binaryName)
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// Run the binary with any additional arguments
	args := []string{}
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run failed: %w", err)
	}

	return nil
}

func snapshot() error {
	fmt.Println("Creating snapshot release...")

	// Check if goreleaser is installed
	if _, err := exec.LookPath("goreleaser"); err != nil {
		return fmt.Errorf("goreleaser not found in PATH. Install with: go install github.com/goreleaser/goreleaser/v2@latest")
	}

	cmd := exec.Command("goreleaser", "release", "--snapshot", "--clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("snapshot failed: %w", err)
	}

	fmt.Println("\nSnapshot release created in ./dist/")
	return nil
}

func crossCompile() error {
	fmt.Println("Cross-compiling for all platforms...")

	// Create build directory
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("error creating build directory: %w", err)
	}

	ldflags := getLdflags()

	for _, target := range targets {
		binary := binaryName
		if target.GOOS == "windows" {
			binary += ".exe"
		}
		outputPath := filepath.Join(buildDir, fmt.Sprintf("%s_%s_%s", binaryName, target.GOOS, target.GOARCH), binary)

		// Create target directory
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("error creating target directory: %w", err)
		}

		fmt.Printf("Building %s/%s...\n", target.GOOS, target.GOARCH)

		cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", outputPath, mainPath)
		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			fmt.Sprintf("GOOS=%s", target.GOOS),
			fmt.Sprintf("GOARCH=%s", target.GOARCH),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("cross-compile failed for %s/%s: %w", target.GOOS, target.GOARCH, err)
		}
	}

	fmt.Println("\nCross-compilation complete. Binaries in ./bin/")
	return nil
}

func showVersion() {
	version := getVersion()
	commit := getCommit()
	date := time.Now().UTC().Format(time.RFC3339)

	fmt.Printf("Platform Foundry Build Script\n")
	fmt.Printf("  Version:     %s\n", version)
	fmt.Printf("  Commit:      %s\n", commit)
	fmt.Printf("  Build Date:  %s\n", date)
	fmt.Printf("  Go Version:  %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func lint() error {
	fmt.Println("Running linters...")

	// Check if golangci-lint is installed
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		fmt.Println("golangci-lint not found, running go vet instead...")
		cmd := exec.Command("go", "vet", "./...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cmd := exec.Command("golangci-lint", "run", "--timeout=5m", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	fmt.Println("Linting passed")
	return nil
}

func printHelp() {
	fmt.Println("Platform Foundry Build Script")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run build.go [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build      Build the binary with version info")
	fmt.Println("  install    Install dependencies")
	fmt.Println("  clean      Remove build artifacts")
	fmt.Println("  test       Run tests with coverage")
	fmt.Println("  run        Build and run the binary")
	fmt.Println("  snapshot   Create snapshot release (requires goreleaser)")
	fmt.Println("  cross      Cross-compile for all platforms")
	fmt.Println("  version    Show version information")
	fmt.Println("  lint       Run linters (golangci-lint or go vet)")
	fmt.Println("  help       Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run build.go build")
	fmt.Println("  go run build.go install")
	fmt.Println("  go run build.go run -- --version")
	fmt.Println("  go run build.go cross")
	fmt.Println("  go run build.go snapshot")
	fmt.Println()
	fmt.Println("Build Targets (cross):")
	for _, t := range targets {
		fmt.Printf("  - %s/%s\n", t.GOOS, t.GOARCH)
	}
}
