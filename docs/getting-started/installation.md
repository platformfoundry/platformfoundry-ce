# Installation

## Prerequisites

- Go 1.21 or later (for building from source)
- Git
- kubectl (for Kubernetes operations)
- Terraform/Pulumi (depending on infrastructure provider)

## Install Methods

### Binary Download

Download the latest release for your platform:

=== "Linux"

    ```bash
    curl -sSL https://github.com/platformfoundry/pf-ce/releases/latest/download/pf-linux-amd64 -o pf
    chmod +x pf
    sudo mv pf /usr/local/bin/
    ```

=== "macOS"

    ```bash
    curl -sSL https://github.com/platformfoundry/pf-ce/releases/latest/download/pf-darwin-amd64 -o pf
    chmod +x pf
    sudo mv pf /usr/local/bin/
    ```

=== "Windows"

    ```powershell
    Invoke-WebRequest -Uri "https://github.com/platformfoundry/pf-ce/releases/latest/download/pf-windows-amd64.exe" -OutFile "pf.exe"
    Move-Item pf.exe C:\Windows\System32\
    ```

### Build from Source

```bash
git clone https://github.com/platformfoundry/pf-ce.git
cd platformfoundry-ce
go run build.go build
```

The binary will be in `bin/pf`.

### Go Install

```bash
go install github.com/platformfoundry/pf-ce/cmd/pf@latest
```

## Verify Installation

```bash
pf version
```

Expected output:
```
PlatformFoundry v0.1.0
Go: go1.21.0
OS/Arch: linux/amd64
```

## Shell Completion

=== "Bash"

    ```bash
    pf completion bash > /etc/bash_completion.d/pf
    ```

=== "Zsh"

    ```bash
    pf completion zsh > "${fpath[1]}/_pf"
    ```

=== "Fish"

    ```bash
    pf completion fish > ~/.config/fish/completions/pf.fish
    ```

## Configuration Directory

PlatformFoundry stores configuration in:

| OS | Path |
|----|------|
| Linux/macOS | `~/.platformfoundry/` or `~/.pf/` |
| Windows | `%USERPROFILE%\.platformfoundry\` |

## Next Steps

- [Quickstart Guide](quickstart.md) - Create your first platform
- [Configuration](configuration.md) - Configure PlatformFoundry
