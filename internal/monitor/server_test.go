package monitor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer() *Server {
	server := New("../..")
	server.now = func() time.Time {
		return time.Unix(1783046400, 0).UTC()
	}
	return server
}

func TestDashboardEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard?hours=24", nil)
	newTestServer().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data DashboardData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Overview.TotalRecords == 0 {
		t.Fatal("expected non-zero total records")
	}
	if len(payload.Data.HourlyStats) != 24 {
		t.Fatalf("expected 24 hourly stats, got %d", len(payload.Data.HourlyStats))
	}
	if len(payload.Data.TopModels) == 0 {
		t.Fatal("expected top models")
	}
}

func TestKeyQuotaEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/key/quota?key=sk-test123456&hours=6", nil)
	newTestServer().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data KeyData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Token.MaskedKey == "" {
		t.Fatal("expected masked key")
	}
	if payload.Data.UsageSummary.ModelCount == 0 {
		t.Fatal("expected model count")
	}
}

func TestInvalidHours(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard?hours=5", nil)
	newTestServer().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestConfiguredDatabaseFailureIsVisible(t *testing.T) {
	t.Setenv("NEW_API_DB_DRIVER", "mysql")
	t.Setenv("NEW_API_DSN", "bad:bad@tcp(127.0.0.1:1)/new_api?parseTime=true")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard?hours=1", nil)
	New("../..").Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "connect") {
		t.Fatalf("expected database connection error, got %s", rec.Body.String())
	}
}
