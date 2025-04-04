package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strings"
	"unsafe"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/axiom-go/axiom/ingest"
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

	// type check on axiom.Event incase it's ever not a map[string]interface{}
	// so we can use unsafe.Pointer for a quick type conversion instead of allocating a new slice
	if !reflect.TypeOf(axiom.Event{}).ConvertibleTo(reflect.TypeOf(map[string]any{})) {
		panic("axiom.Event is not a map[string]interface{}, please contact support")
	}
}

func Decompress(rdr io.ReadCloser, encoding string) (io.ReadCloser, error) {
	switch encoding {
	case "gzip":
		decomp, err := gzip.NewReader(rdr)
		if err != nil {
			return nil, err
		}
		return decomp, nil
	case "zstd":
		decomp, err := zstd.NewReader(rdr)
		if err != nil {
			return nil, err
		}
		return decomp.IOReadCloser(), nil
	default:
		return rdr, nil
	}
}

type hcServer struct {
	proxy *httputil.ReverseProxy
	URL   *url.URL
}

func (hcSrv *hcServer) Host() string {
	return hcSrv.URL.Host
}

func (hcSrv *hcServer) Scheme() string {
	return hcSrv.URL.Scheme
}

type Multiplexer struct {
	client   *axiom.Client
	hcServer *hcServer
}

func NewMultiplexer(client *axiom.Client, honeycombEndpoint string) (*Multiplexer, error) {
	var hcSrv *hcServer
	if honeycombEndpoint != "" {
		hcURL, err := url.Parse(honeycombEndpoint)
		if err != nil {
			return nil, err
		}
		proxy := httputil.NewSingleHostReverseProxy(hcURL)
		hcSrv = &hcServer{proxy: proxy, URL: hcURL}
	}
	return &Multiplexer{
		client:   client,
		hcServer: hcSrv,
	}, nil
}

func (m *Multiplexer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if m.hcServer != nil {
		body := bytes.NewBuffer(nil)
		req.Body = io.NopCloser(io.TeeReader(req.Body, body))
		m.forward(resp, req)
		req.Body = io.NopCloser(body)
	}

	if err := m.multiplex(req); err != nil {
		logger.Error(err.Error())
		if m.hcServer == nil {
			if _, wErr := resp.Write([]byte(err.Error())); wErr != nil {
				logger.Error(wErr.Error())
			}
		}
	}

	if m.hcServer == nil {
		if _, wErr := resp.Write([]byte("{}")); wErr != nil {
			logger.Error(wErr.Error())
		}
	}
}

func (m *Multiplexer) forward(resp http.ResponseWriter, req *http.Request) {
	req.URL.Host = m.hcServer.Host()
	req.URL.Scheme = m.hcServer.Scheme()
	m.hcServer.proxy.ServeHTTP(resp, req)
}

func (m *Multiplexer) multiplex(req *http.Request) error {
	if req.Method != "POST" {
		return nil
	}

	body, err := Decompress(req.Body, req.Header.Get("Content-Encoding"))
	if err != nil {
		return err
	}
	req.Body = body
	defer req.Body.Close()

	switch {
	case strings.HasPrefix(req.URL.Path, "/1/events/"):
		fallthrough
	case strings.HasPrefix(req.URL.Path, "/1/batch/"):
		events, dataset, err := RequestToEvents(req)
		if err != nil {
			return err
		}
		return m.sendEvents(req.Context(), dataset, events...)
	}
	return nil
}

func getDatasetFromRequest(req *http.Request) (dataset string) {
	splitStr := strings.Split(req.URL.Path, "/")
	if len(splitStr) > 0 {
		dataset = splitStr[len(splitStr)-1]
	}
	return
}

func RequestToEvents(req *http.Request) (events []axiom.Event, dataset string, err error) {
	dataset = getDatasetFromRequest(req)
	if dataset == "" {
		err = errors.New("no dataset specified")
		return
	}

	var v any
	switch req.Header.Get("Content-Type") {
	case "application/msgpack":
		if err := msgpack.NewDecoder(req.Body).Decode(&v); err != nil {
			return nil, "", err
		}
	default:
		if err := json.NewDecoder(req.Body).Decode(&v); err != nil {
			return nil, "", err
		}
	}

	switch ev := v.(type) {
	case map[string]any:
		timeStr := req.Header.Get("X-Honeycomb-Event-Time")
		if strings.TrimSpace(timeStr) != "" {
			ev[ingest.TimestampField] = timeStr
		}
		events = append(events, ev)
	case []map[string]any:
		// NOTE: Breaks if axiom.Event is ever not a map[string]interface{} (see init)
		events = *(*[]axiom.Event)(unsafe.Pointer(&ev))
		for _, event := range events {
			if timeStr, ok := event["time"].(string); ok {
				event[ingest.TimestampField] = timeStr
				delete(event, "time")
			}
		}
	case []any:
		// manually parse each item as an event, we can't know for sure that every item will be a map, json is fun.

		for index := range ev {
			event, ok := ev[index].(map[string]any)
			if !ok {
				return nil, "", fmt.Errorf("unexpected event type %T (%+v)", v, v)
			}

			if timeStr, ok := event["time"].(string); ok {
				event[ingest.TimestampField] = timeStr
				delete(event, "time")
			}
			events = append(events, event)
		}
	default:
		return nil, "", fmt.Errorf("unexpected event type %T (%+v)", v, v)
	}

	return
}

func (m *Multiplexer) sendEvents(ctx context.Context, dataset string, events ...axiom.Event) error {
	status, err := m.client.Datasets.IngestEvents(ctx, dataset, events)
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
