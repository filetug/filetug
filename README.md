# FileTug – modern CLI file browser/picker with neat UI

- Free to use & Open source - [MIT License](LICENSE)
- Developed in [Go](https://go.dev/)

## ♺ Continuous Integration — [![Build and Test](https://github.com/datatug/filetug/actions/workflows/build.yml/badge.svg)](https://github.com/datatug/filetug/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/datatug/filetug?cache=0)](https://goreportcard.com/report/github.com/datatug/filetug) [![Coverage Status](https://coveralls.io/repos/github/datatug/filetug/badge.svg?branch=main&cache=1)](https://coveralls.io/github/datatug/filetug?branch=main) [![GoDoc](https://godoc.org/github.com/datatug/filetug?status.svg)](https://godoc.org/github.com/datatug/filetug)

We are targeting 100% test coverage.

## Installation

```shell
brew tap datatug/filetug
brew install filetug 
```

## Usage

To start in the current directory:

```shell
ft
```

To start at the specific directory or file:

```shell
ft <PATH_TO_DIRECTORY_OR_FILE>
```

## Libraries used

- [tview](https://github.com/rivo/tview) - Modern, rich, and extensible Go UI library for terminal applications
- [chrome](github.com/alecthomas/chroma) - Go syntax highlighting library

## Contributing

Contributions are welcome! Please read the [CONTRIBUTING.md](docs/CONTRIBUTING.md) for details on how to contribute to this
project.
