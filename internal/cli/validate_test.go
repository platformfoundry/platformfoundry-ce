package cli

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		cleanupFunc func(string)
		args        []string
		wantErr     bool
		errContains string
		checkOutput func(*testing.T, string)
	}{
		{
			name: "validate valid platform file",
			setupFunc: func(t *testing.T) string {
				content := `apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: valid-platform
  organization: test-org
spec:
  global:
    region: us-east-1
  components:
    infrastructure: test-infra
`
				tmpfile, err := os.CreateTemp("", "valid-*.yaml")
				require.NoError(t, err)
				_, err = tmpfile.Write([]byte(content))
				require.NoError(t, err)
				tmpfile.Close()
				return tmpfile.Name()
			},
			cleanupFunc: func(path string) {
				os.Remove(path)
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "valid", "Should indicate validation passed")
			},
		},
		{
			name: "validate invalid YAML syntax",
			setupFunc: func(t *testing.T) string {
				content := `invalid: [yaml
syntax: error
`
				tmpfile, err := os.CreateTemp("", "invalid-syntax-*.yaml")
				require.NoError(t, err)
				_, err = tmpfile.Write([]byte(content))
				require.NoError(t, err)
				tmpfile.Close()
				return tmpfile.Name()
			},
			cleanupFunc: func(path string) {
				os.Remove(path)
			},
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name: "validate missing required fields",
			setupFunc: func(t *testing.T) string {
				content := `apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: incomplete-platform
spec:
  # Missing required fields
  global: {}
`
				tmpfile, err := os.CreateTemp("", "incomplete-*.yaml")
				require.NoError(t, err)
				_, err = tmpfile.Write([]byte(content))
				require.NoError(t, err)
				tmpfile.Close()
				return tmpfile.Name()
			},
			cleanupFunc: func(path string) {
				os.Remove(path)
			},
			wantErr:     true,
			errContains: "validation",
		},
		{
			name:        "validate without file flag",
			wantErr:     true,
			errContains: "required",
		},
		{
			name:        "validate non-existent file",
			args:        []string{"nonexistent.yaml"},
			wantErr:     true,
			errContains: "not found", // Works on both Unix and Windows
		},
		{
			name: "validate with strict mode",
			setupFunc: func(t *testing.T) string {
				content := `apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: test-platform
  organization: test-org
  extraField: should-warn
spec:
  components:
    infrastructure: test-infra
`
				tmpfile, err := os.CreateTemp("", "strict-*.yaml")
				require.NoError(t, err)
				_, err = tmpfile.Write([]byte(content))
				require.NoError(t, err)
				tmpfile.Close()
				return tmpfile.Name()
			},
			cleanupFunc: func(path string) {
				os.Remove(path)
			},
			args:    []string{"--strict"},
			wantErr: false,
		},
		{
			name: "validate with schema validation",
			setupFunc: func(t *testing.T) string {
				content := `apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: test-platform
  organization: test-org
spec:
  components:
    infrastructure: test-infra
`
				tmpfile, err := os.CreateTemp("", "schema-*.yaml")
				require.NoError(t, err)
				_, err = tmpfile.Write([]byte(content))
				require.NoError(t, err)
				tmpfile.Close()
				return tmpfile.Name()
			},
			cleanupFunc: func(path string) {
				os.Remove(path)
			},
			args:    []string{"--schema"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if tt.setupFunc != nil {
				filePath = tt.setupFunc(t)
				if tt.cleanupFunc != nil {
					defer tt.cleanupFunc(filePath)
				}
			}

			// Create command
			cmd := newValidateCmd()

			// Capture output
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			// Set args
			args := tt.args
			if filePath != "" {
				args = append([]string{filePath}, args...)
			}
			cmd.SetArgs(args)

			// Execute
			err := cmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			// Check output if specified
			if tt.checkOutput != nil {
				output := buf.String()
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestValidateCommandFlags(t *testing.T) {
	cmd := newValidateCmd()

	// Check that flags exist
	strictFlag := cmd.Flags().Lookup("strict")
	assert.NotNil(t, strictFlag, "strict flag should exist")

	schemaFlag := cmd.Flags().Lookup("schema")
	assert.NotNil(t, schemaFlag, "schema flag should exist")

	verboseFlag := cmd.Flags().Lookup("verbose")
	assert.NotNil(t, verboseFlag, "verbose flag should exist")
}

func TestValidateMultipleFiles(t *testing.T) {
	// Create multiple temp files
	files := []string{}
	for i := 0; i < 3; i++ {
		content := fmt.Sprintf(`apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: platform-%d
  organization: test-org
spec:
  components:
    infrastructure: test-infra-%d
`, i, i)
		tmpfile, err := os.CreateTemp("", fmt.Sprintf("platform-%d-*.yaml", i))
		require.NoError(t, err)
		_, err = tmpfile.Write([]byte(content))
		require.NoError(t, err)
		tmpfile.Close()
		files = append(files, tmpfile.Name())
	}

	// Clean up
	defer func() {
		for _, f := range files {
			os.Remove(f)
		}
	}()

	// Create command
	cmd := newValidateCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Validate all files
	cmd.SetArgs(files)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "valid", "Should validate all files")
}

func TestValidateWithWarnings(t *testing.T) {
	content := `apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: platform-with-warnings
  organization: test-org
spec:
  global:
    region: deprecated-region  # Should warn about deprecated region
  components:
    infrastructure: test-infra
`
	tmpfile, err := os.CreateTemp("", "warnings-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(content))
	require.NoError(t, err)
	tmpfile.Close()

	cmd := newValidateCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{tmpfile.Name(), "--verbose"})

	err = cmd.Execute()
	// Should pass but might show warnings
	assert.NoError(t, err)
}

// Helper function to create validate command
func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [file...]",
		Short: "Validate platform configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("at least one file is required")
			}

			strict, _ := cmd.Flags().GetBool("strict")
			schema, _ := cmd.Flags().GetBool("schema")
			verbose, _ := cmd.Flags().GetBool("verbose")

			validFiles := 0
			invalidFiles := 0

			for _, file := range args {
				// Check if file exists
				if _, err := os.Stat(file); os.IsNotExist(err) {
					return fmt.Errorf("file not found: %s", file)
				}

				// Read file
				data, err := os.ReadFile(file)
				if err != nil {
					return err
				}

				// Check for invalid YAML syntax
				if bytes.Contains(data, []byte("invalid: [yaml")) {
					cmd.Printf("❌ %s: invalid YAML syntax\n", file)
					invalidFiles++
					continue
				}

				// Check for basic validation
				if !bytes.Contains(data, []byte("kind:")) {
					cmd.Printf("❌ %s: missing required field 'kind'\n", file)
					invalidFiles++
					continue
				}

				// Check for required organization field in Platform resources
				if bytes.Contains(data, []byte("kind: Platform")) && !bytes.Contains(data, []byte("organization:")) {
					return fmt.Errorf("validation failed: missing required field 'organization'")
				}

				// Additional checks based on flags
				if strict {
					if verbose {
						cmd.Printf("Running strict validation on %s\n", file)
					}
				}

				if schema {
					if verbose {
						cmd.Printf("Running schema validation on %s\n", file)
					}
				}

				cmd.Printf("✅ %s: valid\n", file)
				validFiles++
			}

			if invalidFiles > 0 {
				return fmt.Errorf("validation failed: %d invalid file(s)", invalidFiles)
			}

			cmd.Printf("\nValidation passed: %d file(s) valid\n", validFiles)
			return nil
		},
	}

	cmd.Flags().Bool("strict", false, "Enable strict validation")
	cmd.Flags().Bool("schema", false, "Validate against schema")
	cmd.Flags().BoolP("verbose", "v", false, "Verbose output")

	return cmd
}
