package http

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sync"

	"github.com/axiomhq/axiom-go/axiom"
)

const (
	honeyCombPath  = "/honeycomb/1/events/"
	defaultDataset = "axiom-loki-proxy"
	datasetKey     = "_axiom_dataset"
)

type ingestFunc func(ctx context.Context, id string, opts axiom.IngestOptions, events ...axiom.Event) (*axiom.IngestStatus, error)

// implements the http.Server interface
type PushHandler struct {
	sync.Mutex
	ingestFn      ingestFunc
	honeycombAddr string
}

func NewPushHandler(client *axiom.Client, honeycombAddr string) *PushHandler {
	return &PushHandler{
		honeycombAddr: honeycombAddr,
		//ingestFn: client.Datasets.IngestEvents,
	}
}

func PushPath() string {
	return honeyCombPath
}

func handleErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (push *PushHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	push.Lock()
	defer push.Unlock()

	var (
		data     map[string]interface{}
		dataset  = r.URL.Path[len(honeyCombPath):]
		url, err = url.Parse(push.honeycombAddr)
		typ      = r.Header.Get("Content-Type")
	)

	switch typ {
	case "application/json", "application/x-www-form-urlencoded":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		url.Path = path.Join(url.Path, dataset)

		client := &http.Client{}
		newReq, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		newReq.Header = r.Header.Clone()
		resp, err := client.Do(newReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println(resp, err)
	default:
		err = fmt.Errorf("unsupported Content-Type %v", typ)
	}

	fmt.Println(data)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
