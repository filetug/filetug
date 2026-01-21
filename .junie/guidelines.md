# Development Guidelines for FileTug

This document provides project-specific information for developers working on FileTug.

## 1. Build and Configuration

- **Go Version**: Ensure you are using Go 1.25.5 or later as specified in `go.mod`.
- **Dependencies**: Managed via Go modules. Run `go mod download` to fetch them.
- **Main Entry Point**: The main application entry point is `main.go` in the root directory.
- **Build Command**:
  ```shell
  go build -o ft main.go
  ```
- **Running Locally**:
  ```shell
  go run main.go
  ```

## 2. Testing Information

FileTug aims for high (90%+) test coverage.

- **Running All Tests**:
  ```shell
  go test ./...
  ```
- **Coverage Analysis**:
  ```shell
  go test -coverprofile=coverage.out ./...
  go tool cover -html=coverage.out
  ```
- **Adding New Tests**:
    - Place tests in the same package as the code being tested, using the `_test.go` suffix.
    - The project uses both the standard `testing` package and `github.com/stretchr/testify/assert` for more expressive
      assertions.
    - For UI-related tests, check `pkg/sneatv/ttestutils` for helper functions.

- **Demonstration Test Example**:
  The following example demonstrates a simple test using `testify/assert`:
  ```go
  package pkg

  import (
    "testing"

    "github.com/stretchr/testify/assert"
  )

  func TestExample(t *testing.T) {
      expected := "filetug"
      actual := "filetug"
      assert.Equal(t, expected, actual, "Values should be equal")
  }
  ```

## 3. Additional Development Information

### Code Style

- Strictly follow standard Go idioms and formatting (`go fmt`).
- always check or explicitly ignore returned errors
- avoid calling functions inside calls of other functions:
- at the end always verify changes with `golangci-lint run` - should report no errors or warnings.
- Use `fmt.Fprintf` to `os.Stderr` or specific buffers instead of
  `fmt.Println` or `fmt.Printf` to avoid interfering with the TUI output on `stdout`.
- Prefer explicit error handling and avoid `panic` in production code.

### Readability / Debuggability rules

- No nested calls: don’t write `f2(f1())`; assign the intermediate result in a variable first.

### Project Structure

- `pkg/filetug`: Core TUI logic and components.
- `pkg/sneatv`: UI framework/components used by FileTug (tabs, buttons, tables).
- `pkg/gitutils`: Git integration helpers.
- `pkg/files`: File system abstraction and storage implementations (OS, FTP, HTTP).
- **Git Integration**: The project uses `github.com/go-git/go-git/v5` for git operations. Be mindful of performance when
  querying git status in large repositories; use concurrency where appropriate as seen in `pkg/gitutils`.
