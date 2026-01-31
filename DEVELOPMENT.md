# Platform Foundry Development Guide

## Repository Structure

Platform Foundry uses two independent repositories:

```
your-workspace/
├── platformfoundry-ce/     # Community Edition (Apache 2.0)
├── platformfoundry-ee/     # Enterprise Edition (private)
└── go.work                 # Local workspace file (not tracked)
```

## Initial Setup

### Prerequisites

- Go 1.24.6 or later
- Git

### Clone Repositories

```bash
# Create workspace directory
mkdir pf-workspace && cd pf-workspace

# Clone CE
git clone https://github.com/platformfoundry/platformfoundry-ce.git

# Clone EE (if you have access)
git clone https://github.com/platformfoundry/platformfoundry-ee.git
```

### Setup Go Workspace (for working on both CE and EE)

Create `go.work` in your workspace directory:

```bash
go work init
go work use ./platformfoundry-ce ./platformfoundry-ee
```

This allows EE to use local CE changes during development.

## Building

### Using build.go (CE)

```bash
cd ce/platformfoundry-ce

# Build
go run build.go build

# Run tests
go run build.go test

# Cross-compile
go run build.go cross

# Create snapshot release
go run build.go snapshot

# Show all commands
go run build.go help
```

## Development Workflow

### Working on CE Only

If you're only working on CE, you don't need the workspace setup:

```bash
cd platformfoundry-ce
go run build.go build
go run build.go test
```

### Working on Both CE and EE

The `go.work` file allows Go to resolve CE imports locally instead of fetching from the module proxy:

```bash
cd pf-workspace

# Edit CE code
vim platformfoundry-ce/pkg/types/platform.go

# EE automatically uses local CE changes
cd platformfoundry-ee
go build ./...  # Uses local CE
```

### Before Committing EE Changes

If you made CE changes that EE depends on:

1. Commit and push CE changes first
2. Tag CE if it's a release: `git tag v1.x.x`
3. Update EE's `go.mod` to reference the CE version:
   ```bash
   cd platformfoundry-ee
   go get github.com/platformfoundry/platformfoundry-ce@v1.x.x
   go mod tidy
   ```

## Project Structure (CE)

```
platformfoundry-ce/
├── cmd/pf/                 # CLI entry point
├── pkg/                    # Public API
│   ├── contracts/          # Interface contracts (PPI, PSI, etc.)
│   ├── sdk/                # Plugin SDK
│   ├── services/           # Service container (DI)
│   ├── crd/                # Custom Resource Definitions
│   ├── types/              # Core data types
│   ├── cli/                # CLI interface
│   ├── extensions/         # Extension points
│   └── plugin/             # Plugin interface
├── internal/               # Private implementation
│   ├── cli/                # CLI commands
│   ├── engine/             # Core orchestration
│   ├── plugins/            # Built-in plugins
│   ├── state/              # State backends
│   ├── secrets/            # Secrets engines
│   └── ...
├── build.go                # Build script
├── .goreleaser.yaml        # Release configuration
└── .github/workflows/      # CI/CD pipelines
```

## Running Tests

```bash
cd platformfoundry-ce

# All tests with coverage
go run build.go test

# Specific package
go test -v ./pkg/types/...

# Integration tests
go test -v ./tests/integration/...
```

## Linting

```bash
# Using build.go (falls back to go vet if golangci-lint not installed)
go run build.go lint

# Using golangci-lint directly
golangci-lint run ./...
```

## Creating a Release

Releases are automated via GitHub Actions when a tag is pushed:

```bash
# Ensure you're on main and up to date
git checkout main
git pull

# Create and push tag
git tag v1.2.0
git push origin v1.2.0
```

For local testing of the release process:

```bash
go run build.go snapshot
# Output in ./dist/
```

## Troubleshooting

### "package not found" errors with go.work

```bash
# Sync the workspace
go work sync

# Or regenerate go.work
rm go.work
go work init
go work use ./platformfoundry-ce ./platformfoundry-ee
```

### EE not seeing CE changes

Ensure `go.work` exists at your workspace root and includes both modules.

### Import cycle errors

The `pkg/` packages should not import from `internal/`. Check your import paths.
