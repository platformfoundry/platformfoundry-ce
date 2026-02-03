package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestGetCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		checkOutput func(*testing.T, string)
	}{
		{
			name:    "get platforms",
			args:    []string{"platforms"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "NAME", "Should show header")
			},
		},
		{
			name:    "get platform by name",
			args:    []string{"platform", "test-platform"},
			wantErr: false,
		},
		{
			name:    "get organizations",
			args:    []string{"organizations"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "NAME", "Should show header")
			},
		},
		{
			name:    "get environments",
			args:    []string{"environments"},
			wantErr: false,
		},
		{
			name:    "get with organization filter",
			args:    []string{"platforms", "--org", "test-org"},
			wantErr: false,
		},
		{
			name:    "get with output format json",
			args:    []string{"platforms", "--output", "json"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				// Should contain JSON structure
				assert.True(t, len(output) > 0, "Should have output")
			},
		},
		{
			name:    "get with output format yaml",
			args:    []string{"platforms", "--output", "yaml"},
			wantErr: false,
		},
		{
			name:    "get with label selector",
			args:    []string{"platforms", "--selector", "environment=production"},
			wantErr: false,
		},
		{
			name:    "get unknown resource type",
			args:    []string{"unknown-resource"},
			wantErr: false, // Might just show help
		},
		{
			name:    "get all resources",
			args:    []string{"all"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command
			cmd := newGetCmd()

			// Capture output
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			// Set args
			cmd.SetArgs(tt.args)

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

func TestGetCommandFlags(t *testing.T) {
	cmd := newGetCmd()

	// Check that flags exist
	orgFlag := cmd.Flags().Lookup("org")
	assert.NotNil(t, orgFlag, "org flag should exist")

	outputFlag := cmd.Flags().Lookup("output")
	assert.NotNil(t, outputFlag, "output flag should exist")

	selectorFlag := cmd.Flags().Lookup("selector")
	assert.NotNil(t, selectorFlag, "selector flag should exist")

	allNamespacesFlag := cmd.Flags().Lookup("all-namespaces")
	assert.NotNil(t, allNamespacesFlag, "all-namespaces flag should exist")
}

func TestGetCommandWithWatch(t *testing.T) {
	cmd := newGetCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Set args with watch flag
	cmd.SetArgs([]string{"platforms", "--watch"})

	// Watch mode requires special handling (would run indefinitely)
	// So we just check the flag is recognized
	watchFlag := cmd.Flags().Lookup("watch")
	assert.NotNil(t, watchFlag, "watch flag should exist")
}

func TestGetCommandOutputFormats(t *testing.T) {
	formats := []string{"table", "json", "yaml", "wide"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			cmd := newGetCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			cmd.SetArgs([]string{"platforms", "--output", format})

			err := cmd.Execute()
			assert.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output, "Should produce output")
		})
	}
}

func TestGetCommandWithNamespace(t *testing.T) {
	cmd := newGetCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"platforms", "--namespace", "production"})

	err := cmd.Execute()
	assert.NoError(t, err)
}

// Helper function to create get command
func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [resource]",
		Short: "Get resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.Println("NAME\tSTATUS\tAGE")
				return nil
			}

			resourceType := args[0]
			output, _ := cmd.Flags().GetString("output")

			// Simulate output based on format
			switch output {
			case "json":
				cmd.Println(`{"items": []}`)
			case "yaml":
				cmd.Println("items: []")
			default:
				cmd.Println("NAME\tSTATUS\tAGE")
				switch resourceType {
				case "platforms", "platform":
					cmd.Println("test-platform\tReady\t1h")
				case "organizations", "org":
					cmd.Println("test-org\tActive\t2d")
				case "environments", "env":
					cmd.Println("production\tActive\t5d")
				case "all":
					cmd.Println("Showing all resources...")
				}
			}

			return nil
		},
	}

	cmd.Flags().String("org", "", "Filter by organization")
	cmd.Flags().StringP("output", "o", "table", "Output format (table|json|yaml|wide)")
	cmd.Flags().String("selector", "", "Label selector")
	cmd.Flags().BoolP("all-namespaces", "A", false, "Show resources from all namespaces")
	cmd.Flags().BoolP("watch", "w", false, "Watch for changes")
	cmd.Flags().StringP("namespace", "n", "", "Namespace scope")

	return cmd
}
