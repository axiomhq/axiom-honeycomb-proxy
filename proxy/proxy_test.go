package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/axiomhq/axiom-go/axiom"
)

func forward(t *testing.T, req *http.Request) *http.Request {
	t.Helper()

	reqs := make(chan *http.Request, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		reqs <- r
	}))
	t.Cleanup(upstream.Close)

	mp, err := NewMultiplexer(nil, upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	mp.ServeHTTP(httptest.NewRecorder(), req)

	select {
	case got := <-reqs:
		return got
	default:
		t.Fatal("request did not reach upstream")
		return nil
	}
}

func TestMultiplexerForwardsToHoneycomb(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://proxy.example/1/events/my-dataset?x=1", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	got := forward(t, req)

	if got.URL.Path != "/1/events/my-dataset" || got.URL.RawQuery != "x=1" {
		t.Errorf("upstream URL = %s, want /1/events/my-dataset?x=1", got.URL)
	}
	if got.Host != "proxy.example" {
		t.Errorf("upstream Host = %q, want the inbound host %q", got.Host, "proxy.example")
	}
	if xff, want := got.Header.Get("X-Forwarded-For"), "203.0.113.7, 192.0.2.1"; xff != want {
		t.Errorf("upstream X-Forwarded-For = %q, want %q", xff, want)
	}
}

func TestMultiplexerDropsHopByHopForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://proxy.example/1/events/my-dataset", nil)
	req.Header.Set("Connection", "keep-alive, X-Forwarded-For")
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	got := forward(t, req)

	if xff, want := got.Header.Get("X-Forwarded-For"), "192.0.2.1"; xff != want {
		t.Errorf("upstream X-Forwarded-For = %q, want %q", xff, want)
	}
}

func TestMultiplexerReportsErrorWithoutHoneycomb(t *testing.T) {
	mp, err := NewMultiplexer(nil, "")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/1/events/my-dataset", strings.NewReader("not json"))
	resp := httptest.NewRecorder()
	mp.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	if body := strings.TrimSpace(resp.Body.String()); body == "" || strings.HasSuffix(body, "{}") {
		t.Errorf("body = %q, want the error text only", body)
	}
}

func TestMultiplexerReportsIngestFailureWithoutHoneycomb(t *testing.T) {
	axiomSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(axiomSrv.Close)

	client, err := axiom.NewClient(
		axiom.SetNoEnv(),
		axiom.SetNoRetry(),
		axiom.SetNoTracing(),
		axiom.SetURL(axiomSrv.URL),
		axiom.SetAPITokenConfig("xaat-test"),
	)
	if err != nil {
		t.Fatal(err)
	}

	mp, err := NewMultiplexer(client, "")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/1/events/my-dataset", strings.NewReader(`{"foo":"bar"}`))
	resp := httptest.NewRecorder()
	mp.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", resp.Code, http.StatusBadGateway)
	}
	if body := resp.Body.String(); !strings.HasPrefix(body, "ingest events: ") {
		t.Errorf("body = %q, want the ingest error", body)
	}
}

func TestGetDatasetFromRequest(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/1/events/my-dataset", "my-dataset"},
		{"/1/events/", ""},
		{"my-dataset", "my-dataset"},
	}
	for _, tt := range tests {
		req := &http.Request{URL: &url.URL{Path: tt.path}}
		if got := getDatasetFromRequest(req); got != tt.want {
			t.Errorf("getDatasetFromRequest(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
