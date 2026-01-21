# Contributing to FileTug

Thank you for your interest in contributing to FileTug! We welcome contributions from everyone.

This document provides guidelines for contributing to the project. Following these guidelines helps ensure a smooth
process for everyone involved.

## How to Contribute

### Reporting Bugs

If you find a bug, please check the [existing issues](https://github.com/datatug/filetug/issues) to see if it has
already been reported. If not, please open a new issue and include:

- A clear and descriptive title.
- Steps to reproduce the bug.
- Expected behavior and what actually happened.
- Your operating system and terminal emulator.
- Any relevant logs or screenshots.

### Suggesting Enhancements

We're always looking for ways to improve FileTug! If you have an idea for a new feature or an enhancement:

1. Check the [existing issues](https://github.com/datatug/filetug/issues) to see if the feature has already been
   suggested.
2. If not, open a new issue and describe the proposed change, why it would be useful, and how you imagine it working.

### Pull Requests

We welcome pull requests for bug fixes, new features, and improvements to documentation.

1. **Fork the repository** and create your branch from `main`.
2. **Make your changes**.
3. **Ensure tests pass** (see [Development](#development) section).
4. **Update documentation** if necessary.
5. **Submit a pull request** with a clear description of your changes.

## Development

### Prerequisites

- [Go](https://go.dev/doc/install) (version specified in `go.mod` or later).

### Setup

1. Clone your fork of the repository:
   ```shell
   git clone https://github.com/YOUR_USERNAME/filetug.git
   cd filetug
   ```

2. Download dependencies:
   ```shell
   go mod download
   ```

### Running Tests

To run all tests:

```shell
go test ./...
```

To run tests with coverage:

```shell
go test -cover ./...
```

### Building from source

To build the executable:

```shell
go build -o ft main.go
```

## Coding Standards

Please read [our guidelines](GUIDELINES.md) and follow our [coding standards](CODING_STANDARDS.md).

## License

By contributing to FileTug, you agree that your contributions will be licensed under the
project's [GPLv3 License](../LICENSE).
