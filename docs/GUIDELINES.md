# FileTug Development Guidelines for AI Agents & Humans

This document provides project-specific information for developers and AI agents working on FileTug.

## 1. Build and Configuration

- **Go Version**: Ensure you are using Go version specififedin `go.mod` (_or later_).
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

## 2. Development Information

- all code should follow our [coding standard](docs/CODING_STANDARDS.md)
- [Project structure](docs/project-structure.md)

## 3. Testing Information

FileTug aims for high (_90%+_) test coverage.

Our [coding standards](docs/CODING_STANDARDS.md) have dedicated "Tests" section.

- **Running All Tests**:
  ```shell
  go test ./...
  ```
- **Coverage Analysis**:
  ```shell
  go test -coverprofile=coverage.out ./...
  go tool cover -html=coverage.out
  ```


