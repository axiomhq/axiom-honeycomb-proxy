package main

import (
	"context"
	"flag"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/pkg/cmd"
	"github.com/axiomhq/pkg/http"
	"go.uber.org/zap"

	"github.com/axiomhq/axiom-honeycomb-proxy/proxy"
)

const defaultHoneyCombURL = "https://api.honeycomb.io"

var (
	addr            = flag.String("addr", ":8080", "Listen address <ip>:<port>")
	byPassHoneyComb = flag.Bool("bypass", false, "Bypass Honeycomb")
)

func main() {
	cmd.Run("axiom-honeycomb-proxy", run,
		cmd.WithValidateAxiomCredentials(),
	)
}

func run(ctx context.Context, log *zap.Logger, client *axiom.Client) error {
	flag.Parse()

	url := ""
	if !*byPassHoneyComb {
		url = defaultHoneyCombURL
	}

	mp, err := proxy.NewMultiplexer(client, url)
	if err != nil {
		return cmd.Error("create multiplexer", err)
	}

	srv, err := http.NewServer(*addr, mp,
		http.WithBaseContext(ctx),
		http.WithLogger(log),
	)
	if err != nil {
		return cmd.Error("create http server", err)
	}
	defer func() {
		if shutdownErr := srv.Shutdown(); shutdownErr != nil {
			log.Error("stopping server", zap.Error(shutdownErr))
			return
		}
	}()

	srv.Run(ctx)

	log.Info("server listening",
		zap.String("address", srv.ListenAddr().String()),
		zap.String("network", srv.ListenAddr().Network()),
	)

	select {
	case <-ctx.Done():
		log.Warn("received interrupt, exiting gracefully")
	case err := <-srv.ListenError():
		return cmd.Error("error starting http server, exiting gracefully", err)
	}

	return nil
}
