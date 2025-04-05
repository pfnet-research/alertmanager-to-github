package cli

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v54/github"
	"github.com/pfnet-research/alertmanager-to-github/pkg/notifier"
	"github.com/pfnet-research/alertmanager-to-github/pkg/server"
	"github.com/pfnet-research/alertmanager-to-github/pkg/template"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"
)

const flagListen = "listen"
const flagGitHubURL = "github-url"
const flagLabels = "labels"
const flagBodyTemplateFile = "body-template-file"
const flagTitleTemplateFile = "title-template-file"
const flagGitHubAppID = "github-app-id"
const flagGitHubAppInstallationID = "github-app-installation-id"
const flagGitHubAppPrivateKey = "github-app-private-key"
const flagGitHubToken = "github-token"
const flagAlertIDTemplate = "alert-id-template"
const flagTemplateFile = "template-file"
const flagPayloadFile = "payload-file"
const flagAutoCloseResolvedIssues = "auto-close-resolved-issues"
const flagReopenWindow = "reopen-window"
const flagNoPreviousIssue = "no-previous-issue"

const defaultPayload = `{
  "version": "4",
  "groupKey": "groupKey1",
  "status": "firing",
  "receiver": "receiver1",
  "groupLabels": {
    "groupLabelKey1": "groupLabelValue1",
    "groupLabelKey2": "groupLabelValue2"
  },
  "commonLabels": {
    "groupLabelKey1": "groupLabelValue1",
    "groupLabelKey2": "groupLabelValue2",
    "commonLabelKey1": "commonLabelValue1",
    "commonLabelKey2": "commonLabelValue2"
  },
  "commonAnnotations": {
    "commonAnnotationKey1": "commonAnnotationValue1",
    "commonAnnotationKey2": "commonAnnotationValue2"
  },
  "externalURL": "https://externalurl.example.com",
  "alerts": [
    {
      "status": "firing",
      "labels": {
		"groupLabelKey1": "groupLabelValue1",
		"groupLabelKey2": "groupLabelValue2",
		"commonLabelKey1": "commonLabelValue1",
		"commonLabelKey2": "commonLabelValue2",
		"labelKey1": "labelValue1",
		"labelKey2": "labelValue2"
	  },
      "annotations": {
		"commonAnnotationKey1": "commonAnnotationValue1",
		"commonAnnotationKey2": "commonAnnotationValue2",
		"annotationKey1": "annotationValue1",
		"annotationKey2": "annotationValue2"
      },
      "startsAt": "2020-06-15T11:56:07+09:00",
	  "generatorURL": "https://generatorurl.example.com"
    },
    {
      "status": "firing",
      "labels": {
		"groupLabelKey1": "groupLabelValue1",
		"groupLabelKey2": "groupLabelValue2",
		"commonLabelKey1": "commonLabelValue1",
		"commonLabelKey2": "commonLabelValue2",
		"labelKey1": "labelValue3",
		"labelKey2": "labelValue4"
	  },
      "annotations": {
		"commonAnnotationKey1": "commonAnnotationValue1",
		"commonAnnotationKey2": "commonAnnotationValue2",
		"annotationKey1": "annotationValue3",
		"annotationKey2": "annotationValue4"
      },
      "startsAt": "2020-06-15T11:56:07+09:00",
	  "generatorURL": "https://generatorurl.example.com"
    }
  ]
}`

//go:embed samples/issue.json
var sampleIssue string

//go:embed templates/*.tmpl
var templates embed.FS

