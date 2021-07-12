package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/axiomhq/axiom-go/axiom"
	httpProxy "github.com/axiomhq/axiom-honeycomb-proxy/http"
)

func initHttpPushHandler(mux *http.ServeMux, client *axiom.Client, honeycombAddr string) {
	handler := httpProxy.NewPushHandler(client, honeycombAddr)
	mux.Handle(httpProxy.PushPath(), handler)
}

func main() {
	var (
		//deploymentURL = os.Getenv("AXM_DEPLOYMENT_URL")
		//accessToken   = os.Getenv("AXM_ACCESS_TOKEN")
		addr          = flag.String("addr", ":3111", "a string <ip>:<port>")
		honeycombAddr = flag.String("hcaddr", "https://api.honeycomb.io/1/events", "honeycomb api endpoint")
	)

	//client, err := axiom.NewClient(deploymentURL, accessToken)
	//if err != nil {
	//	log.Fatal(err)
	//}

	mux := http.NewServeMux()
	initHttpPushHandler(mux, nil, *honeycombAddr)

	log.Printf("Now listening on %s...\n", *addr)
	server := http.Server{Handler: mux, Addr: *addr}
	log.Fatal(server.ListenAndServe())
}
