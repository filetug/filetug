# Git Status Worker Pool Implementation

## Overview

This commit adds a worker pool for processing git status checks, replacing the previous approach of spawning unlimited goroutines. This improves resource management and provides better control over concurrent git operations.

## Changes

### New Files

1. **pkg/gitutils/worker_pool.go**
   - `GitStatusWorkerPool`: Manages a fixed pool of workers for git status checks
   - `GitStatusRequest`: Request structure for submitting git status checks
   - `GetGlobalPool()`: Returns singleton worker pool instance
   - `SetGlobalPool()`: Replaces global pool (useful for testing)

2. **pkg/gitutils/worker_pool_test.go**
   - Comprehensive test coverage for worker pool functionality
   - Tests for concurrent submissions, context cancellation, and pool management

### Modified Files

1. **pkg/filetug/files_git.go**
   - `updateGitStatuses()`: Now uses worker pool instead of spawning individual goroutines
   - Submits requests to pool with callbacks for UI updates

## Key Features

### Worker Pool Benefits

1. **Resource Control**: Fixed number of workers (default 4) prevents unbounded goroutine creation
2. **Efficient Queue Management**: Buffered channel prevents blocking when submitting requests
3. **Graceful Shutdown**: Pool can be closed cleanly, waiting for workers to finish
4. **Context Support**: Respects context cancellation for coordinated shutdown
5. **Non-blocking Submission**: Drops requests if queue is full rather than blocking

### Implementation Details

- **Default Workers**: 4 concurrent workers (configurable)
- **Queue Buffer**: 2x workers capacity for smooth operation
- **Idempotent Close**: Safe to call Close() multiple times
- **Global Instance**: Singleton pattern for application-wide use
- **Thread-Safe**: All operations are goroutine-safe

## Usage Example

```go
// Get the global pool
pool := gitutils.GetGlobalPool()

// Submit a request
req := gitutils.GitStatusRequest{
    Repo:  repo,
    Path:  "/path/to/file",
    IsDir: false,
    Callback: func(status *gitutils.RepoStatus) {
        // Handle the status result
        fmt.Println(status.String())
    },
}
pool.Submit(req)

// Pool automatically manages workers
// No need to close unless shutting down application
```

## Testing

All existing tests pass, including:
- Worker pool unit tests
- Integration tests with git operations
- Concurrent submission tests
- Context cancellation tests

Run tests:
```bash
go test ./pkg/gitutils -v -run TestWorkerPool
```

## Performance Comparison

**Before (unlimited goroutines):**
- Each git status check spawned a new goroutine
- Semaphore limited to 2 concurrent operations
- Potential for goroutine explosion with many files
- Less predictable resource usage

**After (worker pool):**
- Fixed 4 workers processing requests from queue
- Better resource utilization
- More predictable performance
- Easier to tune based on system capacity

## Configuration

The worker pool can be configured by creating a custom pool:

```go
// Create pool with 8 workers
customPool := gitutils.NewGitStatusWorkerPool(8)
gitutils.SetGlobalPool(customPool)
```

## Future Enhancements

Potential improvements:
- Dynamic worker scaling based on load
- Priority queue for important requests
- Metrics/monitoring for pool utilization
- Per-repository worker pools for better isolation
