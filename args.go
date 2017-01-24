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

	f.Parse(extractBgdArgs(osArgs))

	args.AppName = f.Arg(0)

	return args
}

func extractBgdArgs(osArgs []string) []string {
	for i, arg := range osArgs {
		if arg == "blue-green-deploy" || arg == "bgd" {
			return osArgs[i+1:]
		}
	}

	return []string{}
}
