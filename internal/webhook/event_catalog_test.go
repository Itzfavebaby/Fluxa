package webhook

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/fluxa/fluxa/internal/domain"
)

// catalogFor drives ListEventCatalog with the given query and decodes the
// response. A nil handler is safe: the catalog is static.
func catalogFor(t *testing.T, query string) []EventCatalogEntry {
	t.Helper()
	path := "/events"
	if query != "" {
		path += "?" + url.Values{"q": []string{query}}.Encode()
	}
	r := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	NewHandler(nil).ListEventCatalog(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var response struct {
		Events []EventCatalogEntry `json:"events"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.Events
}

func TestListEventCatalogReturnsDocumentedExamples(t *testing.T) {
	events := catalogFor(t, "")
	if len(events) == 0 {
		t.Fatal("catalogue returned no supported events")
	}
	for _, event := range events {
		if event.Name == "" || event.Description == "" || !json.Valid(event.Example) {
			t.Errorf("incomplete catalogue entry: %+v", event)
		}
	}
}

// TestEventCatalogAgreesWithSupportedEventTypes is the regression test for the
// drift between the catalogue and SupportedEventTypes. Two hand-maintained
// lists let the catalogue advertise events that RegisterEndpoint rejected, and
// omit events customers could subscribe to. They now share one table, so this
// test pins that agreement rather than a hand-counted set of results.
func TestEventCatalogAgreesWithSupportedEventTypes(t *testing.T) {
	catalogued := make(map[string]struct{}, len(domain.WebhookEvents))
	for _, event := range catalogFor(t, "") {
		catalogued[event.Name] = struct{}{}
	}

	for _, name := range domain.SupportedEventTypes {
		if _, ok := catalogued[name]; !ok {
			t.Errorf("subscribable event %q is missing from the catalogue", name)
		}
	}
	for name := range catalogued {
		if !domain.IsSupportedEventType(name) {
			t.Errorf("catalogued event %q cannot be subscribed to", name)
		}
	}
	if len(catalogued) != len(domain.SupportedEventTypes) {
		t.Errorf("catalogue has %d events, SupportedEventTypes has %d", len(catalogued), len(domain.SupportedEventTypes))
	}
}

// TestEventCatalogDescriptionsAreNonEmpty guards the catalogue's usefulness for
// discovery: an event with no description is subscribable but undiscoverable by
// description search.
func TestEventCatalogDescriptionsAreNonEmpty(t *testing.T) {
	for _, event := range domain.WebhookEvents {
		if event.Description == "" {
			t.Errorf("event %q has no description", event.Name)
		}
		if event.Example == "" || !json.Valid([]byte(event.Example)) {
			t.Errorf("event %q has an invalid example payload %q", event.Name, event.Example)
		}
	}
}

func TestListEventCatalogSearchesNameAndDescription(t *testing.T) {
	// "settled" appears in the transfer.settled name; "claimable" appears in
	// three names; "sanctions" appears only in a description.
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"transfer.settled", []string{"transfer.settled"}},
		{"claimable_balance", []string{"claimable_balance.claimed", "claimable_balance.created", "claimable_balance.expired", "claimable_balance.revoked"}},
		{"sanctions", []string{"sanctions.refresh.failed"}},
		{"no-such-event", nil},
	} {
		events := catalogFor(t, tc.query)
		if len(events) != len(tc.want) {
			var names []string
			for _, e := range events {
				names = append(names, e.Name)
			}
			t.Errorf("q=%q returned %v, want %v", tc.query, names, tc.want)
			continue
		}
		for _, want := range tc.want {
			var found bool
			for _, e := range events {
				if e.Name == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("q=%q did not return %q", tc.query, want)
			}
		}
	}
}

// TestEventCatalogSearchIgnoresExamplePayloadValues pins the documented
// behaviour that search covers the name and description only. Searching for a
// value that appears solely in an example payload must return nothing, or the
// catalogue would leak example payload contents into a search index.
func TestEventCatalogSearchIgnoresExamplePayloadValues(t *testing.T) {
	// "duration_ms" appears in the sanctions example payload but in no name or
	// description.
	if events := catalogFor(t, "duration_ms"); len(events) != 0 {
		t.Errorf("search matched example payload values: %+v", events)
	}
}
