# Workflows-Exporter

This exporter hits the Github API to collect workflow billable time for a single org.
It is explicitely designed to only focus on workflow billable time for now, in an attempt to tackle cases where the organization has a large amount (~1000) of repositories.

To achieve this, it does the following tradeoffs:

- It only accounts for repositories being active in the last x days (default is 35 days)
- It works in best effort mode and tries to refresh the data every x minutes (default is 30 minutes)
- It serves the last retrieved data

## Exported metrics

### Billable Time

```
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
```

### Active Repositories

How many repositories under the organization are considered active. Depends on the `-max-last-push` setting.

```
# HELP github_actions_workflow_active_repos Last reported total of active repositories in the monitored org
# TYPE github_actions_workflow_active_repos gauge
github_actions_workflow_active_repos 174
```

### Last Refresh Timestamp

Last timestamp in seconds where the exported managed to refresh the data. Usefull for detecting stale data.

```
# HELP github_actions_workflow_last_refresh_timestamp_seconds Last timestamp in seconds since epoch of the last dataset refresh
# TYPE github_actions_workflow_last_refresh_timestamp_seconds gauge
github_actions_workflow_last_refresh_timestamp_seconds 1.697328e+09
```

### Last Refresh Duration

How much time it took to refresh the whole dataset.

```
# HELP github_actions_workflow_last_refresh_duration_seconds Last refresh duration in seconds
# TYPE github_actions_workflow_last_refresh_duration_seconds gauge
github_actions_workflow_last_refresh_duration_seconds 1
```


## How to use the exporter?

Run the exporter

```
 go run ./cmd/exporter -refresh-period=2m -organization=someapp -github-auth-token=$(gh auth token)
```

All available options can be found using

```
go run ./cmd/exporter -help
```

Here's the currently supported options

```
-concurency int
    How many requests are allowed in parallel (default 100)
-github-auth-token string
    GitHub auth token
-listen-address string
    The address to listen on for HTTP requests. (default ":8080")
-max-last-pushed duration
    How many time since the last push to consider a repo inactive (default 840h0m0s)
-organization string
    Organization to monitor
-refresh-period duration
    Frequency at which usage data is refreshed (default 30m0s)
-shutdown-delay duration
    Graceful shutdown delay (default 15s)
```

The exporter reads the auth token either from the -github-auth-token flag or the `GITHUB_TOKEN` environment variable.

