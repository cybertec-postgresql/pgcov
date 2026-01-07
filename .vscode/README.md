# VS Code Configuration for pgcov

This directory contains VS Code workspace settings optimized for developing pgcov.

## Files

- **settings.json** - Workspace settings with CGO configuration
- **launch.json** - Debug configurations for running and testing
- **tasks.json** - Build and test tasks with CGO support
- **README.md** - This file

## CGO Configuration

pgcov requires CGO because it uses `pg_query_go`, which wraps PostgreSQL's C parser library.

### Settings Applied (settings.json)

#### 1. Go Tools Environment (`go.toolsEnvVars`)

Configures gopls (Go language server) and other Go tools to use CGO:

- `CGO_ENABLED=1` - Enables CGO compilation
- `CC=C:\msys64\mingw64\bin\gcc.exe` (Windows) - Sets C compiler path

This ensures:

- ✅ gopls can analyze CGO code correctly
- ✅ `Go: Build Package` command works
- ✅ `Go: Test Package` command works
- ✅ IntelliSense and code navigation work properly

#### 2. Integrated Terminal Environment

Configures the VS Code integrated terminal with CGO variables:

**Windows** (`terminal.integrated.env.windows`):

```json
{
  "CGO_ENABLED": "1",
  "CC": "C:\\msys64\\mingw64\\bin\\gcc.exe",
  "PATH": "${env:PATH};C:\\msys64\\mingw64\\bin"
}
```

**Linux/macOS** (`terminal.integrated.env.linux/osx`):

```json
{
  "CGO_ENABLED": "1"
}
```

This ensures:

- ✅ `go build` works in the integrated terminal
- ✅ `go test` works in the integrated terminal
- ✅ No need to manually set environment variables

#### 3. gopls Build Environment

Additional gopls configuration for CGO:

```json
{
  "gopls": {
    "build.env": {
      "CGO_ENABLED": "1"
    }
  }
}
```

## Debug Configurations (launch.json)

### Available Configurations

1. **Launch pgcov** - Run pgcov with test data
   - Runs `pgcov run ./testdata/simple --verbose`
   - Good for quick testing during development

2. **Launch pgcov with custom args** - Run with custom arguments
   - Specify arguments in the debug console
   - Useful for testing different scenarios

3. **Test Current Package** - Test the package of the open file
   - Press `F5` while in any Go file
   - Runs all tests in that package

4. **Test Current File** - Test specific function
   - Select test function name
   - Runs only that test

5. **Integration Test** - Run full integration test suite
   - Runs `TestEndToEndWithTestcontainers`
   - Includes 5-minute timeout

### Using Debug Configurations

1. Open Run and Debug sidebar (`Ctrl+Shift+D`)
2. Select a configuration from the dropdown
3. Press `F5` or click "Start Debugging"
4. Set breakpoints by clicking line numbers

## Build Tasks (tasks.json)

### Available Tasks

Run tasks via `Ctrl+Shift+P` → "Tasks: Run Task" or `Ctrl+Shift+B` for default build.

#### Build Tasks

- **Build pgcov** (default) - Standard debug build
- **Build pgcov (Release)** - Optimized build with `-ldflags=-s -w`

#### Test Tasks

- **Test All** (default) - Run all tests with `-v`
- **Test with Coverage** - Generate coverage report
- **Integration Test** - Run end-to-end tests

#### Utility Tasks

- **Run pgcov** - Build and run pgcov on test data
- **Clean Build Cache** - Clear Go build and test cache
- **Go Mod Tidy** - Clean up go.mod and go.sum
- **Format Code** - Run `go fmt ./...`
- **Go Vet** - Run `go vet ./...`

### Running Tasks

**Via Command Palette**:

1. Press `Ctrl+Shift+P`
2. Type "Tasks: Run Task"
3. Select task from list

**Via Keyboard Shortcut**:

- `Ctrl+Shift+B` - Run default build task

**Via Terminal Menu**:

1. Terminal → Run Task
2. Select task

## Platform-Specific Setup

### Windows

**Prerequisites**:

1. Install [MSYS2](https://www.msys2.org/)
2. Install MinGW-w64 GCC:

   ```bash
   pacman -S mingw-w64-x86_64-gcc
   ```

3. Verify `C:\msys64\mingw64\bin\gcc.exe` exists

**If GCC is installed elsewhere**, update the `CC` path in `settings.json`:

```json
"go.toolsEnvVars": {
    "CC": "C:\\path\\to\\your\\gcc.exe"
},
"terminal.integrated.env.windows": {
    "CC": "C:\\path\\to\\your\\gcc.exe",
    "PATH": "${env:PATH};C:\\path\\to\\your\\mingw64\\bin"
}
```

### Linux

**Prerequisites**:

```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# Fedora/RHEL
sudo dnf install gcc

# Arch Linux
sudo pacman -S base-devel
```

CGO will automatically find GCC via PATH.

### macOS

**Prerequisites**:

```bash
# Install Xcode Command Line Tools
xcode-select --install
```

CGO will automatically find the compiler via Xcode.

## Using VS Code with pgcov

### Building

1. Open integrated terminal (`Ctrl+`` or`Cmd+``)
2. Run:

   ```bash
   go build ./cmd/pgcov
   ```

   CGO environment is automatically applied ✓

### Running Tests

1. Open integrated terminal
2. Run:

   ```bash
   go test ./...
   ```

   Or use VS Code's test UI (Testing sidebar)

### Debugging

1. Set breakpoints in your code
2. Press `F5` or use "Run and Debug" sidebar
3. CGO debugging works with proper C toolchain

### Language Server (gopls)

- Code completion works for CGO code
- Go to definition works across Go/C boundaries
- Hover documentation includes C type information
- Error highlighting respects CGO build constraints

## Troubleshooting

### Issue: "gcc: command not found"

**Solution**: Install GCC (see Platform-Specific Setup above)

### Issue: gopls shows errors but code compiles

**Solution**: Reload VS Code window (`Ctrl+Shift+P` → "Reload Window")

### Issue: Terminal doesn't have CGO variables

**Solution**:

1. Close and reopen the integrated terminal
2. Or restart VS Code

### Issue: Different GCC path on Windows

**Solution**: Edit `.vscode/settings.json` and update:

- `go.toolsEnvVars.CC`
- `terminal.integrated.env.windows.CC`
- `terminal.integrated.env.windows.PATH`

### Issue: Tests fail with "cannot find package"

**Solution**: Run in terminal:

```bash
go mod download
go mod tidy
```

## Recommended Extensions

Install these VS Code extensions for the best experience:

1. **Go** (`golang.go`) - Essential for Go development
2. **C/C++** (`ms-vscode.cpptools`) - Helps with CGO C code
3. **Test Explorer UI** - Better test visualization

## References

- [VS Code Go Documentation](https://code.visualstudio.com/docs/languages/go)
- [gopls Settings](https://github.com/golang/tools/blob/master/gopls/doc/settings.md)
- [pgcov README](../README.md)
- [pgcov BUILD Instructions](../BUILD.md)
