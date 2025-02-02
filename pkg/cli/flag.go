package cli

import (
	"github.com/urfave/cli/v2"
)

// cli.DurationFlag without the default value "0s"
type noDefaultDurationFlag struct {
	cli.DurationFlag
}

func (f *noDefaultDurationFlag) GetDefaultText() string {
	return ""
}

func (f *noDefaultDurationFlag) String() string {
	return cli.FlagStringer(f)
}
