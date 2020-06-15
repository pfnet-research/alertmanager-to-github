package template

import (
	"bytes"
	"encoding/json"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"net/url"
	"text/template"
	"time"
)

type Vars struct {
	Payload *types.WebhookPayload
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

func (t *Template) Execute(payload *types.WebhookPayload) (string, error) {
	vars := &Vars{
		Payload: payload,
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
