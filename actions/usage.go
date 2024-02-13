package actions

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/go-github/v57/github"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type WorkflowUsageFetcher interface {
	Fetch(ctx context.Context) (*Usage, error)
}

type WorkflowUsage struct {
	Owner    string
	Repo     string
	Workflow string
	ID       int64

	BillableTime map[string]time.Duration
}

type Usage struct {
	ActiveRepos int64
	Workflows   []WorkflowUsage
}

type OrgUsageFetcher struct {
	gh     *github.Client
	logger *zap.Logger

	maxLastPushed time.Duration
	org           string
}

func NewOrgUsageFetcher(maxLastPushed time.Duration, org string, gh *github.Client, logger *zap.Logger) *OrgUsageFetcher {
	return &OrgUsageFetcher{
		maxLastPushed: maxLastPushed,
		org:           org,
		gh:            gh,
		logger:        logger,
	}
}

func (f *OrgUsageFetcher) Fetch(ctx context.Context) (*Usage, error) {
	var (
		usageMu sync.Mutex
		usage   Usage

		group, groupCtx = errgroup.WithContext(ctx)
	)

	group.Go(func() error {
		return scanAllOrgRepos(
			groupCtx,
			f.org,
			f.gh.Repositories,
			func(reposBatch []*github.Repository) error {
				var totalInactive int

				f.logger.Info(
					"New batch of repositories",
					zap.Int("length", len(reposBatch)),
				)

				for _, repo := range reposBatch {
					if time.Since(repo.GetPushedAt().Time) >= f.maxLastPushed {
						totalInactive++
						continue
					}

					// No mutex needed here, only one goroutine in writing this integer.
					usage.ActiveRepos++

					repo := repo

					group.Go(func() error {
						return scanAllRepoWorkflows(
							ctx,
							f.org,
							repo.GetName(),
							f.gh.Actions,
							func(workflows *github.Workflows) {
								f.logger.Debug(
									"Collecting data for repo",
									zap.String("owner", f.org),
									zap.String("repo", repo.GetName()),
									zap.Int("workflow_count", workflows.GetTotalCount()),
								)

								for _, workflow := range workflows.Workflows {
									workflow := workflow

									group.Go(func() error {
										workflowUsage, _, err := f.gh.Actions.GetWorkflowUsageByID(
											ctx,
											f.org,
											repo.GetName(),
											workflow.GetID(),
										)
										if err != nil {
											return err
										}

										result := WorkflowUsage{
											Owner:    f.org,
											Repo:     repo.GetName(),
											Workflow: workflow.GetName(),
											ID:       workflow.GetID(),
											BillableTime: makeBillableTime(
												workflowUsage.GetBillable(),
											),
										}

										usageMu.Lock()
										usage.Workflows = append(usage.Workflows, result)
										usageMu.Unlock()

										f.logger.Debug(
											"Collected usage Info",
											zap.String("owner", f.org),
											zap.String("repo", repo.GetName()),
											zap.String("workflow", workflow.GetName()),
										)

										return nil
									})
								}
							},
						)
					})
				}

				// Don't scan for all repos, if all are inactive in a single batch
				// and because we're scanning in pushed_at descending order
				// then we can consider the job done.
				if totalInactive == len(reposBatch) {
					f.logger.Info("Got a full batch of inactive repositories, exiting")
					return errEarlyExit
				}

				return nil
			},
		)
	})

	return &usage, group.Wait()
}

func scanAllRepoWorkflows(ctx context.Context, org, repo string, workflowClient *github.ActionsService, cb func(*github.Workflows)) error {
	var nextPage int

	for {
		workflowBatch, resp, err := workflowClient.ListWorkflows(
			ctx,
			org,
			repo,
			&github.ListOptions{
				Page:    nextPage,
				PerPage: 10,
			},
		)

		if err != nil {
			return err
		}

		cb(workflowBatch)

		if resp.NextPage == 0 {
			return nil
		}

		nextPage = resp.NextPage
	}
}

var errEarlyExit = errors.New("early exit")

func scanAllOrgRepos(ctx context.Context, org string, reposClient *github.RepositoriesService, cb func([]*github.Repository) error) error {
	var nextPage int

	for {
		reposBatch, resp, err := reposClient.ListByOrg(
			ctx,
			org,
			&github.RepositoryListByOrgOptions{
				Sort:      "pushed",
				Direction: "desc",
				ListOptions: github.ListOptions{
					Page:    nextPage,
					PerPage: 100,
				},
			},
		)
		if err != nil {
			return err
		}

		err = cb(reposBatch)
		switch {
		case errors.Is(err, errEarlyExit):
			return nil
		case err != nil:
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		nextPage = resp.NextPage
	}
}

func makeBillableTime(ghBillableTime *github.WorkflowBillMap) map[string]time.Duration {
	result := make(map[string]time.Duration, len(*ghBillableTime))

	for platform, billableTimeMs := range *ghBillableTime {
		result[platform] = time.Duration(billableTimeMs.GetTotalMS()) * time.Millisecond
	}

	return result
}
