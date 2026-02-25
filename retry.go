package zerotrue

import (
	"math"
	"math/rand"
	"time"
)

func shouldRetry(statusCode int) bool {
	switch statusCode {
	case 500, 502, 503, 504, 429:
		return true
	default:
		return false
	}
}

func backoff(attempt int, min, max time.Duration) time.Duration {
	base := float64(min) * math.Pow(2, float64(attempt))
	// Jitter: multiply by random factor in [0.75, 1.25]
	jitter := 0.75 + rand.Float64()*0.5
	d := time.Duration(base * jitter)
	if d > max {
		d = max
	}
	return d
}
