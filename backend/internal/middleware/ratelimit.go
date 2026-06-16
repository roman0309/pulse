package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter applies a per-client-IP token bucket limit.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*rate.Limiter),
		rps:     rate.Limit(rps),
		burst:   burst,
	}
}

func (rl *RateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	l, ok := rl.clients[ip]
	if !ok {
		l = rate.NewLimiter(rl.rps, rl.burst)
		rl.clients[ip] = l
	}
	return l
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.limiterFor(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
