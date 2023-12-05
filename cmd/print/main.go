package main

import (
	"context"
	"flag"
	"sort"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/jlevesy/workflows-exporter/actions"
	"go.uber.org/zap"
)

func main() {
	var (
		githubAuthToken string
		organization    string
		concurencyLimit int
		maxLastPushed   time.Duration
	)

	flag.StringVar(&githubAuthToken, "github-auth-token", "", "github auth token")
	flag.StringVar(&organization, "organization", "", "organization")
	flag.IntVar(&concurencyLimit, "concurency", 100, "concurency limit")
	flag.DurationVar(&maxLastPushed, "max-last-pushed", 30*24*time.Hour, "How many time since the last push to consider a repo inactive")
	flag.Parse()

	logger := zap.Must(zap.NewDevelopment())

	if organization == "" {
		logger.Fatal("You must provide an organization, exiting")
	}

	var (
		ctx = context.Background()
		gh  = github.NewClient(nil).WithAuthToken(
			githubAuthToken,
		)

		fetcher = actions.NewOrgUsageFetcher(
			concurencyLimit,
			maxLastPushed,
			organization,
			gh,
			logger,
		)
	)

	usage, err := fetcher.Fetch(ctx)
	if err != nil {
		logger.Fatal(
			"Unable to retrieve usage information",
			zap.String("org", organization),
			zap.Error(err),
		)
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
}
