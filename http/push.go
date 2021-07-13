package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/axiomhq/axiom-go/axiom"
)

type EventHandler struct {
	*pushHandler
}

func NewEventHandler(client *axiom.Client, apiUrl string) (*EventHandler, error) {
	push, err := newPushHandler(apiUrl, "1/events/", client)
	if err != nil {
		return nil, err
	}
	return &EventHandler{
		pushHandler: push,
	}, nil
}

func (eh *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	timeStr := r.Header.Get("X-Honeycomb-Event-Time")
	dataset, rdr, err := eh.forward(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ev := axiom.Event{}
	if err := json.NewDecoder(rdr).Decode(&ev); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	ev["_time"] = timeStr
	eh.multiplex(r.Context(), w, dataset, ev)
}

type BatchHandler struct {
	*pushHandler
}

func NewBatchHandler(client *axiom.Client, apiUrl string) (*BatchHandler, error) {
	push, err := newPushHandler(apiUrl, "1/batch/", client)
	if err != nil {
		return nil, err
	}
	return &BatchHandler{
		pushHandler: push,
	}, nil
}

func (bh *BatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dataset, rdr, err := bh.forward(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

// implements the http.Server interface
type pushHandler struct {
	sync.Mutex
	client     *axiom.Client
	apiUrl     *url.URL
	httpClient *http.Client
}

func newPushHandler(addr string, apiPath string, client *axiom.Client) (*pushHandler, error) {
	apiUrl, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	apiUrl.Path = path.Join(apiUrl.Path, apiPath)
	return &pushHandler{
		apiUrl:     apiUrl,
		client:     client,
		httpClient: &http.Client{},
	}, nil
}

func (push *pushHandler) forward(r *http.Request) (string, io.Reader, error) {
	push.Lock()
	defer push.Unlock()

	splitStr := strings.Split(r.URL.Path, "/")
	if len(splitStr) != 5 {
		return "", nil, fmt.Errorf("invalid path %s", r.URL.Path)
	}
	dataset := splitStr[4]
	apiUrl := *push.apiUrl
	apiUrl.Path = path.Join(apiUrl.Path, dataset)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", nil, err
	}

	newReq, err := http.NewRequest("POST", apiUrl.String(), bytes.NewBuffer(body))
	if err != nil {
		return "", nil, err
	}

	//newReq.Header = r.Header.Clone()
	if _, err := push.httpClient.Do(newReq); err != nil {
		return "", nil, err
	}

	return dataset, bytes.NewBuffer(body), nil
}

func (push *pushHandler) multiplex(
	ctx context.Context,
	w http.ResponseWriter,
	dataset string,
	data ...axiom.Event) {

	opts := axiom.IngestOptions{}

	status, err := push.client.Datasets.IngestEvents(
		ctx,
		dataset,
		opts,
		data...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
