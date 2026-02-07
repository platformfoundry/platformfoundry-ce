// Package cli provides the public CLI API for Platform Foundry.
// This package re-exports the internal CLI for use by enterprise editions.
package cli

import (
	internalcli "github.com/platformfoundry/pf-ce/internal/cli"
)

// Execute runs the Platform Foundry CLI.
// This is the main entry point for the application.
func Execute() {
	internalcli.Execute()
}
