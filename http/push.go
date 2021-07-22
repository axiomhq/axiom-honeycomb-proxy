package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

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

type RootHandler struct {
	*pushHandler
}

func NewRootHandler(client *axiom.Client, apiURL string) (*RootHandler, error) {
	push, err := newPushHandler(apiURL, "/", client)
	if err != nil {
		return nil, err
	}
	return &RootHandler{
		pushHandler: push,
	}, nil
}

func (rh *RootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := rh.forward(w, r); err != nil {
		logger.Error(err.Error())
	}
}

type EventHandler struct {
	*pushHandler
}

func NewEventHandler(client *axiom.Client, apiURL string) (*EventHandler, error) {
	push, err := newPushHandler(apiURL, "/1/events/", client)
	if err != nil {
		return nil, err
	}
	return &EventHandler{
		pushHandler: push,
	}, nil
}

func (eh *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rdr, err := eh.forward(w, r)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	splitStr := strings.Split(r.URL.Path, "/")
	dataset := splitStr[len(splitStr)-1]

	ev := axiom.Event{}

	switch r.Header.Get("Content-Type") {
	case "application/msgpack":
		if err := msgpack.NewDecoder(rdr).Decode(&ev); err != nil {
			logger.Error(err.Error())
			return
		}
	default:
		if err := json.NewDecoder(rdr).Decode(&ev); err != nil {
			logger.Error(err.Error())
			return
		}
	}

	timeStr := r.Header.Get("X-Honeycomb-Event-Time")
	if strings.TrimSpace(timeStr) != "" {
		ev["_time"] = timeStr
	}
	if err := eh.multiplex(r.Context(), dataset, ev); err != nil {
		logger.Error(err.Error())
	}
}

type BatchHandler struct {
	*pushHandler
}

func NewBatchHandler(client *axiom.Client, apiURL string) (*BatchHandler, error) {
	push, err := newPushHandler(apiURL, "/1/batch/", client)
	if err != nil {
		return nil, err
	}
	return &BatchHandler{
		pushHandler: push,
	}, nil
}

func (bh *BatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rdr, err := bh.forward(w, r)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	splitStr := strings.Split(r.URL.Path, "/")
	dataset := splitStr[len(splitStr)-1]

	data := make([]map[string]interface{}, 0)

	switch r.Header.Get("Content-Type") {
	case "application/msgpack":
		if err := msgpack.NewDecoder(rdr).Decode(&data); err != nil {
			logger.Error(err.Error())
			return
		}
	default:
		if err := json.NewDecoder(rdr).Decode(&data); err != nil {
			logger.Error(err.Error())
			return
		}
	}

	events := make([]axiom.Event, len(data))
	for i, d := range data {
		events[i] = d["data"].(map[string]interface{})
		if timeStr, ok := d["time"].(string); ok {
			events[i]["_time"] = timeStr
		}
	}

	if err := bh.multiplex(r.Context(), dataset, events...); err != nil {
		logger.Error(err.Error())
	}
}

type pushHandler struct {
	sync.Mutex
	client     *axiom.Client
	apiURL     *url.URL
	path       string
	httpClient *http.Client
}

func newPushHandler(addr string, apiPath string, client *axiom.Client) (*pushHandler, error) {
	apiURL, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	apiURL.Path = path.Join(apiURL.Path, apiPath)

	return &pushHandler{
		apiURL:     apiURL,
		client:     client,
		httpClient: &http.Client{},
		path:       apiPath,
	}, nil
}

func (push *pushHandler) forward(w http.ResponseWriter, r *http.Request) (io.Reader, error) {
	push.Lock()
	defer push.Unlock()

	apiURL := *push.apiURL
	apiURL.Path = r.URL.Path

	body := bytes.NewBuffer(nil)

	newReq, err := http.NewRequest("POST", apiURL.String(), io.TeeReader(r.Body, body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}

	newReq.Header = r.Header.Clone()
	resp, err := push.httpClient.Do(newReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var d io.Reader
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		decomp, err := gzip.NewReader(body)
		if err != nil {
			return nil, err
		}
		defer decomp.Close()
		d = decomp
	case "zstd":
		decomp, err := zstd.NewReader(body)
		if err != nil {
			return nil, err
		}
		defer decomp.Close()
		d = decomp
	default:
		d = body
	}

	return d, nil
}

func (push *pushHandler) multiplex(ctx context.Context, dataset string, data ...axiom.Event) error {
	opts := axiom.IngestOptions{}

	status, err := push.client.Datasets.IngestEvents(ctx, dataset, opts, data...)
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

func (push *pushHandler) Path() string {
	return push.path
}
