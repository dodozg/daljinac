// Command daljinac-agent reports this machine's CPU, memory and disk usage to
// the Daljinac server at a fixed interval.
package main

import (
	"context"
	"flag"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dodozg/daljinac/internal/agent"
)

var version = "dev"

func main() {
	serverURL := flag.String("server", os.Getenv("DALJINAC_SERVER"), "server base URL, e.g. https://remote.example.hr (env DALJINAC_SERVER)")
	interval := flag.Duration("interval", time.Minute, "time between reports")
	once := flag.Bool("once", false, "send one report and exit")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// The token comes only from the environment so it never shows in the
	// process list, where any local user could read a command-line flag.
	token := os.Getenv("DALJINAC_TOKEN")
	if *serverURL == "" || token == "" {
		log.Error("set -server (or DALJINAC_SERVER) and the DALJINAC_TOKEN environment variable")
		os.Exit(2)
	}
	reportURL, err := agent.ReportURL(*serverURL)
	if err != nil {
		log.Error("invalid server URL", "err", err)
		os.Exit(2)
	}
	if *interval < 10*time.Second {
		log.Error("interval must be at least 10s")
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client := &http.Client{Timeout: 20 * time.Second}

	report := func() error {
		r, err := agent.Collect(ctx, 2*time.Second, version)
		if err != nil {
			return err
		}
		return agent.Send(ctx, client, reportURL, token, r)
	}

	if *once {
		if err := report(); err != nil {
			log.Error("report failed", "err", err)
			os.Exit(1)
		}
		return
	}

	log.Info("daljinac-agent started", "version", version, "server", *serverURL, "interval", *interval)
	// A random start delay keeps 500 agents from reporting in the same second
	// after a server restart or a power cut.
	wait := time.Duration(rand.Int64N(int64(*interval)))
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		if err := report(); err != nil {
			log.Warn("report failed", "err", err)
		}
		wait = *interval
	}
}
