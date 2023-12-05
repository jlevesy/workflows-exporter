package actions_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/jlevesy/workflows-exporter/actions"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

var (
	billableMetrics = `
# HELP github_actions_workflow_billable_time_seconds Billable time for a repo, per workflow and platform
# TYPE github_actions_workflow_billable_time_seconds gauge
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-A",workflow="build",workflow_id="1"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-A",workflow="release",workflow_id="3"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-A",workflow="run",workflow_id="4"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-A",workflow="test",workflow_id="2"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-B",workflow="build",workflow_id="1"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-B",workflow="release",workflow_id="3"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-B",workflow="run",workflow_id="4"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-B",workflow="test",workflow_id="2"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-C",workflow="build",workflow_id="1"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-C",workflow="release",workflow_id="3"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-C",workflow="run",workflow_id="4"} 15
github_actions_workflow_billable_time_seconds{owner="totocorp",platform="UBUNTU",repo="repo-C",workflow="test",workflow_id="2"} 15
`
	repos = []any{
		[]github.Repository{
			{
				Name: github.String("repo-A"),
				PushedAt: &github.Timestamp{
					Time: time.Now().Add(-1 * time.Second),
				},
			},
			{
				Name: github.String("repo-B"),
				PushedAt: &github.Timestamp{
					Time: time.Now().Add(-1 * time.Second),
				},
			},
		},
		[]github.Repository{
			{
				Name: github.String("repo-C"),
				PushedAt: &github.Timestamp{
					Time: time.Now().Add(-1 * time.Second),
				},
			},
			{
				Name: github.String("repo-D"),
				PushedAt: &github.Timestamp{
					Time: time.Now().Add(-48 * time.Hour),
				},
			},
		},
		// Last page triggers an early return,
		// because all members are considered inactive.
		[]github.Repository{
			{
				Name: github.String("repo-E"),
				PushedAt: &github.Timestamp{
					Time: time.Now().Add(-58 * time.Hour),
				},
			},
			{
				Name: github.String("repo-F"),
				PushedAt: &github.Timestamp{
					Time: time.Now().Add(-58 * time.Hour),
				},
			},
		},
	}
	workflows = []any{
		github.Workflows{
			Workflows: []*github.Workflow{
				{
					ID:   ptr(int64(1)),
					Name: ptr("build"),
				},
				{
					ID:   ptr(int64(2)),
					Name: ptr("test"),
				},
			},
		},
		github.Workflows{
			Workflows: []*github.Workflow{
				{
					ID:   ptr(int64(3)),
					Name: ptr("release"),
				},
				{
					ID:   ptr(int64(4)),
					Name: ptr("run"),
				},
			},
		},
	}
	workflowTiming = github.WorkflowUsage{
		Billable: &github.WorkflowBillMap{
			"UBUNTU": &github.WorkflowBill{
				TotalMS: ptr(int64(15000)),
			}},
	}
)

func TestCollector(t *testing.T) {
	for _, testCase := range []struct {
		metricName  string
		mockOptions []mock.MockBackendOption
		wantMetrics string
	}{
		{
			metricName: "github_actions_workflow_billable_time_seconds",
			mockOptions: []mock.MockBackendOption{
				mock.WithRequestMatchPages(mock.GetOrgsReposByOrg, repos...),
				mock.WithRequestMatchPages(mock.GetReposActionsWorkflowsByOwnerByRepo, workflows...),
				mock.WithRequestMatchPages(
					mock.GetReposActionsWorkflowsTimingByOwnerByRepoByWorkflowId,
					workflowTiming,
				),
			},
			wantMetrics: billableMetrics,
		},
	} {
		t.Run(testCase.metricName, func(t *testing.T) {
			var (
				logger  = zaptest.NewLogger(t)
				gh      = github.NewClient(mock.NewMockedHTTPClient(testCase.mockOptions...))
				fetcher = actions.NewOrgUsageFetcher(
					100,
					24*time.Hour,
					"totocorp",
					gh,
					logger,
				)
				collector = actions.NewUsageCollector(
					fetcher,
					logger,
					10*time.Minute,
				)
				registry = prometheus.NewRegistry()
			)

			err := registry.Register(collector)
			require.NoError(t, err)

			<-collector.Ready()

			err = testutil.GatherAndCompare(
				registry,
				bytes.NewBufferString(testCase.wantMetrics),
				testCase.metricName,
			)
			require.NoError(t, err)
		})
	}
}

func ptr[V any](v V) *V { return &v }
