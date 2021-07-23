package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/axiomhq/axiom-go/axiom"
)

const (
	honeycombEndpoint = "https://api.honeycomb.io"

	// honeycombPathEvents = "/honeycomb/v1/events/"
	// honeycombPathBatch  = "/honeycomb/v1/batch/"
)

var (
	hcurl *url.URL
	proxy *httputil.ReverseProxy
)

func init() {
	hcurl, _ = url.Parse(honeycombEndpoint)
	proxy = httputil.NewSingleHostReverseProxy(hcurl)
}

func GetHandler(client *axiom.Client) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// Try multiplex
		multiplex(res, req)

		// Forward to HC
		forward(res, req)
	}
}

func forward(res http.ResponseWriter, req *http.Request) {
	req.URL.Host = hcurl.Host
	req.URL.Scheme = hcurl.Scheme

	proxy.ServeHTTP(res, req)
}

func multiplex(res http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/1/events/") {
		multiplexEvents(res, req)
	} else if strings.HasPrefix(req.URL.Path, "/1/batch/") {
		multiplexBatch(res, req)
	} else {
		return
	}
}

func multiplexEvents(res http.ResponseWriter, req *http.Request) {

}

func multiplexBatch(res http.ResponseWriter, req *http.Request) {

}
