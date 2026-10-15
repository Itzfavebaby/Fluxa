package reconcile

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestForceSettleRejectsInvalidTransferID(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/v1/admin", NewHandler(nil).AdminRoutes(nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/transfers/not-a-uuid/force-settle", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", rec.Code, rec.Body.String())
	}
}
