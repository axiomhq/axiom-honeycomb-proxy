package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/pkg/version"

	httpProxy "github.com/axiomhq/axiom-honeycomb-proxy/http"
)

const (
	honeycombPathEvents = "/honeycomb/v1/events/"
	honeycombPathBatch  = "/honeycomb/v1/batch/"
)

var (
	deploymentURL     = os.Getenv("AXIOM_DEPLOYMENT_URL")
	accessToken       = os.Getenv("AXIOM_ACCESS_TOKEN")
	addr              = flag.String("addr", ":3111", "Listen address <ip>:<port>")
	honeycombEndpoint = flag.String("honeycomb", "https://api.honeycomb.io", "Honeycomb api url")
)

func main() {
	log.Print("starting axiom-honeycomb-proxy version ", version.Release())

	flag.Parse()

	if deploymentURL == "" {
		log.Fatal("missing AXIOM_DEPLOYMENT_URL")
	}
	if accessToken == "" {
		log.Fatal("missing AXIOM_ACCESS_TOKEN")
	}

	client, err := axiom.NewClient(deploymentURL, accessToken)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	singleEventHandler, err := httpProxy.NewEventHandler(client, *honeycombEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	batchEventHandler, err := httpProxy.NewBatchHandler(client, *honeycombEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	mux.Handle(honeycombPathEvents, singleEventHandler)
	mux.Handle(honeycombPathBatch, batchEventHandler)

	log.Print("listening on", *addr)

	server := http.Server{Handler: mux, Addr: *addr}
	log.Fatal(server.ListenAndServe())
}
