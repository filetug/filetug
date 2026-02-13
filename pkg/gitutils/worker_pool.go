package gitutils

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/go-git/go-git/v5"
)

// GitStatusRequest represents a request to check git status
type GitStatusRequest struct {
	Repo     *git.Repository
	Path     string
	IsDir    bool
	Callback func(*RepoStatus)
}

// GitStatusWorkerPool manages a pool of workers for git status checks
type GitStatusWorkerPool struct {
	workers  int
	requests chan GitStatusRequest
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	closed   atomic.Bool
}

// NewGitStatusWorkerPool creates a new worker pool with the specified number of workers
func NewGitStatusWorkerPool(workers int) *GitStatusWorkerPool {
	if workers <= 0 {
		workers = 4 // Default to 4 workers
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	pool := &GitStatusWorkerPool{
		workers:  workers,
		requests: make(chan GitStatusRequest, workers*2), // Buffer for smoother operation
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// Start workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}
	
	return pool
}

// worker processes git status requests from the queue
func (p *GitStatusWorkerPool) worker() {
	defer p.wg.Done()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case req, ok := <-p.requests:
			if !ok {
				return
			}
			
			var status *RepoStatus
			if req.IsDir {
				status = GetDirStatus(p.ctx, req.Repo, req.Path)
			} else {
				status = GetFileStatus(p.ctx, req.Repo, req.Path)
			}
			
			if req.Callback != nil {
				req.Callback(status)
			}
		}
	}
}

// Submit adds a git status request to the queue
// Returns false if the pool is closed or context is cancelled
func (p *GitStatusWorkerPool) Submit(req GitStatusRequest) bool {
	select {
	case <-p.ctx.Done():
		return false
	default:
	}
	
	select {
	case p.requests <- req:
		return true
	case <-p.ctx.Done():
		return false
	default:
		// Non-blocking: drop request if queue is full
		return false
	}
}

// Close shuts down the worker pool and waits for all workers to finish
// This method is idempotent and can be called multiple times safely
func (p *GitStatusWorkerPool) Close() {
	if p.closed.Swap(true) {
		return // Already closed
	}
	p.cancel()
	close(p.requests)
	p.wg.Wait()
}

// Global worker pool instance
var (
	globalPoolMu   sync.Mutex
	globalPool     *GitStatusWorkerPool
	globalPoolOnce sync.Once
)

// GetGlobalPool returns the global worker pool instance, creating it if necessary
func GetGlobalPool() *GitStatusWorkerPool {
	globalPoolOnce.Do(func() {
		globalPool = NewGitStatusWorkerPool(4)
	})
	return globalPool
}

// SetGlobalPool replaces the global worker pool (useful for testing)
func SetGlobalPool(pool *GitStatusWorkerPool) {
	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()
	
	if globalPool != nil {
		globalPool.Close()
	}
	globalPool = pool
}
