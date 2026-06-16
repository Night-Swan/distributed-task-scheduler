package api

import (
	"testing"
)

func TestGetLimiter(t *testing.T) {
	ip := "192.168.1.1"
	limiter := getLimiter(ip)
	if limiter == nil {
		t.Errorf("Expected limiter to be created for IP %s", ip)
	}
}

func TestGetLimiterSameIP(t *testing.T) {
    ip := "192.168.1.2"
    limiter1 := getLimiter(ip)
    limiter2 := getLimiter(ip)
    if limiter1 != limiter2 {
        t.Fatal("expected same limiter for same IP, got different limiters")
    }
}
