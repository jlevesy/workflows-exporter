# Workflows-Exporter

This exporter hits the Github API to collect workflow billable time for a single org.
It is explicitely designed to only focus on workflow billable time for now, in an attempt to tackle cases where the organization has a large amount (~1000) of repositories.

To achieve this, it does the following tradeoffs:

- It only accounts for repositories being active in the last x days (default is 35 days)
- It works in best effort mode and tries to refresh the data every x minutes (default is 30 minutes)
- It serves the last retrieved data

## Exported metrics

### Billable time

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

## How to use the exporter?

Run the exporter

```
 go run ./cmd/exporter -refresh-period=2m -organization=someapp -github-auth-token=$(gh auth token)
```

All available options can be found using

```
go run ./cmd/exporter -help
```

A docker image will be released soon.
