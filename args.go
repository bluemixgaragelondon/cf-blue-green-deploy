package main

import (
	"flag"
	"fmt"
)

type Args struct {
	SmokeTestPath string
	ManifestPath  string
	AppName       string
}

// NewArgs is called on arguments passed back cf cli core
// cf cli strips the "cf" for you.
// We currently pick up smoke test, manifest path, and app name
func NewArgs(argsFromCF []string) (*Args, error) {
	args := Args{}

	// Assumption: The first argument to the app is the app name
	// issue #27
	// Hypothetically, we could get app name from manifest if manifest only describes
	// one app

	if len(argsFromCF) <= 1 {
		return nil, fmt.Errorf("No arguments passed to blue-green-deploy")
	}

	// The cf docs say
	//  'Name: You can use any series of alpha-numeric characters, without spaces, as the name of your app.'
	// , therefore just take the first argument as the name, same as cf push does.
	args.AppName = argsFromCF[1]

	// Grab the other args using flags library

	if len(argsFromCF) > 1 {
		// Using FlagSet instead of flag so that we can pass string slice to Parse
		f := flag.NewFlagSet("blue-green-deploy", flag.ExitOnError)
		f.StringVar(&args.SmokeTestPath, "smoke-test", "", "")
		f.StringVar(&args.ManifestPath, "f", "", "")

		// Parse all args but the first which we decided was the app name
		if err := f.Parse(argsFromCF[2:]); err != nil {
			return nil, err
		}
	}
	return &args, nil
}
