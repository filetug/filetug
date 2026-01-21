# FileTug – modern CLI file browser/picker with neat UI

- Free to use & Open source - [GPLv3 License](LICENSE)
- Developed using [Go](https://go.dev/) programming language

## ♺ Continuous Integration — [![Build and Test](https://github.com/datatug/filetug/actions/workflows/build.yml/badge.svg)](https://github.com/datatug/filetug/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/datatug/filetug?cache=0)](https://goreportcard.com/report/github.com/datatug/filetug) [![Coverage Status](https://coveralls.io/repos/github/datatug/filetug/badge.svg?branch=main&cache=5)](https://coveralls.io/github/datatug/filetug?branch=main) [![GoDoc](https://godoc.org/github.com/datatug/filetug?status.svg)](https://godoc.org/github.com/datatug/filetug)

We are targeting to achieve 100% test coverage (_with a minimum threshold of 90%_).

## Why FileTug and not MC/ranger/etc.?

> Other file managers show what files exist. Users want to know what those files are.

### Key Differentiators:

- It is fast!
    - Non-blocking progressive UI (_that pulls data in the background_).
    - Predictive pre-fetching
    - Caching of data for network resources (_with in-background refresh_)
- Smart previewers showing summary and key info for a file/directory
- Smart summarizer that provides a concise overview of directory contents
- Quick selection of files and directories by mask with a collection of named patterns
- Quick navigation to favotite, frequently used and recent directories
- Build-in git client that provides git status and allows to stage/commit/rollback/etc.

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
- [chroma](https://github.com/alecthomas/chroma) - Go syntax highlighting library

## Contributing

Contributions are welcome!

Please read the [CONTRIBUTING.md](docs/CONTRIBUTING.md) for details on how to contribute to this project.

All contributors and AI agents should follow [our guidelines](docs/GUIDELINES.md).
