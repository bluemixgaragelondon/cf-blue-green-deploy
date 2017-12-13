package main

import (
	"flag"
)

type Args struct {
	SmokeTestPath string
	ManifestPath  string
	AppName       string
	KeepOldApps   bool
}

func NewArgs(osArgs []string) Args {
	args := Args{}
	args.AppName = extractAppName(osArgs)

	// Only use FlagSet so that we can pass string slice to Parse
	f := flag.NewFlagSet("blue-green-deploy", flag.ExitOnError)

	f.StringVar(&args.SmokeTestPath, "smoke-test", "", "")
	f.StringVar(&args.ManifestPath, "f", "", "")
	f.BoolVar(&args.KeepOldApps, "keep-old-apps", false, "")

	f.Parse(extractBgdArgs(osArgs))

	return args
}

func indexOfAppName(osArgs []string) int {
	index := 0
	for i, arg := range osArgs {
		if arg == "blue-green-deploy" || arg == "bgd" {
			index = i + 1
			break
		}
	}
	if len(osArgs) > index {
		return index
	}
	return -1
}

func extractAppName(osArgs []string) string {
	// Assume an app name will be passed - issue #27
	index := indexOfAppName(osArgs)
	if index >= 0 {
		return osArgs[index]
	}
	return ""
}

func extractBgdArgs(osArgs []string) []string {
	index := indexOfAppName(osArgs)
	if index >= 0 && len(osArgs) > index+1 {
		return osArgs[index+1:]
	}

	return []string{}
}
