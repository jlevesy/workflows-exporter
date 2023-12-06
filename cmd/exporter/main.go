package main

import (
	"context"
	"flag"
	"net/http"
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
		concurencyLimit int
		maxLastPushed   time.Duration
		refreshPeriod   time.Duration
	)

	flag.StringVar(&githubAuthToken, "github-auth-token", "", "github auth token")
	flag.StringVar(&organization, "organization", "", "organization")
	flag.IntVar(&concurencyLimit, "concurency", 100, "How many request are allowed in parallel")
	flag.DurationVar(&maxLastPushed, "max-last-pushed", 35*24*time.Hour, "How many time since the last push to consider a repo inactive")
	flag.DurationVar(&refreshPeriod, "refresh-period", 30*time.Minute, "frequency at which usage data is refreshed")
	flag.StringVar(&listenAddress, "listen-address", ":8080", "The address to listen on for HTTP requests.")
	flag.Parse()

	logger := zap.Must(zap.NewProduction())

	logger.Info(
		"Starting exporter",
		zap.String("organization", organization),
		zap.Int("concurrency", concurencyLimit),
		zap.Duration("max_last_pushed", maxLastPushed),
		zap.Duration("refresh_period", refreshPeriod),
		zap.String("listen_address", listenAddress),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	gh, err := github.NewClient(ctx, githubAuthToken, logger)
	if err != nil {
		logger.Error("Could not setup github client", zap.Error(err))
		return 1
	}

	fetcher := actions.NewOrgUsageFetcher(
		concurencyLimit,
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

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))

	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		logger.Error(
			"Could not listen over HTTP",
			zap.Error(err),
		)
		return 1
	}

	return 0
}
