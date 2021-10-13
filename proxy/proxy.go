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
	if !reflect.TypeOf(axiom.Event{}).ConvertibleTo(reflect.TypeOf(map[string]interface{}{})) {
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

	var v interface{}
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
	case map[string]interface{}:
		timeStr := req.Header.Get("X-Honeycomb-Event-Time")
		if strings.TrimSpace(timeStr) != "" {
			ev["_time"] = timeStr
		}
		events = append(events, ev)
	case []map[string]interface{}:
		// NOTE: Breaks if axiom.Event is ever not a map[string]interface{} (see init)
		events = *(*[]axiom.Event)(unsafe.Pointer(&ev))
		for _, event := range events {
			if timeStr, ok := event["time"].(string); ok {
				event["_time"] = timeStr
			}
		}
	case []interface{}:
		// manually parse each item as an event, we can't know for sure that every item will be a map, json is fun.

		for index := range ev {
			event, ok := ev[index].(map[string]interface{})
			if ok == false {
				return nil, "", fmt.Errorf("unexpected event type %T (%+v)", v, v)
			}

			if timeStr, ok := event["time"].(string); ok {
				event["_time"] = timeStr
			}
			events = append(events, event)
		}
	default:
		return nil, "", fmt.Errorf("unexpected event type %T (%+v)", v, v)
	}

	return
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
