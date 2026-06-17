package api

import (
	"golang.org/x/time/rate"
	"github.com/gin-gonic/gin"
	"sync"
	"time"
    "os"
)

var (
    mu sync.Mutex
    limiters = make(map[string]*rate.Limiter)
)

func getLimiter(ip string) *rate.Limiter {
    mu.Lock()
    defer mu.Unlock()
    if l, exists := limiters[ip]; exists {
        return l
    }
    l := rate.NewLimiter(rate.Every(time.Second), 5)
    limiters[ip] = l
    return l
}

func RateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if os.Getenv("DISABLE_RATE_LIMIT") == "true" {
            c.Next()
            return
        }
        limiter := getLimiter(c.ClientIP())
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "Too many requests"})
            c.Abort()
            return
        }
        c.Next()
    }
}