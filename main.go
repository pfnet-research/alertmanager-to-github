package main

import (
	"context"
	"os"

	"github.com/pfnet-research/alertmanager-to-github/pkg/cli"
	"github.com/rs/zerolog/log"
)

func main() {
	err := cli.NewCommand().Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal().Err(err)
	}
}
