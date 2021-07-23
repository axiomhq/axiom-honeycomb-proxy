package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/klauspost/compress/zstd"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

func decompress(rdr io.ReadCloser, encoding string) (io.ReadCloser, error) {
	switch encoding {
	case "gzip":
		decomp, err := gzip.NewReader(rdr)
		if err != nil {
			return nil, err
		}
		defer decomp.Close()
		return decomp, nil
	case "zstd":
		decomp, err := zstd.NewReader(rdr)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(decomp), nil
	default:
		return rdr, nil
	}
}

type Multiplexer struct {
	client *axiom.Client
	proxy  *httputil.ReverseProxy
	hcURL  *url.URL
}

func NewMultiplexer(client *axiom.Client, honeycombEndpoint string) (*Multiplexer, error) {
	hcURL, err := url.Parse(honeycombEndpoint)
	if err != nil {
		return nil, err
	}
	return &Multiplexer{
		client: client,
		proxy:  httputil.NewSingleHostReverseProxy(hcURL),
		hcURL:  hcURL,
	}, nil
}

func (m *Multiplexer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	body := bytes.NewBuffer(nil)
	req.Body = io.NopCloser(io.TeeReader(req.Body, body))
	m.forward(resp, req)
	req.Body = io.NopCloser(body)
	if err := m.multiplex(req); err != nil {
		logger.Error(err.Error())
	}
}

func (m *Multiplexer) forward(resp http.ResponseWriter, req *http.Request) {
	req.URL.Host = m.hcURL.Host
	req.URL.Scheme = m.hcURL.Scheme
	m.proxy.ServeHTTP(resp, req)
}

func (m *Multiplexer) multiplex(req *http.Request) error {
	if req.Method != "POST" {
		return nil
	}

	body, err := decompress(req.Body, req.Header.Get("Content-Encoding"))
	if err != nil {
		return err
	}
	req.Body = body
	defer req.Body.Close()

	switch {
	case strings.HasPrefix(req.URL.Path, "/1/events/"):
		return m.multiplexEvents(req)
	case strings.HasPrefix(req.URL.Path, "/1/batch/"):
		return m.multiplexBatch(req)
	default:
		return nil
	}
}

func (m *Multiplexer) multiplexEvents(req *http.Request) error {
	splitStr := strings.Split(req.URL.Path, "/")
	dataset := splitStr[len(splitStr)-1]

	ev := axiom.Event{}

	switch req.Header.Get("Content-Type") {
	case "application/msgpack":
		if err := msgpack.NewDecoder(req.Body).Decode(&ev); err != nil {
			return err
		}
	default:
		if err := json.NewDecoder(req.Body).Decode(&ev); err != nil {
			return err
		}
	}

	timeStr := req.Header.Get("X-Honeycomb-Event-Time")
	if strings.TrimSpace(timeStr) != "" {
		ev["_time"] = timeStr
	}
	return m.sendEvents(req.Context(), dataset, ev)
}

func (m *Multiplexer) multiplexBatch(req *http.Request) error {
	splitStr := strings.Split(req.URL.Path, "/")
	dataset := splitStr[len(splitStr)-1]

	data := make([]map[string]interface{}, 0)

	switch req.Header.Get("Content-Type") {
	case "application/msgpack":
		if err := msgpack.NewDecoder(req.Body).Decode(&data); err != nil {
			return err
		}
	default:
		if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
			return err
		}
	}

	events := make([]axiom.Event, len(data))
	for i, d := range data {
		events[i] = d["data"].(map[string]interface{})
		if timeStr, ok := d["time"].(string); ok {
			events[i]["_time"] = timeStr
		}
	}
	return m.sendEvents(req.Context(), dataset, events...)
}

func (m *Multiplexer) sendEvents(ctx context.Context, dataset string, events ...axiom.Event) error {
	opts := axiom.IngestOptions{}

	status, err := m.client.Datasets.IngestEvents(ctx, dataset, opts, events...)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(status); err != nil {
		return err
	}

	logger.Info(buf.String())
	return nil
}
