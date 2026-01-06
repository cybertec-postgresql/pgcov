# Contributing to pgcov

Thank you for your interest in contributing to pgcov! This document provides guidelines and instructions for setting up your development environment and contributing to the project.

## Table of Contents

- [Development Environment Setup](#development-environment-setup)
- [Building the Project](#building-the-project)
- [Running Tests](#running-tests)
- [Code Style](#code-style)
- [Submitting Changes](#submitting-changes)
- [Troubleshooting](#troubleshooting)

## Development Environment Setup

### Prerequisites

1. **Go 1.21+**
   - Download from [golang.org](https://golang.org/dl/)
   - Verify: `go version`

2. **C Compiler** (required for CGO)

   **Linux**:

   ```bash
   # Ubuntu/Debian
   sudo apt-get install build-essential
   
   # Fedora/RHEL
   sudo dnf install gcc
   
   # Arch Linux
   sudo pacman -S base-devel
   ```

   **macOS**:

   ```bash
   # Install Xcode Command Line Tools
   xcode-select --install
   ```

   **Windows**:
   - Download and install [MSYS2](https://www.msys2.org/)
   - Open MSYS2 terminal:

     ```bash
     pacman -Syu
     pacman -S mingw-w64-x86_64-gcc
     ```

   - Add `C:\msys64\mingw64\bin` to your system PATH

3. **Docker** (for integration tests)
   - Linux: [Docker Engine](https://docs.docker.com/engine/install/)
   - macOS/Windows: [Docker Desktop](https://www.docker.com/products/docker-desktop/)
   - Verify: `docker ps`

4. **PostgreSQL** (optional, for manual testing)
   - Version 13 or later
   - Integration tests use Docker, but you may want a local instance for development

### Clone and Setup

```bash
# Fork the repository on GitHub first, then:
git clone https://github.com/YOUR_USERNAME/pgcov.git
cd pgcov

# Add upstream remote
git remote add upstream https://github.com/pashagolub/pgcov.git

# Install dependencies
go mod download
```

## Building the Project

### Linux/macOS

```bash
# Set CGO environment
export CGO_ENABLED=1

# Development build
go build -o pgcov ./cmd/pgcov

# Release build (optimized, smaller binary)
go build -ldflags="-s -w" -o pgcov ./cmd/pgcov

# Verify the build
./pgcov --version
```

### Windows (PowerShell)

```powershell
# Set CGO environment
$env:CGO_ENABLED = "1"
$env:CC = "C:\msys64\mingw64\bin\gcc.exe"
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"

# Development build
go build -o pgcov.exe .\cmd\pgcov

# Release build
go build -ldflags="-s -w" -o pgcov.exe .\cmd\pgcov

# Verify the build
.\pgcov.exe --version
```

### Build Script (Recommended)

Create a build script for convenience:

**Linux/macOS** (`build.sh`):

```bash
#!/bin/bash
set -e

export CGO_ENABLED=1
echo "Building pgcov..."
go build -ldflags="-s -w" -o pgcov ./cmd/pgcov
echo "Build complete: $(./pgcov --version)"
```

**Windows** (`build.ps1`):

```powershell
$ErrorActionPreference = "Stop"

$env:CGO_ENABLED = "1"
$env:CC = "C:\msys64\mingw64\bin\gcc.exe"
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"

Write-Host "Building pgcov..."
go build -ldflags="-s -w" -o pgcov.exe .\cmd\pgcov
Write-Host "Build complete: $(.\pgcov.exe --version)"
```

## Running Tests

### Quick Test

```bash
# Linux/macOS
export CGO_ENABLED=1
go test ./...

# Windows (PowerShell)
$env:CGO_ENABLED = "1"
$env:CC = "C:\msys64\mingw64\bin\gcc.exe"
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"
go test .\...
```

### Comprehensive Testing

**Linux/macOS**:

```bash
# Enable CGO
export CGO_ENABLED=1

# All tests with verbose output
go test -v ./...

# Integration tests only
go test -v ./internal -run TestEndToEndWithTestcontainers

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run tests with timeout (important for integration tests)
go test -timeout 5m ./...

# Run tests in parallel
go test -parallel 4 ./...

# Clean test cache (force re-run)
go clean -testcache
go test ./...
```

**Windows (PowerShell)**:

```powershell
# Enable CGO
$env:CGO_ENABLED = "1"
$env:CC = "C:\msys64\mingw64\bin\gcc.exe"
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"

# All tests with verbose output
go test -v .\...

# Integration tests only
go test -v .\internal -run TestEndToEndWithTestcontainers

# Run tests with coverage
go test -cover -coverprofile=coverage.out .\...
go tool cover -html=coverage.out -o coverage.html

# Run tests with timeout
go test -timeout 5m .\...

# Clean test cache
go clean -testcache
go test .\...
```

### Test Structure

- **Unit tests**: Fast, no external dependencies
  - `internal/discovery/*_test.go`
  - `internal/parser/*_test.go`
  - `internal/coverage/*_test.go`

- **Integration tests**: Use testcontainers (require Docker)
  - `internal/integration_test.go` - Full end-to-end workflow

### Integration Test Requirements

Integration tests use [testcontainers-go](https://golang.testcontainers.org/) to spin up a real PostgreSQL instance:

1. **Docker must be running**

   ```bash
   docker ps  # Should not error
   ```

2. **Sufficient Docker resources**
   - Memory: At least 2GB available
   - Disk: At least 1GB free space

3. **Network access**
   - Tests pull `postgres:16-alpine` image
   - Tests pull `testcontainers/ryuk:0.13.0` image

### Running Specific Tests

```bash
# Linux/macOS
export CGO_ENABLED=1

# Run only discovery tests
go test -v ./internal/discovery

# Run only a specific test function
go test -v ./internal -run TestEndToEndWithTestcontainers/Discovery

# Run tests matching a pattern
go test -v ./... -run TestParse

# Skip integration tests (no Docker required)
go test -v -short ./...
```

## Code Style

### Formatting

```bash
# Format all code
go fmt ./...

# Check for suspicious constructs
go vet ./...

# Install and run staticcheck
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

### Go Conventions

- Follow standard Go idioms and best practices
- Use `gofmt` for formatting
- Write clear, self-documenting code
- Add comments for exported functions and types
- Keep functions small and focused
- Use meaningful variable names

### Project-Specific Guidelines

1. **Error Handling**
   - Always check and handle errors
   - Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
   - Use custom error types in `internal/errors` package

2. **Testing**
   - Write tests for new features
   - Maintain or improve code coverage
   - Use table-driven tests where appropriate
   - Test error paths, not just happy paths

3. **Documentation**
   - Update README.md for user-facing changes
   - Add godoc comments for exported symbols
   - Include examples in documentation

4. **Commits**
   - Write clear, descriptive commit messages
   - Use conventional commit format: `type(scope): description`
   - Example: `feat(parser): add support for SQL functions`
   - Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`

## Submitting Changes

### Before Submitting

1. **Ensure all tests pass**

   ```bash
   go test ./...
   ```

2. **Format your code**

   ```bash
   go fmt ./...
   go vet ./...
   ```

3. **Update documentation**
   - Update README.md if adding features
   - Add/update godoc comments
   - Update CHANGELOG.md (if exists)

4. **Test your changes manually**

   ```bash
   # Build and test the binary
   go build -o pgcov ./cmd/pgcov
   ./pgcov run ./testdata/simple
   ```

### Pull Request Process

1. **Create a feature branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Keep commits focused and atomic
   - Write clear commit messages

3. **Push to your fork**

   ```bash
   git push origin feature/your-feature-name
   ```

4. **Open a Pull Request**
   - Provide a clear description
   - Reference any related issues
   - Include screenshots/examples if applicable

5. **Address review feedback**
   - Make requested changes
   - Push updates to the same branch

## Troubleshooting

### CGO Build Errors

**Error: `gcc: command not found`**

```bash
# Linux
sudo apt-get install build-essential

# macOS
xcode-select --install

# Windows - ensure MSYS2 MinGW64 is in PATH
```

**Error: `cannot find -lpthread`**

```bash
# Linux - install development libraries
sudo apt-get install build-essential
```

**Windows: Missing DLL errors**

```powershell
# Add MinGW bin to PATH
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"

# Or permanently via System Properties > Environment Variables
```

### Test Failures

**Integration test fails: "Cannot connect to Docker"**

```bash
# Verify Docker is running
docker ps

# Linux - ensure user is in docker group
sudo usermod -aG docker $USER
# Log out and back in
```

**Integration test fails: "Failed to pull image"**

```bash
# Pull images manually
docker pull postgres:16-alpine
docker pull testcontainers/ryuk:0.13.0
```

**Test timeout**

```bash
# Increase timeout
go test -timeout 10m ./...
```

### Import Errors

**Error: `package github.com/pganalyze/pg_query_go/v6: cannot find package`**

```bash
# Download missing dependencies
go mod download
go mod tidy
```

### Getting Help

- **Issues**: [GitHub Issues](https://github.com/pashagolub/pgcov/issues)
- **Discussions**: [GitHub Discussions](https://github.com/pashagolub/pgcov/discussions)
- **Documentation**: Check README.md and code comments

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
