package contact

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGoogleSyncRepeatsSyncTokenOnEveryPage checks that an incremental sync
// sends the (escaped) syncToken with every page, not just the first, since
// the People API requires paginated calls to repeat the first call's params.
func TestGoogleSyncRepeatsSyncTokenOnEveryPage(t *testing.T) {
	const token = "tok+en/with=chars&x"
	var pages int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		q := r.URL.Query()
		if got := q.Get("syncToken"); got != token {
			t.Errorf("page %d syncToken = %q, want %q", pages, got, token)
		}
		if q.Get("x") != "" {
			t.Errorf("page %d: unescaped token leaked a query param", pages)
		}
		switch q.Get("pageToken") {
		case "":
			_, _ = w.Write([]byte(`{"connections":[],"nextPageToken":"p&2"}`))
		case "p&2":
			_, _ = w.Write([]byte(`{"connections":[],"nextSyncToken":"next"}`))
		default:
			t.Errorf("unexpected pageToken %q", q.Get("pageToken"))
		}
	}))
	defer srv.Close()
	old := googleConnectionsURL
	googleConnectionsURL = srv.URL
	t.Cleanup(func() { googleConnectionsURL = old })

	res, err := NewGoogleContactsSyncer().SyncContactsDelta("access", token)
	if err != nil {
		t.Fatal(err)
	}
	if pages != 2 || res.NextSyncToken != "next" || res.IsFullSync {
		t.Errorf("pages = %d, result = %+v; want 2 pages, NextSyncToken next, incremental", pages, res)
	}
}
