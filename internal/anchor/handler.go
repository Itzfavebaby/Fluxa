package anchor

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/fluxa/fluxa/internal/api"
	"github.com/fluxa/fluxa/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	registry *Registry
}

func NewHandler(registry *Registry) *Handler {
	return &Handler{registry: registry}
}

// AdminRoutes mounts anchor registration/listing under whatever prefix the
// caller mounts it at (e.g. "/admin/anchors"), so both land on
// POST/GET /v1/admin/anchors. Registration is a platform-wide mutation — the
// anchors table has no tenant_id, so a registered anchor (home domain,
// transfer server, SEP-10 signing key) is visible to and usable by every
// tenant — and therefore takes the caller's guard (RequirePlatformOperator)
// so only a platform operator can register one (#384).
func (h *Handler) AdminRoutes(guard func(http.Handler) http.Handler) func(r chi.Router) {
	return func(r chi.Router) {
		post := r.Post
		if guard != nil {
			post = r.With(guard).Post
		}
		post("/", h.register)
		r.Get("/", h.list)
	}
}

type registerAnchorRequest struct {
	HomeDomain string `json:"home_domain" validate:"required"`
}

type anchorResponse struct {
	ID                  string   `json:"id"`
	HomeDomain          string   `json:"home_domain"`
	TransferServer      string   `json:"transfer_server,omitempty"`
	TransferServerSep24 string   `json:"transfer_server_sep24,omitempty"`
	WebAuthEndpoint     string   `json:"web_auth_endpoint,omitempty"`
	SupportedAssets     []string `json:"supported_assets"`
	SepVersions         []int    `json:"sep_versions"`
	RegisteredAt        string   `json:"registered_at"`
}

func toAnchorResponse(a *domain.Anchor) anchorResponse {
	return anchorResponse{
		ID:                  a.ID,
		HomeDomain:          a.HomeDomain,
		TransferServer:      a.TransferServer,
		TransferServerSep24: a.TransferServerSep24,
		WebAuthEndpoint:     a.WebAuthEndpoint,
		SupportedAssets:     a.SupportedAssets,
		SepVersions:         a.SepVersions,
		RegisteredAt:        a.RegisteredAt.Format(time.RFC3339),
	}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerAnchorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}
	if err := api.Validate(req); err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	a, err := h.registry.Register(r.Context(), req.HomeDomain)
	if err != nil {
		log.Error().Err(err).Msg("anchor registration failed")
		api.Error(w, http.StatusBadGateway, "ANCHOR_REGISTRATION_FAILED", "anchor registration failed")
		return
	}

	api.JSON(w, http.StatusCreated, toAnchorResponse(a))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	anchors := h.registry.List()
	out := make([]anchorResponse, 0, len(anchors))
	for _, a := range anchors {
		out = append(out, toAnchorResponse(a))
	}
	api.JSON(w, http.StatusOK, map[string]interface{}{"anchors": out})
}
