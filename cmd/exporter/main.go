package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/jlevesy/workflows-exporter/actions"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
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
	flag.IntVar(&concurencyLimit, "concurency", 100, "concurency limit")
	flag.DurationVar(&maxLastPushed, "max-last-pushed", 30*24*time.Hour, "How much time since the last push to consider a repo inactive")
	flag.DurationVar(&refreshPeriod, "refresh-period", 15*time.Minute, "frequency at which usage data is refreshed")
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

	gh := github.NewClient(nil).WithAuthToken(
		githubAuthToken,
	)

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
	}
}