func App() *cli.App {
	return &cli.App{
		Name:  os.Args[0],
		Usage: "Webhook receiver Alertmanager to create GitHub issues",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start webhook HTTP server",
				Action: func(c *cli.Context) error {
					if err := actionStart(c); err != nil {
						return cli.Exit(fmt.Errorf("error: %w", err), 1)
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    flagListen,
						Value:   ":8080",
						Usage:   "HTTP listen on",
						EnvVars: []string{"ATG_LISTEN"},
					},
					&cli.StringFlag{
						Name:    flagGitHubURL,
						Usage:   "GitHub Enterprise URL (e.g. https://github.example.com)",
						EnvVars: []string{"ATG_GITHUB_URL"},
					},
					&cli.StringSliceFlag{
						Name:    flagLabels,
						Usage:   "Issue labels",
						EnvVars: []string{"ATG_LABELS"},
					},
					&cli.StringFlag{
						Name:    flagBodyTemplateFile,
						Usage:   "Body template file",
						EnvVars: []string{"ATG_BODY_TEMPLATE_FILE"},
					},
					&cli.StringFlag{
						Name:    flagTitleTemplateFile,
						Usage:   "Title template file",
						EnvVars: []string{"ATG_TITLE_TEMPLATE_FILE"},
					},
					&cli.StringFlag{
						Name:    flagAlertIDTemplate,
						Value:   "{{.Payload.GroupKey}}",
						Usage:   "Alert ID template",
						EnvVars: []string{"ATG_ALERT_ID_TEMPLATE"},
					},
					&cli.Int64Flag{
						Name:     flagGitHubAppID,
						Required: false,
						Usage:    "GitHub App ID",
						EnvVars:  []string{"ATG_GITHUB_APP_ID"},
					},
					&cli.Int64Flag{
						Name:     flagGitHubAppInstallationID,
						Required: false,
						Usage:    "GitHub App installation ID",
						EnvVars:  []string{"ATG_GITHUB_APP_INSTALLATION_ID"},
					},
					&cli.StringFlag{
						Name:     flagGitHubAppPrivateKey,
						Required: false,
						Usage:    "GitHub App private key (command line argument is not recommended)",
						EnvVars:  []string{"ATG_GITHUB_APP_PRIVATE_KEY"},
					},
					&cli.StringFlag{
						Name:     flagGitHubToken,
						Required: false,
						Usage:    "GitHub API token (command line argument is not recommended)",
						EnvVars:  []string{"ATG_GITHUB_TOKEN"},
					},
					&cli.BoolFlag{
						Name:     flagAutoCloseResolvedIssues,
						Required: false,
						Value:    true,
						Usage:    "Should issues be automatically closed when resolved",
						EnvVars:  []string{"ATG_AUTO_CLOSE_RESOLVED_ISSUES"},
					},
					&noDefaultDurationFlag{
						cli.DurationFlag{
							Name:     flagReopenWindow,
							Required: false,
							Usage:    "Alerts will create a new issue instead of reopening closed issues if the specified duration has passed",
							EnvVars:  []string{"ATG_REOPEN_WINDOW"},
						},
					},
				},
			},
			{
				Name:  "test-template",
				Usage: "Test rendering a template",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     flagTemplateFile,
						Usage:    "Template file",
						Required: true,
					},
					&cli.StringFlag{
						Name:  flagPayloadFile,
						Usage: "Payload data file",
					},
					&cli.BoolFlag{
						Name:  flagNoPreviousIssue,
						Usage: "Set `.PreviousIssue` to nil",
					},
				},
				Action: func(c *cli.Context) error {
					if err := actionTestTemplate(c); err != nil {
						return cli.Exit(fmt.Errorf("error: %w", err), 1)
					}
					return nil
				},
			},
		},
	}
}

func buildGitHubClientWithAppCredentials(
	githubURL string, appID int64, installationID int64, privateKey []byte,
) (*github.Client, error) {
	fmt.Printf(
		"Building a GitHub client with GitHub App credentials (app ID: %d, installation ID: %d)...\n",
		appID, installationID,
	)

	tr, err := ghinstallation.New(http.DefaultTransport, appID, installationID, privateKey)
	if err != nil {
		return nil, err
	}

	if githubURL == "" {
		return github.NewClient(&http.Client{Transport: tr}), nil
	}

	tr.BaseURL = githubURL
	return github.NewEnterpriseClient(githubURL, githubURL, &http.Client{Transport: tr})
}

