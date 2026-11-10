package domain

import "time"

// MaxRequestsPerSecond bounds the plan and walk requests of a player: each one runs a pathfinding.
const MaxRequestsPerSecond = 10

// RequestLimiter is a bucket of MaxRequestsPerSecond requests that refills at that rate.
type RequestLimiter struct {
	tokens float64
	last   time.Time
}

// NewRequestLimiter starts full.
func NewRequestLimiter(now time.Time) RequestLimiter {
	return RequestLimiter{tokens: MaxRequestsPerSecond, last: now}
}

// Allow takes one request from the bucket, false when it's empty.
func (l *RequestLimiter) Allow(now time.Time) bool {
	refill := now.Sub(l.last).Seconds() * MaxRequestsPerSecond
	l.tokens = min(MaxRequestsPerSecond, l.tokens+max(refill, 0))
	l.last = now
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}
