package cli

import (
	"testing"
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
