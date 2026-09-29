package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCronRequiresAuthorization(t *testing.T) {
	t.Setenv("CRON_SECRET", "expected-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/cron", nil)
	rec := httptest.NewRecorder()

	Cron(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without cron token, got %d", rec.Code)
	}
}
