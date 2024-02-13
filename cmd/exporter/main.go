package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jlevesy/workflows-exporter/actions"
	"github.com/jlevesy/workflows-exporter/pkg/github"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() { os.Exit(run()) }

func run() int {
	var (
		listenAddress   string
		githubAuthToken string
		organization    string
		enablePprof     bool
		maxLastPushed   time.Duration
		refreshPeriod   time.Duration
		shutdownDelay   time.Duration
	)

	flag.StringVar(&githubAuthToken, "github-auth-token", "", "GitHub auth token")
	flag.StringVar(&organization, "organization", "", "Organization to monitor")
	flag.DurationVar(&maxLastPushed, "max-last-pushed", 35*24*time.Hour, "How many time since the last push to consider a repo inactive")
	flag.DurationVar(&refreshPeriod, "refresh-period", 30*time.Minute, "Frequency at which usage data is refreshed")
	flag.DurationVar(&shutdownDelay, "shutdown-delay", 15*time.Second, "Graceful shutdown delay")
	flag.BoolVar(&enablePprof, "pprof", false, "Enable pprof endpoints")
	flag.StringVar(&listenAddress, "listen-address", ":8080", "The address to listen on for HTTP requests.")
	flag.Parse()

	logger := zap.Must(zap.NewProduction())

	logger.Info(
		"Starting exporter",
		zap.String("organization", organization),
		zap.Duration("max_last_pushed", maxLastPushed),
		zap.Duration("refresh_period", refreshPeriod),
		zap.String("listen_address", listenAddress),
		zap.Bool("pprof", enablePprof),
	)

	if githubAuthToken == "" {
		githubAuthToken = os.Getenv("GITHUB_TOKEN")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	gh, err := github.NewClient(ctx, githubAuthToken, logger)
	if err != nil {
		logger.Error("Could not setup github client", zap.Error(err))
		return 1
	}

	fetcher := actions.NewOrgUsageFetcher(
		maxLastPushed,
		organization,
		gh,
		logger,
	)

	usageCollector := actions.NewUsageCollector(fetcher, logger, refreshPeriod)

	defer usageCollector.Close()

	reg := prometheus.NewRegistry()

	reg.MustRegister(
		collectors.NewGoCollector(),
		usageCollector,
	)

	var (
		mux http.ServeMux
		srv = http.Server{
			Addr:    listenAddress,
			Handler: &mux,
		}
	)

	// Expose /metrics HTTP endpoint using the created custom registry.
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))

	if enablePprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	go func() {
		<-ctx.Done()

		logger.Info("Received a signal, exiting")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownDelay)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("Could not gracefully shut down the server, stopping", zap.Error(err))
			_ = srv.Close()
		}

	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(
			"Could not listen over HTTP",
			zap.Error(err),
		)
		return 1
	}

	logger.Info("Server stopped")

	return 0
}