func buildGitHubClientWithToken(githubURL string, token string) (*github.Client, error) {
	fmt.Println("Building a GitHub client with token...")

	ctx := context.TODO()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	if githubURL == "" {
		return github.NewClient(tc), nil
	}

	return github.NewEnterpriseClient(githubURL, githubURL, tc)
}

func templateFromReader(r io.Reader) (*template.Template, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return templateFromString(string(b))
}

func templateFromFile(path string) (*template.Template, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return templateFromString(string(b))
}

func templateFromString(s string) (*template.Template, error) {
	t, err := template.Parse(s)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func actionStart(c *cli.Context) error {
	githubClient, err := func() (*github.Client, error) {
		appID := c.Int64(flagGitHubAppID)
		installationID := c.Int64(flagGitHubAppInstallationID)
		appKey := c.String(flagGitHubAppPrivateKey)
		if appID != 0 && installationID != 0 && appKey != "" {
			return buildGitHubClientWithAppCredentials(c.String(flagGitHubURL), appID, installationID, []byte(appKey))
		}

		if token := c.String(flagGitHubToken); token != "" {
			return buildGitHubClientWithToken(c.String(flagGitHubURL), token)
		}

		return nil, errors.New("GitHub credentials must be specified")
	}()
	if err != nil {
		return err
	}

	bodyReader, err := openReader(c.String(flagBodyTemplateFile), "templates/body.tmpl")
	if err != nil {
		return err
	}
	defer func() {
		if err := bodyReader.Close(); err != nil {
			log.Err(err).Msg("failed to close bodyReader")
		}
	}()
	bodyTemplate, err := templateFromReader(bodyReader)
	if err != nil {
		return err
	}

	titleReader, err := openReader(c.String(flagTitleTemplateFile), "templates/title.tmpl")
	if err != nil {
		return err
	}
	defer func() {
		if err := titleReader.Close(); err != nil {
			log.Err(err).Msg("failed to close titleReader")
		}
	}()
	titleTemplate, err := templateFromReader(titleReader)
	if err != nil {
		return err
	}

	alertIDTemplate, err := templateFromString(c.String(flagAlertIDTemplate))
	if err != nil {
		return err
	}

	var reopenWindow *time.Duration
	if c.IsSet(flagReopenWindow) {
		d := c.Duration(flagReopenWindow)
		reopenWindow = &d
	}

	nt, err := notifier.NewGitHub()
	if err != nil {
		return err
	}
	nt.GitHubClient = githubClient
	nt.Labels = c.StringSlice(flagLabels)
	if nt.Labels == nil {
		nt.Labels = []string{}
	}
	nt.BodyTemplate = bodyTemplate
	nt.TitleTemplate = titleTemplate
	nt.AlertIDTemplate = alertIDTemplate
	nt.AutoCloseResolvedIssues = c.Bool(flagAutoCloseResolvedIssues)
	nt.ReopenWindow = reopenWindow

	router := server.New(nt).Router()
	if err := router.Run(c.String(flagListen)); err != nil {
		return err
	}

	return nil
}

func actionTestTemplate(c *cli.Context) error {
	t, err := templateFromFile(c.String(flagTemplateFile))
	if err != nil {
		return err
	}

	payloadData := defaultPayload
	if path := c.String(flagPayloadFile); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		payloadData = string(b)
	}

	payload := &types.WebhookPayload{}

	dec := json.NewDecoder(strings.NewReader(payloadData))
	err = dec.Decode(payload)
	if err != nil {
		return err
	}

	var previousIssue *github.Issue
	if !c.Bool(flagNoPreviousIssue) {
		previousIssue = &github.Issue{}
		err = json.NewDecoder(strings.NewReader(sampleIssue)).Decode(previousIssue)
		if err != nil {
			return err
		}
	}

	s, err := t.Execute(payload, previousIssue)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", s)

	return nil
}

func openReader(path string, defaultFile string) (io.ReadCloser, error) {
	if path == "" {
		return templates.Open(defaultFile)
	} else {
		return os.Open(path)
	}
}
