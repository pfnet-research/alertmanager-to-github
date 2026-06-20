package notifier

import (
	"net/url"
	"testing"

	pkgtemplate "github.com/pfnet-research/alertmanager-to-github/pkg/template"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "trims empties and duplicates",
			input:    []string{" team/platform ", "", "team/platform", "bug", "  "},
			expected: []string{"team/platform", "bug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := normalizeLabels(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGitHubNotifierResolveLabels(t *testing.T) {
	labelTemplate, err := pkgtemplate.Parse("team/{{ .Payload.CommonLabels.team }}")
	require.NoError(t, err)

	emptyTemplate, err := pkgtemplate.Parse("{{ with .Payload.CommonLabels.missing }}{{ . }}{{ end }}")
	require.NoError(t, err)

	duplicatedTemplate, err := pkgtemplate.Parse("team/{{ .Payload.CommonLabels.team }}")
	require.NoError(t, err)

	notifier := &GitHubNotifier{
		LabelTemplates: []*pkgtemplate.Template{labelTemplate, emptyTemplate, duplicatedTemplate},
	}
	payload := &types.WebhookPayload{
		CommonLabels: map[string]string{
			"team": "platform",
		},
	}

	tests := []struct {
		name     string
		params   url.Values
		expected []string
		err      string
	}{
		{
			name:     "renders templates and normalizes labels",
			params:   url.Values{},
			expected: []string{"team/platform"},
		},
		{
			name: "uses labels query param when present",
			params: url.Values{
				"labels": []string{"override, team/platform ,override,,  "},
			},
			expected: []string{"override", "team/platform"},
		},
		{
			name:   "returns template execution errors",
			params: url.Values{},
			err:    `template: template:1:23: executing "template" at <1>: expected string; found 1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentNotifier := notifier
			if tt.err != "" {
				badTemplate, err := pkgtemplate.Parse("team/{{ urlQueryEscape 1 }}")
				require.NoError(t, err)
				currentNotifier = &GitHubNotifier{
					LabelTemplates: []*pkgtemplate.Template{badTemplate},
				}
			}

			labels, err := currentNotifier.resolveLabels(payload, nil, tt.params)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, labels)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, labels)
		})
	}
}
