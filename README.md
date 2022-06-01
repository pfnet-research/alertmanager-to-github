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
docker pull ghcr.io/pfnet-research/alertmanager-to-github:v0.0.2
```

### go get

```shell
go get github.com/pfnet-research/alertmanager-to-github
```

## Usage

Start webhook server:

```shell
$ read ATG_GITHUB_TOKEN
(Personal Access Token)
$ export ATG_GITHUB_TOKEN

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
   --listen value               HTTP listen on (default: ":8080") [$ATG_LISTEN]
   --github-url value           GitHub Enterprise URL (e.g. https://github.example.com) [$ATG_GITHUB_URL]
   --labels value               Issue labels [$ATG_LABELS]
   --body-template-file value   Body template file [$ATG_BODY_TEMPLATE_FILE]
   --title-template-file value  Title template file [$ATG_TITLE_TEMPLATE_FILE]
   --alert-id-template value    Alert ID template (default: "{{.Payload.GroupKey}}") [$ATG_ALERT_ID_TEMPLATE]
   --github-token value         GitHub API token (command line argument is not recommended) [$ATG_GITHUB_TOKEN]
   --auto-close-resolved-issues Close resolved issues automatically (default: true) [$ATG_AUTO_CLOSE_RESOLVED_ISSUES]
   --help, -h                   show help (default: false)
```

### GitHub Enterprise

To create issues in GHE, set `--github-url` option or `ATG_GITHUB_URL` environment variable.

### Customize issue title and body

Issue title and body are rendered from [Go template](https://golang.org/pkg/text/template/) and you can use custom templates via `--body-template-file` and `--title-template-file` options. In the templates, you can use the following variables and functions.

- Variables
  - `.Payload`: Webhook payload incoming to this receiver. For more information, see `WebhookPayload` in [pkg/types/payload.go](https://github.com/pfnet-research/alertmanager-to-github/blob/master/pkg/types/payload.go)
- Functions
  - `urlQueryEscape`: Escape a string as a URL query
  - `json`: Marshal an object to JSON string
  - `timeNow`: Get current time

## Deployment

### Kubernetes

https://github.com/pfnet-research/alertmanager-to-github/tree/master/example/kubernetes
