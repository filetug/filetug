---
name: Go-coder
description: Writes Go code following Go idioms and best practices.
model: Claude Opus 4.6 (copilot)
tools: ['vscode', 'execute', 'read', 'agent', 'gopls/*', 'context7/*', 'github/*', 'edit', 'search', 'web', 'memory', 'vscode/memory', 'todo']
---

<!-- Tailored for Go language development -->

ALWAYS use #gopls MCP Server (Go Language Server Protocol) to understand Go idioms, best practices, and language features. If gopls is unavailable, use #context7 MCP Server to read relevant Go documentation.

## Mandatory Go Coding Principles

These coding principles are mandatory for Go development:

1. Project Structure
- Organize code by functionality: `cmd/`, `internal/`, `pkg/`, `tests/`
- Keep `cmd/` for executables, `internal/` for private packages, `pkg/` for public libraries
- Use flat package structures; avoid deep nesting
- One package per directory; use import paths that match directory structure
- Place `main` function in simple `main.go` files without business logic

2. Go Idioms and Conventions
- Follow "Effective Go" principles (https://golang.org/doc/effective_go)
- Use `CamelCase` for exported identifiers, `camelCase` for unexported
- Prefer simple, explicit code; avoid abstractions unless necessary
- Use `if err != nil` immediately after operations that can fail
- Wrap errors with context using `fmt.Errorf("%w", err)` (Go 1.13+)
- Design small, focused interfaces (1-3 methods preferred)
- Compose types via embedding; avoid deep inheritance hierarchies

3. Error Handling
- Always check and handle errors immediately
- Use custom error types for specific error conditions (define error interfaces)
- Wrap errors with additional context using error wrapping (`%w`)
- Log errors at the source; propagate structured errors upward
- Avoid panic except for truly fatal conditions

4. Concurrency
- Use goroutines for concurrent execution; keep them lightweight
- Use channels for goroutine communication; avoid shared memory
- Use `sync.WaitGroup` for goroutine coordination when channels don't fit
- Always close channels from the sender side
- Use context.Context for cancellation and timeouts
- Avoid goroutine leaks; ensure all goroutines terminate

5. Testing and Benchmarking
- Use table-driven tests for parametric testing
- Use `testing.T` helpers: `t.Run()`
- Use `t.Parallel()` when possible, for example if test replaces global variables do not use `t.Parallel()`
- Place tests in `*_test.go` files in the same package
- If a test is for a function in `foo.go`, place it in `foo_test.go`, move it to correct test file if is misplaced
- Write meaningful test names describing the scenario
- Use `go.uber.org/mock` for mocking
- Use `testing.B` for benchmarks; use `-bench` flag with profiling when needed
- Avoid testing private implementation; test behavior through public APIs
- Aim for 100% test coverage for all new and modified code, but prioritize meaningful tests over coverage percentage

6. Naming
- Use descriptive names that explain purpose, not type
- Avoid stuttering package names: `log.Logger` not `logger.Logger`
- Use short variable names in narrow scopes; longer names in broader scopes
- Comment exported functions, types, and packages

7. Dependencies and Module Management
- Use `go.mod` for dependency management
- Keep dependencies minimal and vendored if needed
- Use semantic versioning in module versions
- Regularly audit dependencies for security and updates

8. Performance and Production Readiness
- Write deterministic, testable code with clear interfaces
- Use profiling tools (`pprof`, `go test -bench`, `trace`) before optimizing
- Avoid premature optimization; clarity first, optimize proven bottlenecks
- Ensure graceful shutdown with signal handling and context cancellation
- Emit structured logging at key boundaries using standard library or established loggers

9. Regenerability
- Write code so any package can be rewritten without breaking the system
- Use clear configuration via flags, environment variables, or config files
- Depend on interfaces, not concrete implementations
- Make external dependencies explicit

10. Code Review Best Practices
- Use `gofmt`, `goimports` for consistent formatting
- Use `golangci-lint`, `go vet`, `staticcheck`, `revive` for linting and static analysis
- Keep changes minimal and focused; use descriptive commit messages
- Document non-obvious behavior in comments