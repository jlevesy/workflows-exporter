package actions

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type UsageCollectorOpt func(c *UsageCollector)

func WithNowFunc(fn func() time.Time) UsageCollectorOpt {
	return func(c *UsageCollector) {
		c.nowFunc = fn
	}
}

type SinceFunc func(time.Time, time.Time) time.Duration

func WithSinceFunc(fn SinceFunc) UsageCollectorOpt {
	return func(c *UsageCollector) {
		c.sinceFunc = fn
	}
}

type UsageCollector struct {
	billableTimeDesc        *prometheus.Desc
	lastRefreshTimeDesc     *prometheus.Desc
	lastRefreshDurationDesc *prometheus.Desc

	refreshTicker *time.Ticker
	cancelFunc    func()

	usagefetcher WorkflowUsageFetcher

	ready chan struct{}

	lastUsageDataMu     sync.RWMutex
	lastUsageData       Usage
	lastRefreshTime     time.Time
	lastRefreshDuration time.Duration

	logger    *zap.Logger
	nowFunc   func() time.Time
	sinceFunc SinceFunc
}

func NewUsageCollector(usagefetcher WorkflowUsageFetcher, logger *zap.Logger, refreshPeriod time.Duration, opts ...UsageCollectorOpt) *UsageCollector {
	ctx, cancel := context.WithCancel(context.Background())

	c := UsageCollector{
		cancelFunc:    cancel,
		refreshTicker: time.NewTicker(refreshPeriod),
		logger:        logger,
		ready:         make(chan struct{}),
		usagefetcher:  usagefetcher,
		nowFunc:       time.Now,
		sinceFunc:     since,

		billableTimeDesc: prometheus.NewDesc(
			"github_actions_workflow_billable_time_seconds",
			"Billable time for a repo, per workflow and platform",
			[]string{"owner", "repo", "workflow", "workflow_id", "platform"},
			nil,
		),
		lastRefreshTimeDesc: prometheus.NewDesc(
			"github_actions_workflow_last_refresh_timestamp_seconds",
			"Last timestamp in seconds since epoch of the last dataset refresh",
			nil,
			nil,
		),
		lastRefreshDurationDesc: prometheus.NewDesc(
			"github_actions_workflow_last_refresh_duration_seconds",
			"Last refresh duration in seconds",
			nil,
			nil,
		),
	}

	for _, opt := range opts {
		opt(&c)
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
	ch <- c.billableTimeDesc
	ch <- c.lastRefreshTimeDesc
	ch <- c.lastRefreshDurationDesc
}

func (c *UsageCollector) Collect(ch chan<- prometheus.Metric) {
	c.lastUsageDataMu.RLock()
	defer c.lastUsageDataMu.RUnlock()

	for _, workflowData := range c.lastUsageData {
		for platform, value := range workflowData.BillableTime {
			ch <- prometheus.MustNewConstMetric(
				c.billableTimeDesc,
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

	if !c.lastRefreshTime.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.lastRefreshTimeDesc,
			prometheus.GaugeValue,
			float64(c.lastRefreshTime.Unix()),
		)
	}

	if c.lastRefreshDuration != 0 {
		ch <- prometheus.MustNewConstMetric(
			c.lastRefreshDurationDesc,
			prometheus.GaugeValue,
			c.lastRefreshDuration.Seconds(),
		)
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

	startTime := c.nowFunc()
	usageData, err := c.usagefetcher.Fetch(ctx)
	if err != nil {
		c.logger.Error(
			"Could not retrieve updated usage data",
			zap.Error(err),
		)

		return
	}
	endTime := c.nowFunc()

	duration := c.sinceFunc(startTime, endTime)

	c.logger.Info("Done refreshing usage data", zap.Duration("took", duration))

	c.lastUsageDataMu.Lock()
	c.lastUsageData = usageData
	c.lastRefreshDuration = duration
	c.lastRefreshTime = endTime
	c.lastUsageDataMu.Unlock()
}

func since(t1, t2 time.Time) time.Duration { return t2.Sub(t1) }
