package main

import (
	"flag"
)

type Args struct {
	SmokeTestPath string
	ManifestPath  string
	AppName       string
}

func NewArgs(osArgs []string) Args {
	args := Args{}

	// Only use FlagSet so that we can pass string slice to Parse
	f := flag.NewFlagSet("", flag.ExitOnError)

	f.StringVar(&args.SmokeTestPath, "smoke-test", "", "")
	f.StringVar(&args.ManifestPath, "f", "", "")

	f.Parse(osArgs[2:])

	args.AppName = f.Arg(0)

	return args
}
