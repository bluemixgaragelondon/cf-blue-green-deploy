package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/cli/plugin"
)

func main() {

	// Remove timestamps on flags output by logger
	log.SetFlags(0)

	p := CfPlugin{
		Deployer: &BlueGreenDeploy{
			ErrorFunc: func(message string, err error) {
				log.Fatalf("%v - %v", message, err)
			},
			Out: os.Stdout,
		},
	}

	plugin.Start(&p)
}
