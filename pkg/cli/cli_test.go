package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenReader(t *testing.T) {
	tests := []struct {
		name string
		err  string
	}{
		{
			name: "templates/body.tmpl",
			err:  "",
		},
		{
			name: "templates/title.tmpl",
			err:  "",
		},
		{
			name: "templates/unknown.tmpl",
			err:  "open templates/unknown.tmpl: file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := openReader("", tt.name)
			if err == nil {
				if tt.err != "" {
					t.Errorf("expected %v, but got nil", tt.err)
				}
			} else if tt.err != err.Error() {
				t.Errorf("expected %v, but got %v", tt.err, err)
			}
		})
	}
}

func TestLabelTemplatesFromStrings(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		err    string
	}{
		{
			name:   "static labels",
			labels: []string{"bug", "team/{{ .Payload.CommonLabels.team }}"},
		},
		{
			name:   "invalid template",
			labels: []string{"team/{{ .Payload.CommonLabels.team }"},
			err:    "template: template:1: unexpected \"}\" in operand",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templates, err := labelTemplatesFromStrings(tt.labels)
			if tt.err == "" {
				assert.NoError(t, err)
				assert.Len(t, templates, len(tt.labels))
				return
			}

			assert.EqualError(t, err, tt.err)
			assert.Nil(t, templates)
		})
	}
}
