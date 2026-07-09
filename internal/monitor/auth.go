package monitor

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// 鉴权为可选：仅当配置了 MONITOR_TOKEN 时校验 /api/*（/api/health 除外）。
// 支持 Authorization: Bearer <token> 或 X-Monitor-Token 头。

func (s *Server) withAuth(next http.Handler) http.Handler {
	token := strings.TrimSpace(s.authToken)
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !tokenMatches(extractToken(r), token) {
			writeError(w, http.StatusUnauthorized, "未授权：请提供有效的访问令牌")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requiresAuth(path string) bool {
	if path == "/api/health" || path == "/api/config" {
		return false
	}
	return strings.HasPrefix(path, "/api/")
}

func extractToken(r *http.Request) string {
	if h := strings.TrimSpace(r.Header.Get("X-Monitor-Token")); h != "" {
		return h
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(auth) >= 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}

func tokenMatches(provided, expected string) bool {
	if provided == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
