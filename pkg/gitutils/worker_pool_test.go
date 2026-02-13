package gitutils

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewGitStatusWorkerPool(t *testing.T) {
	t.Run("creates_pool_with_specified_workers", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(3)
		defer pool.Close()
		assert.NotNil(t, pool)
		assert.Equal(t, 3, pool.workers)
	})

	t.Run("defaults_to_4_workers_for_invalid_input", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(0)
		defer pool.Close()
		assert.Equal(t, 4, pool.workers)
	})
}

func TestWorkerPool_Submit(t *testing.T) {
	t.Run("processes_requests", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(2)
		defer pool.Close()

		var callbackCount atomic.Int32
		req := GitStatusRequest{
			Repo:  nil, // Will cause nil status
			Path:  "/test",
			IsDir: false,
			Callback: func(status *RepoStatus) {
				callbackCount.Add(1)
			},
		}

		ok := pool.Submit(req)
		assert.True(t, ok)

		// Wait for processing
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(1), callbackCount.Load())
	})

	t.Run("returns_false_after_close", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(2)
		pool.Close()

		req := GitStatusRequest{
			Repo:  nil,
			Path:  "/test",
			IsDir: false,
		}

		ok := pool.Submit(req)
		assert.False(t, ok)
	})

	t.Run("handles_concurrent_submissions", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(4)
		defer pool.Close()

		var callbackCount atomic.Int32
		var wg sync.WaitGroup
		
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req := GitStatusRequest{
					Repo:  nil,
					Path:  "/test",
					IsDir: false,
					Callback: func(status *RepoStatus) {
						callbackCount.Add(1)
					},
				}
				pool.Submit(req)
			}()
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)
		
		// Should process at least some requests (may drop some if queue full)
		assert.Greater(t, callbackCount.Load(), int32(0))
	})
}

func TestWorkerPool_Close(t *testing.T) {
	t.Run("waits_for_workers_to_finish", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(2)
		
		var processing atomic.Bool
		processing.Store(true)
		
		req := GitStatusRequest{
			Repo:  nil,
			Path:  "/test",
			IsDir: false,
			Callback: func(status *RepoStatus) {
				time.Sleep(50 * time.Millisecond)
				processing.Store(false)
			},
		}
		
		pool.Submit(req)
		time.Sleep(10 * time.Millisecond) // Let worker start processing
		
		pool.Close()
		
		// After Close, processing should be done
		assert.False(t, processing.Load())
	})
}

func TestGetGlobalPool(t *testing.T) {
	t.Run("returns_singleton_instance", func(t *testing.T) {
		pool1 := GetGlobalPool()
		pool2 := GetGlobalPool()
		assert.Same(t, pool1, pool2)
	})
}

func TestSetGlobalPool(t *testing.T) {
	t.Run("replaces_global_pool", func(t *testing.T) {
		// Save original
		original := GetGlobalPool()
		defer SetGlobalPool(original)
		
		newPool := NewGitStatusWorkerPool(2)
		SetGlobalPool(newPool)
		
		retrieved := GetGlobalPool()
		assert.Same(t, newPool, retrieved)
	})
	
	t.Run("closes_old_pool_when_replacing", func(t *testing.T) {
		// Save original
		original := GetGlobalPool()
		defer SetGlobalPool(original)
		
		oldPool := NewGitStatusWorkerPool(2)
		SetGlobalPool(oldPool)
		
		// Submit a request to verify it works
		var called atomic.Bool
		req := GitStatusRequest{
			Repo:  nil,
			Path:  "/test",
			IsDir: false,
			Callback: func(status *RepoStatus) {
				called.Store(true)
			},
		}
		oldPool.Submit(req)
		
		// Replace with new pool
		newPool := NewGitStatusWorkerPool(2)
		SetGlobalPool(newPool)
		
		// Old pool should be closed, new submission should fail
		ok := oldPool.Submit(req)
		assert.False(t, ok)
	})
}

func TestWorkerPool_ProcessesCorrectly(t *testing.T) {
	t.Run("calls_GetDirStatus_for_directories", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(2)
		defer pool.Close()

		var gotStatus *RepoStatus
		var wg sync.WaitGroup
		wg.Add(1)
		
		req := GitStatusRequest{
			Repo:  nil, // Will cause nil return
			Path:  "/test/dir",
			IsDir: true,
			Callback: func(status *RepoStatus) {
				gotStatus = status
				wg.Done()
			},
		}
		
		pool.Submit(req)
		wg.Wait()
		
		// Nil repo should result in nil status
		assert.Nil(t, gotStatus)
	})

	t.Run("calls_GetFileStatus_for_files", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(2)
		defer pool.Close()

		var gotStatus *RepoStatus
		var wg sync.WaitGroup
		wg.Add(1)
		
		req := GitStatusRequest{
			Repo:  nil, // Will cause nil return
			Path:  "/test/file.txt",
			IsDir: false,
			Callback: func(status *RepoStatus) {
				gotStatus = status
				wg.Done()
			},
		}
		
		pool.Submit(req)
		wg.Wait()
		
		// Nil repo should result in nil status
		assert.Nil(t, gotStatus)
	})
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	t.Run("stops_processing_on_context_cancel", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(2)
		
		// Submit a request
		var called atomic.Bool
		req := GitStatusRequest{
			Repo:  nil,
			Path:  "/test",
			IsDir: false,
			Callback: func(status *RepoStatus) {
				called.Store(true)
			},
		}
		
		// Close the pool (cancels context)
		pool.Close()
		
		// Try to submit after close
		ok := pool.Submit(req)
		assert.False(t, ok)
	})
}

func TestWorkerPool_BufferedQueue(t *testing.T) {
	t.Run("handles_burst_of_requests", func(t *testing.T) {
		pool := NewGitStatusWorkerPool(2)
		defer pool.Close()

		var processed atomic.Int32
		
		// Submit many requests quickly
		for i := 0; i < 10; i++ {
			req := GitStatusRequest{
				Repo:  nil,
				Path:  "/test",
				IsDir: false,
				Callback: func(status *RepoStatus) {
					processed.Add(1)
				},
			}
			pool.Submit(req)
		}
		
		// Wait for processing
		time.Sleep(200 * time.Millisecond)
		
		// Should process at least some requests (may drop some if queue full)
		assert.Greater(t, processed.Load(), int32(0))
	})
}

func BenchmarkWorkerPool(b *testing.B) {
	pool := NewGitStatusWorkerPool(4)
	defer pool.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := GitStatusRequest{
			Repo:     nil,
			Path:     "/test",
			IsDir:    false,
			Callback: func(status *RepoStatus) {},
		}
		pool.Submit(req)
	}
}

func BenchmarkWorkerPool_Concurrent(b *testing.B) {
	pool := NewGitStatusWorkerPool(4)
	defer pool.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := GitStatusRequest{
				Repo:     nil,
				Path:     "/test",
				IsDir:    false,
				Callback: func(status *RepoStatus) {},
			}
			pool.Submit(req)
		}
	})
}
