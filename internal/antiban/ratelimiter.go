package antiban

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu         sync.Mutex
	timestamps []int64
	limit      int
	windowMs   int64
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		windowMs: 60_000,
	}
}

func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UnixMilli()
	r.prune(now)

	if len(r.timestamps) >= r.limit {
		return false
	}

	r.timestamps = append(r.timestamps, now)
	return true
}

func (r *RateLimiter) Usage() (current int, limit int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.prune(time.Now().UnixMilli())
	return len(r.timestamps), r.limit
}

func (r *RateLimiter) prune(now int64) {
	cutoff := now - r.windowMs
	i := 0
	for i < len(r.timestamps) && r.timestamps[i] < cutoff {
		i++
	}
	r.timestamps = r.timestamps[i:]
}
