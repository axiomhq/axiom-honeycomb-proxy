package main

import (
	"context"
	"flag"
	"os"
	"runtime"
	"runtime/pprof"

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

	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
)

func main() {
	cmd.Run("axiom-honeycomb-proxy", run,
		cmd.WithValidateAxiomCredentials(),
	)
}

func run(ctx context.Context, log *zap.Logger, client *axiom.Client) error {
	// Export `AXIOM_TOKEN` and `AXIOM_ORG_ID` for Axiom Cloud.
	// Export `AXIOM_URL` and `AXIOM_TOKEN` for Axiom Selfhost.

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", zap.Error(err))
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", zap.Error(err))
		}
		defer pprof.StopCPUProfile()
	}

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

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", zap.Error(err))
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", zap.Error(err))
		}
	}

	return nil
}
