// Package version provides build version information for Platform Foundry.
// These variables are set at build time using ldflags:
//
//	go build -ldflags "-X github.com/platformfoundry/pf-ce/internal/version.Version=v1.0.0"
package version

import (
	"fmt"
	"runtime"
)

// Build information. These are set via ldflags during build.
var (
	// Version is the semantic version of the build (e.g., "v1.0.0")
	Version = "dev"

	// Commit is the git commit SHA of the build
	Commit = "none"

	// Date is the build date in RFC3339 format
	Date = "unknown"

	// BuiltBy indicates who/what built this binary (e.g., "goreleaser")
	BuiltBy = "unknown"
)

// Info represents the complete version information
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuiltBy   string `json:"builtBy"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetInfo returns the complete version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuiltBy:   BuiltBy,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a human-readable version string
func String() string {
	return fmt.Sprintf("pf version %s (commit: %s, built: %s)", Version, Commit, Date)
}

// Short returns just the version number
func Short() string {
	return Version
}

// Full returns the full version string with all details
func Full() string {
	info := GetInfo()
	return fmt.Sprintf(
		"pf version %s\n"+
			"  Commit:     %s\n"+
			"  Built:      %s\n"+
			"  Built by:   %s\n"+
			"  Go version: %s\n"+
			"  OS/Arch:    %s/%s",
		info.Version,
		info.Commit,
		info.Date,
		info.BuiltBy,
		info.GoVersion,
		info.OS,
		info.Arch,
	)
}
