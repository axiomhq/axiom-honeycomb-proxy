package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/axiomhq/axiom-go/axiom"

	"github.com/tj/assert"
)

func dummyIngest(ctx context.Context, id string, opts axiom.IngestOptions, events ...axiom.Event) (*axiom.IngestStatus, error) {
	fmt.Println(events)
	return nil, nil
}

var dummyStreams = map[string]interface{}{
	"streams": []map[string]interface{}{
		map[string]interface{}{
			"stream": map[string]string{
				"label1": "value1",
				"label2": "value2",
			},
			"values": [][2]string{
				[2]string{"1", "hello world"},
				[2]string{"2", "the answer is 42"},
				[2]string{"3", "foobar"},
			},
		},
	},
}

func TestMyHandler(t *testing.T) {
	push := &PushHandler{
		ingestFn: dummyIngest,
	}

	server := httptest.NewServer(push)
	defer server.Close()

	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(dummyStreams)
	assert.NoError(t, err)

	resp, err := http.Post(server.URL, "application/json", buf)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.EqualValues(t, resp.StatusCode, 200)
}
