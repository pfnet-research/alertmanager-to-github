package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookPayloadHasSkipAutoCloseAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		payload  *WebhookPayload
		expected bool
	}{
		{
			name: "no annotations",
			payload: &WebhookPayload{
				Alerts: []WebhookAlert{{
					Labels: map[string]string{
						"job": "example",
					},
				}},
			},
			expected: false,
		},
		{
			name: "has the annotation",
			payload: &WebhookPayload{
				Alerts: []WebhookAlert{{
					Annotations: map[string]string{
						"atg_skip_auto_close": "true",
					},
				}},
			},
			expected: true,
		},
		{
			name: "don't has the annotation",
			payload: &WebhookPayload{
				Alerts: []WebhookAlert{{
					Annotations: map[string]string{
						"description": "example",
					},
				}},
			},
			expected: false,
		},
		{
			name: "no alerts has the annotation",
			payload: &WebhookPayload{
				Alerts: []WebhookAlert{
					{
						Labels: map[string]string{
							"job": "example",
						},
					},
					{
						Labels: map[string]string{
							"job": "example",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "some alerts have the annotation",
			payload: &WebhookPayload{
				Alerts: []WebhookAlert{
					{
						Annotations: map[string]string{
							"atg_skip_auto_close": "true",
						},
					},
					{
						Labels: map[string]string{
							"job": "example",
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.payload.HasSkipAutoCloseAnnotation()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
