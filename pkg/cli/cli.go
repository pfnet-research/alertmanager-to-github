package cli

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/google/go-github/v43/github"
	"github.com/pfnet-research/alertmanager-to-github/pkg/notifier"
	"github.com/pfnet-research/alertmanager-to-github/pkg/server"
	"github.com/pfnet-research/alertmanager-to-github/pkg/template"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"
)

const flagListen = "listen"
const flagGitHubURL = "github-url"
const flagLabels = "labels"
const flagBodyTemplateFile = "body-template-file"
const flagTitleTemplateFile = "title-template-file"
const flagGitHubToken = "github-token"
const flagAlertIDTemplate = "alert-id-template"
const flagTemplateFile = "template-file"
const flagPayloadFile = "payload-file"
const flagAutoCloseResolvedIssues = "auto-close-resolved-issues"

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
					&cli.StringFlag{
						Name:     flagGitHubToken,
						Required: true,
						Usage:    "GitHub API token (command line argument is not recommended)",
						EnvVars:  []string{"ATG_GITHUB_TOKEN"},
					},
					&cli.BoolFlag{
						Name:     flagAutoCloseResolvedIssues,
						Required: false,
						Value: true,
						Usage:    "Should issues be automatically closed when resolved",
						EnvVars:  []string{"ATG_AUTO_CLOSE_RESOLVED_ISSUES"},
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

func buildGitHubClient(githubURL string, token string) (*github.Client, error) {
	var err error
	var client *github.Client

	ctx := context.TODO()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	if githubURL == "" {
		client = github.NewClient(tc)
	} else {
		client, err = github.NewEnterpriseClient(githubURL, githubURL, tc)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func templateFromReader(r io.Reader) (*template.Template, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return templateFromString(string(b))
}

func templateFromFile(path string) (*template.Template, error) {
	b, err := ioutil.ReadFile(path)
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
	githubClient, err := buildGitHubClient(c.String(flagGitHubURL), c.String(flagGitHubToken))
	if err != nil {
		return err
	}

	bodyReader, err := openReader(c.String(flagBodyTemplateFile), "templates/body.tmpl")
	if err != nil {
		return err
	}
	defer bodyReader.Close()
	bodyTemplate, err := templateFromReader(bodyReader)
	if err != nil {
		return err
	}

	titleReader, err := openReader(c.String(flagTitleTemplateFile), "templates/title.tmpl")
	if err != nil {
		return err
	}
	defer titleReader.Close()
	titleTemplate, err := templateFromReader(titleReader)
	if err != nil {
		return err
	}

	alertIDTemplate, err := templateFromString(c.String(flagAlertIDTemplate))
	if err != nil {
		return err
	}

	nt, err := notifier.NewGitHub()
	if err != nil {
		return err
	}
	nt.GitHubClient = githubClient
	nt.Labels = c.StringSlice(flagLabels)
	nt.BodyTemplate = bodyTemplate
	nt.TitleTemplate = titleTemplate
	nt.AlertIDTemplate = alertIDTemplate
	nt.AutoCloseResolvedIssues = c.Bool(flagAutoCloseResolvedIssues)

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
		b, err := ioutil.ReadFile(path)
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

	s, err := t.Execute(payload)
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
