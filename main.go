package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/cli/plugin"
)

func main() {

	log.SetFlags(0)

	p := CfPlugin{
		Deployer: &BlueGreenDeploy{
			ErrorFunc: func(message string, err error) {
				log.Fatalf("%v - %v", message, err)
			},
			Out: os.Stdout,
		},
	}

	// TODO issue #24 - (Rufus) - not sure if I'm using the plugin correctly, but if I build (go build) and run without arguments
	// I expected to see available arguments but instead the code panics.
	plugin.Start(&p)
}
