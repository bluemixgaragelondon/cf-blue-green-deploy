package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudfoundry/cli/plugin"
)

type BlueGreenDeployPlugin struct {
	Connection plugin.CliConnection
}

func (p *BlueGreenDeployPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	p.Connection = cliConnection

	p.DeleteOldAppVersions("cf-blue-green-deploy-test-app")

	fmt.Println("Hello world! The sky is all blue/green.")
}

func (p *BlueGreenDeployPlugin) GetMetadata() plugin.PluginMetadata {
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

func (p *BlueGreenDeployPlugin) OldAppVersionList(appName string) ([]string, error) {
	r := regexp.MustCompile("app-name-[0-9]{14}-old")
	apps, err := p.Connection.CliCommandWithoutTerminalOutput("apps")
	oldApps := r.FindAllString(strings.Join(apps, " "), -1)

	return oldApps, err
}

func (p *BlueGreenDeployPlugin) DeleteApps(appNames []string) error {
	for _, appName := range appNames {
		if _, err := p.Connection.CliCommand("delete", appName, "-f", "-r"); err != nil {
			return err
		}
	}

	return nil
}

func (p *BlueGreenDeployPlugin) DeleteOldAppVersions(appName string) error {
	appNames, err := p.OldAppVersionList(appName)
	p.DeleteApps(appNames)
	return err
}

func main() {
	plugin.Start(&BlueGreenDeployPlugin{})
}
