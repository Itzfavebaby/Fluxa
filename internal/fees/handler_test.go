package fees_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fluxa/fluxa/internal/domain"
	"github.com/fluxa/fluxa/internal/fees"
	"github.com/fluxa/fluxa/internal/tenant"
	"github.com/go-chi/chi/v5"
)

// recordingFeeRepo captures every schedule a handler asks it to persist so a
// test can assert the tenant it would have written.
type recordingFeeRepo struct {
	fees.Repository
	sets []*domain.FeeSchedule
}

func (r *recordingFeeRepo) SetSchedule(_ context.Context, schedule *domain.FeeSchedule) error {
	r.sets = append(r.sets, schedule)
	return nil
}

func doPutSchedule(t *testing.T, callerTenant, body string) (int, *recordingFeeRepo) {
	t.Helper()

	repo := &recordingFeeRepo{}
	h := fees.NewHandler(fees.NewService(repo))
	router := chi.NewRouter()
	router.Route("/v1/admin/fees", h.AdminRoutes())

	req := httptest.NewRequest(http.MethodPut, "/v1/admin/fees/schedule", bytes.NewBufferString(body))
	if callerTenant != "" {
		req = req.WithContext(tenant.WithID(req.Context(), callerTenant))
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Code, repo
}

// TestPutSchedule_PlatformOperatorMaySetGlobalAndAnyTenant covers #382's
// platform-operator half: only a platform identity can change the global
// schedule or target an arbitrary tenant.
func TestPutSchedule_PlatformOperatorMaySetGlobalAndAnyTenant(t *testing.T) {
	body := `{"transfer_fee_bps":100,"conversion_fee_bps":150,"min_fee_amount":"0.1"}`

	code, repo := doPutSchedule(t, "platform", body)
	if code != http.StatusOK {
		t.Fatalf("platform global schedule status = %d, want 200", code)
	}
	if len(repo.sets) != 1 || repo.sets[0].TenantID != nil {
		t.Fatalf("platform global schedule wrote %v, want the global (nil tenant) row", repo.sets)
	}

	code, repo = doPutSchedule(t, "platform", `{"transfer_fee_bps":100,"conversion_fee_bps":150,"min_fee_amount":"0.1","tenant_id":"tenant-2"}`)
	if code != http.StatusOK {
		t.Fatalf("platform tenant schedule status = %d, want 200", code)
	}
	if len(repo.sets) != 1 || repo.sets[0].TenantID == nil || *repo.sets[0].TenantID != "tenant-2" {
		t.Fatalf("platform tenant schedule wrote %v, want tenant-2", repo.sets)
	}
}

// TestPutSchedule_TenantAdminScopedToOwnTenant covers #382's tenant half: the
// tenant always comes from context, never the body, and a tenant admin can
// neither touch the global row nor another tenant's schedule.
func TestPutSchedule_TenantAdminScopedToOwnTenant(t *testing.T) {
	ownBody := `{"transfer_fee_bps":100,"conversion_fee_bps":150,"min_fee_amount":"0.1"}`

	// No tenant_id in the body: the write is scoped to the caller's tenant, so
	// it can never fall through to the global row.
	code, repo := doPutSchedule(t, "tenant-1", ownBody)
	if code != http.StatusOK {
		t.Fatalf("tenant admin own schedule status = %d, want 200", code)
	}
	if len(repo.sets) != 1 || repo.sets[0].TenantID == nil || *repo.sets[0].TenantID != "tenant-1" {
		t.Fatalf("tenant admin wrote %v, want tenant-1", repo.sets)
	}

	// A body tenant_id that matches the caller is accepted.
	code, repo = doPutSchedule(t, "tenant-1", `{"transfer_fee_bps":100,"conversion_fee_bps":150,"min_fee_amount":"0.1","tenant_id":"tenant-1"}`)
	if code != http.StatusOK {
		t.Fatalf("tenant admin matching tenant_id status = %d, want 200", code)
	}
	if len(repo.sets) != 1 || repo.sets[0].TenantID == nil || *repo.sets[0].TenantID != "tenant-1" {
		t.Fatalf("tenant admin wrote %v, want tenant-1", repo.sets)
	}

	// Another tenant's schedule is refused and nothing is persisted.
	code, repo = doPutSchedule(t, "tenant-1", `{"transfer_fee_bps":0,"conversion_fee_bps":0,"min_fee_amount":"0","tenant_id":"tenant-2"}`)
	if code != http.StatusForbidden {
		t.Fatalf("cross-tenant schedule status = %d, want 403", code)
	}
	if len(repo.sets) != 0 {
		t.Fatalf("cross-tenant schedule persisted %d row(s), want 0", len(repo.sets))
	}
}
