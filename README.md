# FileTug – modern CLI file browser/picker with neat UI

- Free to use & Open source - [GPLv3 License](LICENSE)
- Developed in [Go](https://go.dev/)

## ♺ Continuous Integration — [![Build and Test](https://github.com/datatug/filetug/actions/workflows/build.yml/badge.svg)](https://github.com/datatug/filetug/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/datatug/filetug?cache=0)](https://goreportcard.com/report/github.com/datatug/filetug) [![Coverage Status](https://coveralls.io/repos/github/datatug/filetug/badge.svg?branch=main&cache=4)](https://coveralls.io/github/datatug/filetug?branch=main) [![GoDoc](https://godoc.org/github.com/datatug/filetug?status.svg)](https://godoc.org/github.com/datatug/filetug)

We are targeting 100% test coverage.

## Why FileTug and not MC/ranger/etc.?

> Existing file managers show what files exist. Users want to know what those files are.

If I had to summarize in one sentence.

> Existing terminal file managers optimize for experts who already know them,
> not for humans trying to get work done.

### Killer features

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
- [chrome](https://github.com/alecthomas/chroma) - Go syntax highlighting library

## Contributing

Contributions are welcome! Please read the [CONTRIBUTING.md](docs/CONTRIBUTING.md) for details on how to contribute to
this
project.
