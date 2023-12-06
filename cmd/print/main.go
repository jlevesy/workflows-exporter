package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/jlevesy/workflows-exporter/actions"
	"github.com/jlevesy/workflows-exporter/pkg/github"
	"go.uber.org/zap"
)

func main() { os.Exit(run()) }

func run() int {
	var (
		githubAuthToken string
		organization    string
		concurencyLimit int
		maxLastPushed   time.Duration
	)

	flag.StringVar(&githubAuthToken, "github-auth-token", "", "GitHub auth token")
	flag.StringVar(&organization, "organization", "", "organization")
	flag.IntVar(&concurencyLimit, "concurency", 100, "How many request are allowed in parallel")
	flag.DurationVar(&maxLastPushed, "max-last-pushed", 30*24*time.Hour, "How many time since the last push to consider a repo inactive")
	flag.Parse()

	logger := zap.Must(zap.NewDevelopment())

	if organization == "" {
		logger.Error("You must provide an organization, exiting")
		return 1
	}

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

	usage, err := fetcher.Fetch(ctx)
	if err != nil {
		logger.Error(
			"Unable to retrieve usage information",
			zap.String("org", organization),
			zap.Error(err),
		)

		return 1
	}

	sort.Slice(usage, func(i, j int) bool {
		return usage[i].Repo < usage[j].Repo
	})

	for _, workflowUsage := range usage {
		logger.Info(
			"Got usage stats",
			zap.String("owner", workflowUsage.Owner),
			zap.String("repo", workflowUsage.Repo),
			zap.String("workflow", workflowUsage.Workflow),
			zap.Int64("worfkow_id", workflowUsage.ID),
			zap.Any("billable_time", workflowUsage.BillableTime),
		)
	}

	return 0
}
