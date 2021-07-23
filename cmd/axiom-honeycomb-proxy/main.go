package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/pkg/version"

	"github.com/axiomhq/axiom-honeycomb-proxy/proxy"
)

var (
	deploymentURL = os.Getenv("AXIOM_URL")
	accessToken   = os.Getenv("AXIOM_TOKEN")
	addr          = flag.String("addr", ":3111", "Listen address <ip>:<port>")
)

func main() {
	log.Print("starting axiom-honeycomb-proxy version ", version.Release())

	flag.Parse()

	if deploymentURL == "" {
		log.Fatal("missing AXIOM_URL")
	}
	if accessToken == "" {
		log.Fatal("missing AXIOM_TOKEN")
	}

	client, err := axiom.NewClient(deploymentURL, accessToken)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("listening on", *addr)

	server := http.Server{Handler: proxy.GetHandler(client), Addr: *addr}
	log.Fatal(server.ListenAndServe())
}
