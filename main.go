package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudfoundry/cli/plugin"
)

type BlueGreenDeploymentPlugin struct {
	Connection plugin.CliConnection
}

func (p *BlueGreenDeploymentPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	p.Connection = cliConnection

	fmt.Println("Hello world! The sky is all blue/green.")
}

func (p *BlueGreenDeploymentPlugin) GetMetadata() plugin.PluginMetadata {
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

func (p *BlueGreenDeploymentPlugin) OldAppVersionList(appName string) []string {
	r := regexp.MustCompile("app-name-[0-9]{14}-old")
	apps, _ := p.Connection.CliCommandWithoutTerminalOutput("apps")
	oldApps := r.FindAllString(strings.Join(apps, " "), -1)

	return oldApps
}

func main() {
	plugin.Start(&BlueGreenDeploymentPlugin{})
}
