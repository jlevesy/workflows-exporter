package actions

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type UsageCollector struct {
	billableTimeDescription *prometheus.Desc

	refreshTicker *time.Ticker
	cancelFunc    func()

	usagefetcher WorkflowUsageFetcher

	ready chan struct{}

	lastUsageDataMu sync.RWMutex
	lastUsageData   Usage

	logger *zap.Logger
}

func NewUsageCollector(usagefetcher WorkflowUsageFetcher, logger *zap.Logger, refreshPeriod time.Duration) *UsageCollector {
	ctx, cancel := context.WithCancel(context.Background())

	c := UsageCollector{
		cancelFunc:    cancel,
		refreshTicker: time.NewTicker(refreshPeriod),
		logger:        logger,
		ready:         make(chan struct{}),
		usagefetcher:  usagefetcher,

		billableTimeDescription: prometheus.NewDesc(
			"github_actions_workflow_billable_time_seconds",
			"Billable time for a repo, per workflow and platform",
			[]string{"owner", "repo", "workflow", "workflow_id", "platform"},
			nil,
		),
	}

	go func() {
		c.refresh(ctx)

		close(c.ready)

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.refreshTicker.C:
				c.refresh(ctx)
			}
		}
	}()

	return &c
}
func (c *UsageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.billableTimeDescription
}

func (c *UsageCollector) Collect(ch chan<- prometheus.Metric) {
	c.lastUsageDataMu.RLock()
	defer c.lastUsageDataMu.RUnlock()

	for _, workflowData := range c.lastUsageData {
		for platform, value := range workflowData.BillableTime {
			ch <- prometheus.MustNewConstMetric(
				c.billableTimeDescription,
				prometheus.GaugeValue,
				value.Seconds(),
				workflowData.Owner,
				workflowData.Repo,
				workflowData.Workflow,
				strconv.FormatInt(workflowData.ID, 10),
				platform,
			)
		}
	}
}

func (c *UsageCollector) Close() error {
	c.cancelFunc()
	c.refreshTicker.Stop()

	return nil
}

func (c *UsageCollector) Ready() <-chan struct{} {
	return c.ready
}

func (c *UsageCollector) refresh(ctx context.Context) {
	c.logger.Info("Refreshing usage data")

	usageData, err := c.usagefetcher.Fetch(ctx)
	if err != nil {
		c.logger.Error(
			"Could not retrieve updated usage data",
			zap.Error(err),
		)

		return
	}

	c.logger.Info("Done refreshing usage data")

	c.lastUsageDataMu.Lock()
	c.lastUsageData = usageData
	c.lastUsageDataMu.Unlock()
}
