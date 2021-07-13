package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/axiomhq/axiom-go/axiom"
	httpProxy "github.com/axiomhq/axiom-honeycomb-proxy/http"
)

const (
	honeycombPathEvents = "/honeycomb/v1/events/"
	honeycombPathBatch  = "/honeycomb/v1/batch/"
)

func initHttpPushHandler(mux *http.ServeMux, client *axiom.Client, addr string) error {
	h1, err := httpProxy.NewEventHandler(client, addr)
	if err != nil {
		return err
	}
	h2, err := httpProxy.NewBatchHandler(client, addr)
	if err != nil {
		return err
	}

	mux.Handle(honeycombPathEvents, h1)
	mux.Handle(honeycombPathBatch, h2)
	return nil
}

func main() {
	var (
		//deploymentURL = os.Getenv("AXM_DEPLOYMENT_URL")
		//accessToken   = os.Getenv("AXM_ACCESS_TOKEN")
		addr              = flag.String("addr", ":3111", "a string <ip>:<port>")
		honeycombEndpoint = flag.String("honeycomb", "https://api.honeycomb.io", "honeycomb api endpoint")
	)

	//client, err := axiom.NewClient(deploymentURL, accessToken)
	//if err != nil {
	//	log.Fatal(err)
	//}

	mux := http.NewServeMux()
	if err := initHttpPushHandler(mux, nil, *honeycombEndpoint); err != nil {
		panic(err)
	}

	log.Printf("Now listening on %s...\n", *addr)
	server := http.Server{Handler: mux, Addr: *addr}
	log.Fatal(server.ListenAndServe())
}
