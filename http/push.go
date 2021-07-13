package http

import (
	"bytes"
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
	rdr, err := eh.forward(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ev := &axiom.Event{}
	json.NewDecoder(rdr).Decode(ev)
	fmt.Println(ev)
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
	rdr, err := bh.forward(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ev := &axiom.Event{}
	json.NewDecoder(rdr).Decode(ev)
	fmt.Println(ev)
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

func (push *pushHandler) forward(r *http.Request) (io.Reader, error) {
	push.Lock()
	defer push.Unlock()

	splitStr := strings.Split(r.URL.Path, "/")
	if len(splitStr) != 5 {
		return nil, fmt.Errorf("invalid path %s", r.URL.Path)
	}
	dataset := splitStr[4]
	apiUrl := *push.apiUrl
	apiUrl.Path = path.Join(apiUrl.Path, dataset)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	newReq, err := http.NewRequest("POST", apiUrl.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	//newReq.Header = r.Header.Clone()
	if _, err := push.httpClient.Do(newReq); err != nil {
		return nil, err
	}

	return bytes.NewBuffer(body), nil
}
