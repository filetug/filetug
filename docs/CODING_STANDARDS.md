# FileTug Coding Standards

## Generic and common sense

- Follow standard Go idioms and formatting (`go fmt`).
- Ensure that tests cover new code for at least 90% of lines.
- Keep commit focused on a single change.
- If possible keep pull requests focused on a single change.

## Code Style

- Strictly follow standard Go idioms and formatting (`go fmt`).
- always check or explicitly ignore returned errors
- avoid calling functions inside calls of other functions:
- at the end always verify changes with `golangci-lint run` - should report no errors or warnings.
- Use `fmt.Fprintf` to `os.Stderr` or specific buffers instead of
  `fmt.Println` or `fmt.Printf` to avoid interfering with the TUI output on `stdout`.
- Prefer explicit error handling and avoid `panic` in production code.

## Readability / Debuggability rules

- No nested calls: don’t write `f2(f1())`; assign the intermediate result in a variable first.

## Standard for commit messages:

All commits MUST follow the Conventional Commits specification.

```text
<type>(<scope>): <short summary>
```

- The type MUST be one of: `feat, fix, docs, refactor, test, chore, ci, perf`
- The summary MUST be descriptive, concise, imperative, and written in lowercase
- The summary MUST NOT end with a period
- Commits that introduce breaking changes MUST include ! after the type or scope

## Performance & UX

Responsiveness of the app is critical for a good user experience.

- Avoid blocking the UI thread for long operations.
- Use goroutine for asynchronous operations.
  For example, when fetching files from a remote server
  or checking git status on a git directory, use a goroutine.
  The result of the long-running operation should be provided to UI with a call to
  `*Navigator.queueUpdateDraw(f func())`
  which can be replaced with a mock in tests if needed to avoid deadlocks.
- While information is loading display a loading indicator (or progress bar if possible).
- Use `sync.WaitGroup`, `sync.Mutex`, etc. for managing goroutines
  and ensure that all goroutines are properly waited for before proceeding with the next operation.
- Avoid using global variables unless absolutely necessary.
  Instead, pass data through function parameters and return values.
- Use `context.Context` for cancellation and timeouts in goroutines to ensure that resources are released properly.
- Make sure if UI component initiated a goroutine and is closed, it cancels the context.

## Tests

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
