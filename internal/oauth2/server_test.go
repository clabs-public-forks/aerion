package oauth2

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleCallback(t *testing.T) {
	s := NewCallbackServer("good")

	// A stray request with the wrong state is rejected and does not take the slot.
	rec := httptest.NewRecorder()
	s.handleCallback(rec, httptest.NewRequest(http.MethodGet, "/callback?code=x&state=bad", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad state: status %d, want 400", rec.Code)
	}
	if len(s.resultCh) != 0 {
		t.Fatal("bad state consumed the result slot")
	}

	// Error values are HTML-escaped.
	rec = httptest.NewRecorder()
	s.handleCallback(rec, httptest.NewRequest(http.MethodGet, "/callback?state=good&error=%3Cscript%3Ex%3C%2Fscript%3E", nil))
	if body := rec.Body.String(); strings.Contains(body, "<script>x") || !strings.Contains(body, "&lt;script&gt;x") {
		t.Fatal("error value not escaped")
	}
	if got := <-s.resultCh; got.Error != "<script>x</script>" {
		t.Fatalf("result error = %q", got.Error)
	}

	// The real callback is delivered.
	rec = httptest.NewRecorder()
	s.handleCallback(rec, httptest.NewRequest(http.MethodGet, "/callback?code=c&state=good", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("good state: status %d", rec.Code)
	}
	if got := <-s.resultCh; got.Code != "c" {
		t.Fatalf("result code = %q", got.Code)
	}
}
