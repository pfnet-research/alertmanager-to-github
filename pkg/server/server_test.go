package server

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"github.com/stretchr/testify/assert"
)

type dummyNotifier struct {
	payloads []*types.WebhookPayload
}

func (n *dummyNotifier) Notify(ctx context.Context, payload *types.WebhookPayload, params url.Values) error {
	n.payloads = append(n.payloads, payload)
	return nil
}

func TestV1Webhook(t *testing.T) {
	nt := &dummyNotifier{}
	router := New(nt).Router()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/webhook", strings.NewReader(`{
  "version": "4",
  "groupKey": "group1",
  "truncatedAlerts": 2,
  "status": "firing",
  "receiver": "receiver1",
  "groupLabels": {"a": "b"},
  "commonLabels": {"c": "d"},
  "commonAnnotations": {"e": "f"},
  "externalURL": "http://alert.example.com",
  "alerts": [
    {
      "status": "firing",
      "labels": {"g": "h"},
      "annotations": {"i": "j"},
      "startsAt": "2020-01-01T12:34:56Z",
      "endsAt": "2020-01-02T12:34:56Z",
      "generatorURL": "http://alert.example.com"
    }
  ]
}`))
	router.ServeHTTP(w, req)
	if !assert.Equal(t, 200, w.Code) {
		t.Log(w.Body.String())
	}
	if assert.Len(t, nt.payloads, 1) {
		assert.Equal(t, nt.payloads[0], &types.WebhookPayload{
			Version:           "4",
			GroupKey:          "group1",
			TruncatedAlerts:   2,
			Status:            "firing",
			Receiver:          "receiver1",
			GroupLabels:       map[string]string{"a": "b"},
			CommonLabels:      map[string]string{"c": "d"},
			CommonAnnotations: map[string]string{"e": "f"},
			ExternalURL:       "http://alert.example.com",
			Alerts: []types.WebhookAlert{
				{
					Status:       "firing",
					Labels:       map[string]string{"g": "h"},
					Annotations:  map[string]string{"i": "j"},
					StartsAt:     mustParseTime("2020-01-01T12:34:56Z"),
					EndsAt:       mustParseTime("2020-01-02T12:34:56Z"),
					GeneratorURL: "http://alert.example.com",
				},
			},
		})
	}
}

func TestMetrics(t *testing.T) {
	nt := &dummyNotifier{}
	router := New(nt).Router()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)
	if !assert.Equal(t, 200, w.Code) {
		t.Log(w.Body.String())
	}
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
