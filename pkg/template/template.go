package template

import (
	"bytes"
	"encoding/json"
	"net/url"
	"text/template"
	"time"

	"github.com/google/go-github/v54/github"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
)

type Vars struct {
	Payload       *types.WebhookPayload
	PreviousIssue *github.Issue
}

type Template struct {
	inner *template.Template
}

func Parse(s string) (*Template, error) {
	funcs := map[string]interface{}{
		"urlQueryEscape": url.QueryEscape,
		"json":           marshalToJSON,
		"timeNow":        timeNow,
	}
	t, err := template.New("template").Funcs(funcs).Parse(s)
	if err != nil {
		return nil, err
	}
	return &Template{inner: t}, nil
}

func (t *Template) Execute(payload *types.WebhookPayload, previousIssue *github.Issue) (string, error) {
	vars := &Vars{
		Payload:       payload,
		PreviousIssue: previousIssue,
	}

	var buf bytes.Buffer
	if err := t.inner.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func marshalToJSON(obj interface{}) (string, error) {
	jsonb, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(jsonb), nil
}

func timeNow() time.Time {
	return time.Now()
}
