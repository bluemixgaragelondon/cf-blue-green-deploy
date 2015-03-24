package main

import (
	"fmt"

	"github.com/cloudfoundry/cli/plugin"
)

type BgdPlugin struct {
}

func (p *BgdPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	fmt.Println("Hello world! The sky is all blue/green.")
}

func (p *BgdPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "blue-green-deploy",
		Commands: []plugin.Command{
			{
				Name:     "blue-green-deploy",
				Alias:    "bgd",
				HelpText: "Do zero-time deploys in a non-sucky way",
			},
		},
	}
}

func main() {
	plugin.Start(&BgdPlugin{})
}
