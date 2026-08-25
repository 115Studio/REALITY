package reality

import (
	"net"
	"sync"
	"time"
)

// A censor's probe sweep sends dozens of bad hellos a minute; a real client behind a broken
// clock or a stale key sends a handful. 10 was low enough that a few retrying clients on
// one NAT went dark together, and "silent after ten connections" is itself a tell.
const (
	fallbackRateWindow   = time.Minute
	fallbackRateMaxCount = 60
	fallbackCleanupEvery = 5 * time.Minute
)

type fallbackRateLimiter struct {
	mu      sync.Mutex
	windows map[string][]time.Time
}

var globalFallbackRL = &fallbackRateLimiter{
	windows: make(map[string][]time.Time),
}

func init() {
	go globalFallbackRL.cleanupLoop()
}

// recordFallback records a non-authenticated fallback connection and returns
// the IPv4 string and whether the rate limit has been exceeded.
// Returns ("", false) for IPv6, invalid, or unparseable addresses.
func (r *fallbackRateLimiter) recordFallback(remoteAddr string) (string, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return "", false
	}
	if ip.To4() == nil {
		return "", false
	}

	ipStr := ip.String()
	now := time.Now()
	cutoff := now.Add(-fallbackRateWindow)

	r.mu.Lock()
	defer r.mu.Unlock()

	timestamps := r.windows[ipStr]
	pruned := make([]time.Time, 0, len(timestamps)+1)
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}
	pruned = append(pruned, now)
	r.windows[ipStr] = pruned

	return ipStr, len(pruned) > fallbackRateMaxCount
}

func (r *fallbackRateLimiter) cleanupLoop() {
	for {
		time.Sleep(fallbackCleanupEvery)
		r.cleanup()
	}
}

func (r *fallbackRateLimiter) cleanup() {
	cutoff := time.Now().Add(-fallbackRateWindow)

	r.mu.Lock()
	defer r.mu.Unlock()

	for ip, timestamps := range r.windows {
		alive := false
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				alive = true
				break
			}
		}
		if !alive {
			delete(r.windows, ip)
		}
	}
}
