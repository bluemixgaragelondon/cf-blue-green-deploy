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

	f := flag.NewFlagSet("blue-green-deploy", flag.ExitOnError)
	f.StringVar(&args.SmokeTestPath, "smoke-test", "", "")
	f.StringVar(&args.ManifestPath, "f", "", "")

	// Parse. We skip the first element because that is "bgd" or "blue-green-deploy"
	if err := f.Parse(argsFromCF[1:]); err != nil {
		return nil, err
	}

	// Assume if there is an argument left over after parsing including the element after
	// plugin name, it is the app name.
	// The cf docs say
	//  'Name: You can use any series of alpha-numeric characters, without spaces, as the name of your app.'
	if f.Arg(0) != "" {
		args.AppName = f.Arg(0)
	}

	// Parsing appears to stop if the first argument isn't a match. In our case this
	// is unwanted because the first argument could or could not be the app name.
	// So... if the array contains enough elements, parse again starting
	// on index 2 of our input array.
	if len(argsFromCF) > 2 && (args.ManifestPath == "" || args.SmokeTestPath == "") {
		if err := f.Parse(argsFromCF[2:]); err != nil {
			return nil, err
		}
	}

	return &args, nil
}
