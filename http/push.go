package http

import (
	"bytes"
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
)

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
	if _, err := rh.forward(r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
	rdr, err := eh.forward(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	splitStr := strings.Split(r.URL.Path, "/")
	dataset := splitStr[len(splitStr)-1]

	ev := axiom.Event{}
	if err := json.NewDecoder(rdr).Decode(&ev); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	timeStr := r.Header.Get("X-Honeycomb-Event-Time")
	if strings.TrimSpace(timeStr) != "" {
		ev["_time"] = timeStr
	}
	eh.multiplex(r.Context(), w, dataset, ev)
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
	rdr, err := bh.forward(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	splitStr := strings.Split(r.URL.Path, "/")
	dataset := splitStr[len(splitStr)-1]

	data := make([]map[string]interface{}, 0)
	if err := json.NewDecoder(rdr).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	events := make([]axiom.Event, len(data))
	for i, d := range data {
		events[i] = d["data"].(map[string]interface{})
		if timeStr, ok := d["time"].(string); ok {
			events[i]["_time"] = timeStr
		}
	}

	bh.multiplex(r.Context(), w, dataset, events...)
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

func (push *pushHandler) forward(r *http.Request) (io.Reader, error) {
	push.Lock()
	defer push.Unlock()

	apiURL := *push.apiURL
	apiURL.Path = r.URL.Path

	body := bytes.NewBuffer(nil)

	newReq, err := http.NewRequest("POST", apiURL.String(), io.TeeReader(r.Body, body))
	if err != nil {
		return nil, err
	}

	newReq.Header = r.Header.Clone()
	resp, err := push.httpClient.Do(newReq)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return body, nil
}

func (push *pushHandler) multiplex(ctx context.Context, w http.ResponseWriter, dataset string, data ...axiom.Event) {
	opts := axiom.IngestOptions{}

	status, err := push.client.Datasets.IngestEvents(ctx, dataset, opts, data...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (push *pushHandler) Path() string {
	return push.path
}
