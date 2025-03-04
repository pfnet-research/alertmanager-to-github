package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestNoDefaultDurationFlag(t *testing.T) {
	writer := bytes.Buffer{}
	app := &cli.App{
		Name:  "app",
		Usage: "app usage",
		Flags: []cli.Flag{
			&noDefaultDurationFlag{
				cli.DurationFlag{
					Name:  "no-default-flag",
					Usage: "no default flag usage",
				},
			},
		},
		Writer: &writer,
	}

	err := app.Run([]string{"help"})

	assert.NoError(t, err)
	assert.Contains(t, writer.String(), "--no-default-flag value  no default flag usage\n")
	assert.NotContains(t, writer.String(), "default: 0s")
}
