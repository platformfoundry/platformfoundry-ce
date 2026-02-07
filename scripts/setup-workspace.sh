#!/bin/bash
# Setup script for Platform Foundry development workspace
# Run this from the parent directory containing both ce/ and ee/ repos

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "Setting up Platform Foundry workspace at: $WORKSPACE_ROOT"

# Check directory structure
if [[ ! -d "$WORKSPACE_ROOT/ce/platformfoundry-ce" ]]; then
    echo "Error: Expected ce/platformfoundry-ce directory not found"
    echo "Please ensure your directory structure is:"
    echo "  pf-org/"
    echo "  ├── ce/platformfoundry-ce/"
    echo "  └── ee/platformfoundry-ee/"
    exit 1
fi

# Create go.work if it doesn't exist
if [[ ! -f "$WORKSPACE_ROOT/go.work" ]]; then
    echo "Creating go.work..."
    cat > "$WORKSPACE_ROOT/go.work" << 'EOF'
go 1.24.6

use (
    ./ce/platformfoundry-ce
    ./ee/platformfoundry-ee
)
EOF
    echo "Created go.work"
else
    echo "go.work already exists"
fi

# Create Makefile if it doesn't exist
if [[ ! -f "$WORKSPACE_ROOT/Makefile" ]]; then
    echo "Creating Makefile..."
    cat > "$WORKSPACE_ROOT/Makefile" << 'MAKEFILE'
# Platform Foundry Monorepo Makefile

.PHONY: all build test clean ce ee lint install help

all: ce ee

ce:
	cd ce/platformfoundry-ce && go run build.go build

ee:
	cd ee/platformfoundry-ee && go build -o bin/pf ./cmd/pf

test: test-ce test-ee

test-ce:
	cd ce/platformfoundry-ce && go run build.go test

test-ee:
	cd ee/platformfoundry-ee && go test -v ./...

clean:
	cd ce/platformfoundry-ce && go run build.go clean
	rm -rf ee/platformfoundry-ee/bin

install:
	cd ce/platformfoundry-ce && go run build.go install
	cd ee/platformfoundry-ee && go mod download && go mod tidy

workspace-sync:
	go work sync

help:
	@echo "Targets: all, ce, ee, test, test-ce, test-ee, clean, install, workspace-sync"
MAKEFILE
    echo "Created Makefile"
else
    echo "Makefile already exists"
fi

# Sync workspace
echo "Syncing Go workspace..."
cd "$WORKSPACE_ROOT"
go work sync

echo ""
echo "Workspace setup complete!"
echo "You can now use 'make' commands from: $WORKSPACE_ROOT"
