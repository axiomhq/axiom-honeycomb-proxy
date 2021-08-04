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
	defaultHoneyCombURL = "https://api.honeycomb.io"
	deploymentURL       = os.Getenv("AXIOM_URL")
	accessToken         = os.Getenv("AXIOM_TOKEN")
	addr                = flag.String("addr", ":8080", "Listen address <ip>:<port>")
)

func main() {
	log.Print("starting axiom-honeycomb-proxy version ", version.Release())

	flag.Parse()

	if deploymentURL == "" {
		deploymentURL = axiom.CloudURL
	}
	if accessToken == "" {
		log.Fatal("missing AXIOM_TOKEN")
	}

	client, err := axiom.NewClient(deploymentURL, accessToken)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("listening on", *addr)
	mp, err := proxy.NewMultiplexer(client, defaultHoneyCombURL)
	if err != nil {
		panic(err)
	}

	server := http.Server{Handler: mp, Addr: *addr}
	log.Fatal(server.ListenAndServe())
}
