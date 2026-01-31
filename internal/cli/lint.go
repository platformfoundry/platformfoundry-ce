package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/platformfoundry/platformfoundry-ce/internal/lint"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint [files...]",
	Short: "Lint platform configuration files",
	Long: `Check platform configuration files for best practices and common issues.

The linter checks for:
- Required fields (apiVersion, kind, metadata)
- Naming conventions
- Security best practices
- Resource limits
- Hardcoded secrets
- And more...

Examples:
  pf lint platform.yaml              # Lint a single file
  pf lint *.yaml                     # Lint all YAML files
  pf lint -d ./configs               # Lint all files in directory
  pf lint --strict platform.yaml     # Fail on any warning`,
	Args: cobra.MinimumNArgs(0),
	RunE: runLint,
}

var (
	lintDirectory string
	lintStrict    bool
	lintFormat    string
	lintQuiet     bool
)

func init() {
	lintCmd.Flags().StringVarP(&lintDirectory, "directory", "d", "", "Lint all YAML files in directory")
	lintCmd.Flags().BoolVar(&lintStrict, "strict", false, "Treat warnings as errors")
	lintCmd.Flags().StringVar(&lintFormat, "format", "text", "Output format (text, json)")
	lintCmd.Flags().BoolVarP(&lintQuiet, "quiet", "q", false, "Only show files with issues")
}

func runLint(cmd *cobra.Command, args []string) error {
	linter := lint.New()

	var files []string

	// Collect files from arguments
	for _, arg := range args {
		matches, err := filepath.Glob(arg)
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", arg, err)
		}
		files = append(files, matches...)
	}

	// Collect files from directory
	if lintDirectory != "" {
		dirFiles, err := findYAMLFiles(lintDirectory)
		if err != nil {
			return err
		}
		files = append(files, dirFiles...)
	}

	if len(files) == 0 {
		fmt.Println("No files to lint.")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  pf lint <file.yaml>           # Lint specific file")
		fmt.Println("  pf lint -d <directory>        # Lint all YAML in directory")
		fmt.Println("  pf lint *.yaml                # Lint matching files")
		return nil
	}

	// Lint all files
	var results []*lint.Result
	totalErrors := 0
	totalWarnings := 0
	totalInfo := 0

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
			continue
		}

		var result *lint.Result
		if strings.Contains(string(content), "\n---") {
			result, err = linter.LintMultiple(content, file)
		} else {
			result, err = linter.Lint(content, file)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error linting %s: %v\n", file, err)
			continue
		}

		results = append(results, result)
		totalErrors += result.Summary.Errors
		totalWarnings += result.Summary.Warnings
		totalInfo += result.Summary.Info
	}

	// Output results
	for _, result := range results {
		if lintQuiet && len(result.Issues) == 0 {
			continue
		}
		fmt.Print(result.Format())
		fmt.Println()
	}

	// Summary
	fmt.Println("===================")
	fmt.Printf("Files checked: %d\n", len(files))
	fmt.Printf("Total issues:  %d errors, %d warnings, %d info\n", totalErrors, totalWarnings, totalInfo)

	// Exit code
	if totalErrors > 0 {
		fmt.Println("\n[FAILED] Lint check failed with errors")
		os.Exit(1)
	}

	if lintStrict && totalWarnings > 0 {
		fmt.Println("\n[FAILED] Lint check failed (strict mode)")
		os.Exit(1)
	}

	if totalErrors == 0 && totalWarnings == 0 {
		fmt.Println("\n[OK] All checks passed")
	} else {
		fmt.Println("\n[OK] No errors found")
	}

	return nil
}

func findYAMLFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
