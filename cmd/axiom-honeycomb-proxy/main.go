package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/pkg/http"
	"github.com/axiomhq/pkg/version"

	"github.com/axiomhq/axiom-honeycomb-proxy/proxy"
)

const (
	exitOK int = iota
	exitConfig
	exitInternal
)

const defaultHoneyCombURL = "https://api.honeycomb.io"

var addr = flag.String("addr", ":8080", "Listen address <ip>:<port>")

func main() {
	os.Exit(Main())
}

func Main() int {
	// Export `AXIOM_TOKEN` and `AXIOM_ORG_ID` for Axiom Cloud
	// Export `AXIOM_URL` and `AXIOM_TOKEN` for Axiom Selfhost

	log.Print("starting axiom-honeycomb-proxy version ", version.Release())

	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt,
		os.Kill,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	defer cancel()

	client, err := axiom.NewClient()
	if err != nil {
		log.Print(err)
		return exitConfig
	} else if err = client.ValidateCredentials(ctx); err != nil {
		log.Print(err)
		return exitConfig
	}

	mp, err := proxy.NewMultiplexer(client, defaultHoneyCombURL)
	if err != nil {
		log.Print(err)
		return exitInternal
	}

	srv, err := http.NewServer(*addr, mp)
	if err != nil {
		log.Print(err)
		return exitInternal
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer shutdownCancel()

		if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Print(shutdownErr)
		}
	}()

	srv.Run(ctx)

	log.Print("listening on ", srv.ListenAddr().String())

	select {
	case <-ctx.Done():
		log.Print("received interrupt, exiting gracefully")
	case err := <-srv.ListenError():
		log.Print("error starting http server, exiting gracefully: ", err)
		return exitInternal
	}

	return exitOK
}
