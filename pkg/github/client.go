package github

import (
	"context"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v57/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

func NewClient(ctx context.Context, token string, logger *zap.Logger) (*github.Client, error) {
	tc := oauth2.NewClient(
		ctx,
		oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		),
	)
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(
		tc.Transport,
		github_ratelimit.WithLimitDetectedCallback(
			func(ctx *github_ratelimit.CallbackContext) {
				logger.Error(
					"Detected ratelimit, all calls are currently held",
					zap.Any("context", ctx),
				)
			},
		),
	)
	if err != nil {
		return nil, err
	}

	return github.NewClient(rateLimiter), nil
}
