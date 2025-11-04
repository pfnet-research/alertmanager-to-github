# alertmanager-to-github

This receives webhook requests from Alertmanager and creates GitHub issues.

It does:

- open an issue on a new alert
- close the issue when the alert is in resolved status
- reopen the issue when the alert is in firing status
  - alerts are identified by `groupKey`; configurable via `--alert-id-template` option

<kbd>![screen shot](doc/screenshot.png)</kbd>

## Installation

### Docker image

```shell
docker pull ghcr.io/pfnet-research/alertmanager-to-github:v0.1.0
```

### go get

```shell
go get github.com/pfnet-research/alertmanager-to-github
```

## Usage

Set GitHub API credentials to environment variables:

```shell
$ read ATG_GITHUB_APP_PRIVATE_KEY
(GitHub App's private key)
$ export ATG_GITHUB_APP_PRIVATE_KEY
```

or,

```shell
$ read ATG_GITHUB_TOKEN
(Personal Access Token)
$ export ATG_GITHUB_TOKEN
```

Start webhook server:

```
$ alertmanager-to-github start
```

Add a receiver to Alertmanager config:

```yaml
route:
  receiver: "togithub" # default

receivers:
  - name: "togithub"
    webhook_configs:
      # Create issues in "bar" repo in "foo" organization.
      # these are the default values and can be overriden by labels on the alert
      # repo and owner parameters must be URL-encoded.
      - url: "http://localhost:8080/v1/webhook?owner=foo&repo=bar"
```

## Configuration

```shell
$ alertmanager-to-github start -h
NAME:
   alertmanager-to-github start - Start webhook HTTP server

USAGE:
   alertmanager-to-github start [command options] [arguments...]

OPTIONS:
   --listen value                      HTTP listen on (default: ":8080") [$ATG_LISTEN]
   --github-url value                  GitHub Enterprise URL (e.g. https://github.example.com) [$ATG_GITHUB_URL]
   --labels value [ --labels value ]   Issue labels [$ATG_LABELS]
   --body-template-file value          Body template file [$ATG_BODY_TEMPLATE_FILE]
   --title-template-file value         Title template file [$ATG_TITLE_TEMPLATE_FILE]
   --alert-id-template value           Alert ID template (default: "{{.Payload.GroupKey}}") [$ATG_ALERT_ID_TEMPLATE]
   --github-app-id value               GitHub App ID (default: 0) [$ATG_GITHUB_APP_ID]
   --github-app-installation-id value  GitHub App installation ID (default: 0) [$ATG_GITHUB_APP_INSTALLATION_ID]
   --github-app-private-key value      GitHub App private key (command line argument is not recommended) [$ATG_GITHUB_APP_PRIVATE_KEY]
   --github-token value                GitHub API token (command line argument is not recommended) [$ATG_GITHUB_TOKEN]
   --auto-close-resolved-issues        Should issues be automatically closed when resolved. If alerts have 'atg_skip_auto_close=true' annotation, issues will not be auto-closed. (default: true) [$ATG_AUTO_CLOSE_RESOLVED_ISSUES]
   --reopen-window value               Alerts will create a new issue instead of reopening closed issues if the specified duration has passed [$ATG_REOPEN_WINDOW]
   --help, -h                          show help
```

### GitHub Enterprise

To create issues in GHE, set `--github-url` option or `ATG_GITHUB_URL` environment variable.

### Customize issue title and body

Issue title and body are rendered from [Go template](https://golang.org/pkg/text/template/) and you can use custom templates via `--body-template-file` and `--title-template-file` options. In the templates, you can use the following variables and functions.

- Variables
  - `.Payload`: Webhook payload incoming to this receiver. For more information, see `WebhookPayload` in [pkg/types/payload.go](https://github.com/pfnet-research/alertmanager-to-github/blob/master/pkg/types/payload.go)
  - `.PreviousIssue`: The previous issue with the same alert ID, or `nil` if there is no such issue. For more information, see `Issue` in [github.com/google/go-github/v54/github](https://pkg.go.dev/github.com/google/go-github/v54@v54.0.0/github#Issue). Useful when `--reopen-window` is specified.
- Functions
  - `urlQueryEscape`: Escape a string as a URL query
  - `json`: Marshal an object to JSON string
  - `timeNow`: Get current time

### Automatically close issues when alerts are resolved

You can use the `--auto-close-resolved-issues` flag to automatically close issues when alerts are resolved.

If you want to skip auto-close for some alerts, add the `atg_skip_auto_close=true` annotation to them.

```yaml
- alert: HighRequestLatency
  expr: job:request_latency_seconds:mean5m{job="myjob"} > 0.5
  labels:
    severity: critical
  annotations:
    atg_skip_auto_close: "true"
```

## Customize organization and repository

The organization/repository where issues are raised can be customized per-alert by specifying the `atg_owner` label for the organization and/or the `atg_repo` label for the repository on the alert.

e.g.

```yaml
- alert: HighRequestLatency
  expr: job:request_latency_seconds:mean5m{job="myjob"} > 0.5
  for: 10m
  labels:
    severity: page
    atg_owner: my-alternative-org
    atg_repo: specific-service-repository
  annotations:
    summary: High request latency
```

This mechanism has precedence over the receiver URL query parameters.

## Deployment

### Kubernetes

https://github.com/pfnet-research/alertmanager-to-github/tree/master/example/kubernetes

## Metrics

alertmanager-to-github exposes Prometheus metrics on `/metrics`.

In addition to standard Go runtime and process metrics, the following application-specific metrics are exposed:

| Metric name                 | Metric type | Description                                                      | Labels                                                                            |
|-----------------------------|-------------|------------------------------------------------------------------|-----------------------------------------------------------------------------------|
| `github_api_rate_limit`     | Gauge       | The limit of API requests the client can make.                   | `api`=&lt;search\|issues&gt;                                                      |
| `github_api_rate_remaining` | Gauge       | The remaining API requests the client can make until reset time. | `api`=&lt;search\|issues&gt;                                                      |
| `github_api_rate_reset`     | Gauge       | The time when the current rate limit will reset.                 | `api`=&lt;search\|issues&gt;                                                      |
| `github_api_requests_total` | Counter     | Number of API operations performed.                              | `api`=&lt;search\|issues&gt;<br>`status`=&lt;The status code of the reponse&gt;   |

## Releaese

The release process is fully automated by [tagpr](https://github.com/Songmu/tagpr). To release, just merge [the latest release PR](https://github.com/pfnet-research/alertmanager-to-github/pulls?q=is:pr+is:open+label:tagpr).
