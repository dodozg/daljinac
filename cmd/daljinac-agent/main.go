// Command daljinac-agent reports this machine's CPU, memory and disk usage to
// the Daljinac server at a fixed interval.
//
// Installed (daljinac-agent -install), it runs as a Windows service and reads
// the server address and token from a config file only administrators can
// read. For trials it can also run in a console with the settings given by
// -server and the DALJINAC_TOKEN environment variable.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dodozg/daljinac/internal/agent"
)

var version = "dev"

const minInterval = 10 * time.Second

func main() {
	configPath := flag.String("config", agent.DefaultConfigPath(), "config file with the server URL and token")
	serverURL := flag.String("server", os.Getenv("DALJINAC_SERVER"), "server base URL, e.g. https://remote.example.hr; overrides the config file (env DALJINAC_SERVER)")
	interval := flag.Duration("interval", time.Minute, "time between reports")
	once := flag.Bool("once", false, "send one report and exit")
	install := flag.Bool("install", false, "install or update the Windows service (run as administrator); the token comes from DALJINAC_TOKEN or the existing config")
	uninstall := flag.Bool("uninstall", false, "remove the Windows service and its config (run as administrator)")
	flag.Parse()

	if *interval < minInterval {
		fmt.Fprintf(os.Stderr, "interval must be at least %s\n", minInterval)
		os.Exit(2)
	}

	if isService() {
		runService(*configPath, *interval)
		return
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	// The token is read only from the environment or the config file, never
	// from a flag: any local user can read a process's command line.
	token := os.Getenv("DALJINAC_TOKEN")

	switch {
	case *install && *uninstall:
		log.Error("use either -install or -uninstall")
		os.Exit(2)
	case *install:
		if err := installService(log, *configPath, *serverURL, token); err != nil {
			log.Error("install failed", "err", err)
			os.Exit(1)
		}
		return
	case *uninstall:
		if err := uninstallService(log, *configPath); err != nil {
			log.Error("uninstall failed", "err", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := agent.Resolve(*configPath, *serverURL, token)
	if err != nil {
		log.Error("no usable settings: install the service, or set -server and DALJINAC_TOKEN", "err", err)
		os.Exit(2)
	}
	report, err := newReporter(cfg)
	if err != nil {
		log.Error("invalid config", "err", err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *once {
		if err := report(ctx); err != nil {
			log.Error("report failed", "err", err)
			os.Exit(1)
		}
		return
	}
	log.Info("daljinac-agent started", "version", version, "server", cfg.Server, "interval", *interval)
	agent.Loop(ctx, log, *interval, report)
}

// newReporter returns a function that collects and sends one report.
func newReporter(cfg agent.Config) (func(context.Context) error, error) {
	reportURL, err := agent.ReportURL(cfg.Server)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	return func(ctx context.Context) error {
		r, err := agent.Collect(ctx, 2*time.Second, version)
		if err != nil {
			return err
		}
		return agent.Send(ctx, client, reportURL, cfg.Token, r)
	}, nil
}
