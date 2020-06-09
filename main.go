package main

import (
	"github.com/rs/zerolog/log"
	"github.com/pfnet-research/alertmanager-to-github/pkg/cli"
	"os"
)

func main() {
	err := cli.App().Run(os.Args)
	if err != nil {
		log.Fatal().Err(err)
	}
}
