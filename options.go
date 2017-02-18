package main

import (
	"log"

	"github.com/SeeSpotRun/coerce"
	docopt "github.com/docopt/docopt-go"
)

type options struct {
	githubRepo   string
	assetPattern string
	binDir       string
	store        string
}

const docstring = `
Usage: install_releases [options] <github-repo> <asset-pattern> <bin-dir>

Options:
	--store=<dir>  The base directory to download releases to. [default: /var/store]
`

func parseOpts() *options {
	parsed, err := docopt.Parse(docstring, nil, true, "", false)
	if err != nil {
		log.Fatal(err)
	}

	opts := options{}
	err = coerce.Struct(&opts, parsed, "-%s", "--%s", "<%s>")
	if err != nil {
		log.Fatal(err)
	}

	return &opts
}
