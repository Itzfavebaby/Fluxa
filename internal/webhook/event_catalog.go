package webhook

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/fluxa/fluxa/internal/api"
	"github.com/fluxa/fluxa/internal/domain"
)

// EventCatalogEntry is one event as presented by GET /events. The wire shape is
// part of the public contract and is asserted by the event catalogue tests, so
// it is declared separately from the domain table it is rendered from.
type EventCatalogEntry struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Example     json.RawMessage `json:"example"`
}

// catalogEntries renders the domain event contract into wire entries. It runs
// on every request and the source table is small, so the copy is cheap; it
// exists to keep handlers from handing out a mutable view of package state.
func catalogEntries() []EventCatalogEntry {
	source := domain.WebhookEvents
	entries := make([]EventCatalogEntry, 0, len(source))
	for _, event := range source {
		entries = append(entries, EventCatalogEntry{
			Name:        event.Name,
			Description: event.Description,
			Example:     json.RawMessage(event.Example),
		})
	}
	return entries
}

// ListEventCatalog returns the event contract catalogue. Search is performed
// over the event name and description, never over example payload values.
//
// Entries come from domain.WebhookEvents, so an event is advertised here exactly
// when RegisterEndpoint will accept it. This handler used to iterate its own
// hand-maintained slice, which drifted from SupportedEventTypes and advertised
// events the API rejected.
func (h *Handler) ListEventCatalog(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	entries := catalogEntries()
	filtered := make([]EventCatalogEntry, 0, len(entries))
	for _, entry := range entries {
		if query == "" ||
			strings.Contains(strings.ToLower(entry.Name), query) ||
			strings.Contains(strings.ToLower(entry.Description), query) {
			filtered = append(filtered, entry)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	api.JSON(w, http.StatusOK, map[string]interface{}{"events": filtered})
}
