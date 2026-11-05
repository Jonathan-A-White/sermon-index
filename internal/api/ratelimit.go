package api

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The rate limit, a token bucket per client IP: 60 requests a minute (a
// token a second), up to 30 at once. Constants, not settings: a reader of
// Lampas makes a few requests a chapter.
const (
	ratePerMinute = 60
	rateBurst     = 30
	rateRefill    = time.Minute / ratePerMinute
)

type bucket struct {
	tokens float64
	at     time.Time
}

type limiter struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[string]*bucket
	swept   time.Time
}

func newLimiter(now func() time.Time) *limiter {
	return &limiter{now: now, buckets: map[string]*bucket{}}
}

// take spends one of ip's tokens; when it has none it says how long until
// it has one.
func (l *limiter) take(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: rateBurst, at: now}
		l.buckets[ip] = b
	}
	if elapsed := now.Sub(b.at); elapsed > 0 {
		b.tokens = math.Min(rateBurst, b.tokens+float64(elapsed)/float64(rateRefill))
	}
	b.at = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) * float64(rateRefill))
}

// sweep forgets, once a minute, the buckets idle long enough to be full
// again: a forgotten bucket and a full one are the same.
func (l *limiter) sweep(now time.Time) {
	if now.Sub(l.swept) < time.Minute {
		return
	}
	l.swept = now
	for ip, b := range l.buckets {
		if now.Sub(b.at) >= rateBurst*rateRefill {
			delete(l.buckets, ip)
		}
	}
}

func (l *limiter) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, wait := l.take(clientIP(r))
		if !ok {
			secs := int(math.Ceil(wait.Seconds()))
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			fail(w, r, http.StatusTooManyRequests, "rate_limited",
				"too many requests: "+strconv.Itoa(ratePerMinute)+" a minute, "+strconv.Itoa(rateBurst)+" at once; try again in "+strconv.Itoa(secs)+"s")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP is the request's client. The peer is the client, unless the
// peer is loopback or a private address (the VPS's nginx over WireGuard,
// or a proxy on this host): then X-Real-IP, else the last address in
// X-Forwarded-For, names the client. A public peer's headers are not
// believed, so a client cannot pick its own bucket.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer == nil || !(peer.IsLoopback() || peer.IsPrivate()) {
		return host
	}
	if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
		return ip.String()
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := net.ParseIP(strings.TrimSpace(parts[len(parts)-1])); ip != nil {
			return ip.String()
		}
	}
	return host
}
