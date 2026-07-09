package monitor

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// 简易 IP 滑动窗口限流，主要用于 Key / 渠道探测接口。

type rateLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	limit    int
	visitors map[string]*visitor
}

type visitor struct {
	hits  []time.Time
	last  time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		limit = 30
	}
	if window <= 0 {
		window = time.Minute
	}
	return &rateLimiter{
		window:   window,
		limit:    limit,
		visitors: make(map[string]*visitor),
	}
}

func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok {
		rl.visitors[key] = &visitor{hits: []time.Time{now}, last: now}
		return true
	}

	cutoff := now.Add(-rl.window)
	kept := v.hits[:0]
	for _, t := range v.hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	v.hits = kept
	v.last = now

	if len(v.hits) >= rl.limit {
		return false
	}
	v.hits = append(v.hits, now)

	// 偶尔清理过期 visitor
	if len(rl.visitors) > 2048 {
		for k, vis := range rl.visitors {
			if now.Sub(vis.last) > rl.window*2 {
				delete(rl.visitors, k)
			}
		}
	}
	return true
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) withKeyRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !s.keyLimiter.allow(ip) {
			writeError(w, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
			return
		}
		next(w, r)
	}
}
