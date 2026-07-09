package monitor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthOptionalWhenTokenEmpty(t *testing.T) {
	t.Setenv("MONITOR_TOKEN", "")
	server := New("../..")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard?hours=1", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("未配置令牌时应放行，got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthRequiredWhenTokenSet(t *testing.T) {
	t.Setenv("MONITOR_TOKEN", "secret-token-xyz")
	server := New("../..")

	// 无令牌 → 401
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard?hours=1", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	// Bearer 正确 → 200
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/dashboard?hours=1", nil)
	req.Header.Set("Authorization", "Bearer secret-token-xyz")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with bearer, got %d: %s", rec.Code, rec.Body.String())
	}

	// X-Monitor-Token 正确 → 200
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/dashboard?hours=1", nil)
	req.Header.Set("X-Monitor-Token", "secret-token-xyz")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with header, got %d", rec.Code)
	}

	// health / config 始终放行
	for _, path := range []string{"/api/health", "/api/config"} {
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, path, nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s should be open, got %d", path, rec.Code)
		}
	}
}

func TestConfigExposesAuthRequired(t *testing.T) {
	t.Setenv("MONITOR_TOKEN", "abc")
	server := New("../..")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"authRequired":true`) {
		t.Fatalf("expected authRequired true in config, got %s", body)
	}
}
