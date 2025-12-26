# Installation

This guide covers all installation methods for Relaypoint.

## System Requirements

| Requirement | Minimum               | Recommended |
| ----------- | --------------------- | ----------- |
| CPU         | 1 core                | 2+ cores    |
| Memory      | 64 MB                 | 256 MB      |
| Disk        | 20 MB                 | 50 MB       |
| OS          | Linux, macOS, Windows | Linux       |

## Installation Methods

### Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/relaypoint/relaypoint/releases).

#### Linux

```bash
# AMD64
curl -LO https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-linux-amd64.tar.gz
tar -xzf relaypoint-linux-amd64.tar.gz
sudo mv relaypoint /usr/local/bin/
sudo chmod +x /usr/local/bin/relaypoint

# ARM64 (Raspberry Pi, AWS Graviton, etc.)
curl -LO https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-linux-arm64.tar.gz
tar -xzf relaypoint-linux-arm64.tar.gz
sudo mv relaypoint /usr/local/bin/
sudo chmod +x /usr/local/bin/relaypoint
```

#### macOS

```bash
# Intel Mac
curl -LO https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-darwin-amd64.tar.gz
tar -xzf relaypoint-darwin-amd64.tar.gz
sudo mv relaypoint /usr/local/bin/

# Apple Silicon (M1/M2/M3)
curl -LO https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-darwin-arm64.tar.gz
tar -xzf relaypoint-darwin-arm64.tar.gz
sudo mv relaypoint /usr/local/bin/
```

#### Windows

1. Download `relaypoint-windows-amd64.zip` from [GitHub Releases](https://github.com/relaypoint/relaypoint/releases)
2. Extract the ZIP file
3. Add the directory to your PATH or move `relaypoint.exe` to a directory in your PATH

```powershell
# PowerShell
Invoke-WebRequest -Uri "https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-windows-amd64.zip" -OutFile "relaypoint.zip"
Expand-Archive -Path "relaypoint.zip" -DestinationPath "C:\Program Files\Relaypoint"
```

### Building from Source

#### Prerequisites

- Go 1.24 or later
- Git
- Make (optional, but recommended)

#### Clone and Build

```bash
# Clone the repository
git clone https://github.com/relaypoint/relaypoint.git
cd relaypoint

# Build using Make
make build

# Or build directly with Go
go build -o relaypoint ./cmd/relaypoint
```

#### Build Options

```bash
# Build for all platforms
make build-all

# Build with version information
VERSION=v1.0.0 make build

# Build for a specific platform
GOOS=linux GOARCH=amd64 go build -o relaypoint-linux-amd64 ./cmd/relaypoint
```

#### Install Locally

```bash
# Install to /usr/local/bin (requires sudo)
make install

# Uninstall
make uninstall
```

## Verifying Installation

After installation, verify Relaypoint is working:

```bash
# Check version
relaypoint -version

# Validate configuration
relaypoint -config relaypoint.yml -validate

# Start in foreground
relaypoint -config relaypoint.yml

# Test endpoints
curl http://localhost:8080/health
curl http://localhost:9090/metrics
```

## Upgrading

### Binary Upgrade

````bash
# Download new version
curl -LO https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-linux-amd64.tar.gz
tar -xzf relaypoint-linux-amd64.tar.gz
sudo mv relaypoint /usr/local/bin/

## Uninstalling

### Binary Uninstall

```bash
# Remove files
sudo rm /usr/local/bin/relaypoint
sudo rm -rf /etc/relaypoint
sudo rm -rf /var/log/relaypoint
````

## Next Steps

- [Getting Started](./getting-started.md) - Your first Relaypoint configuration
- [Configuration Reference](./configuration.md) - Complete configuration options
