package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/cloudfoundry/cli/cf/configuration/config_helpers"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/plugin"
)

type Application struct {
	Name string
}

type BlueGreenDeployPlugin struct {
	Connection plugin.CliConnection
}

func (p *BlueGreenDeployPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	p.Connection = cliConnection

	if len(args) < 2 {
		fmt.Printf("appname must be specified")
		os.Exit(1)
	}

	appName := args[1]
	p.DeleteOldAppVersions(appName)

	fmt.Println("Hello world! The sky is all blue/green.")
}

func (p *BlueGreenDeployPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "blue-green-deploy",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 1,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "blue-green-deploy",
				Alias:    "bgd",
				HelpText: "Do zero-time deploys in a non-sucky way",
			},
		},
	}
}

func (p *BlueGreenDeployPlugin) OldAppVersionList(appName string) (oldApps []Application, err error) {
	apps, err := p.appsInCurrentSpace()
	if err != nil {
		return
	}
	oldApps = filterOldApps(appName, apps)
	return
}

func (p *BlueGreenDeployPlugin) DeleteApps(apps []Application) error {
	for _, app := range apps {
		if _, err := p.Connection.CliCommand("delete", app.Name, "-f", "-r"); err != nil {
			return err
		}
	}

	return nil
}

func (p *BlueGreenDeployPlugin) DeleteOldAppVersions(appName string) error {
	appNames, err := p.OldAppVersionList(appName)
	if err != nil {
		return err
	}
	return p.DeleteApps(appNames)
}

func (p *BlueGreenDeployPlugin) appsInCurrentSpace() ([]Application, error) {
	var apps []Application
	path := fmt.Sprintf("/v2/spaces/%s/summary", getSpaceGuid())

	output, err := p.Connection.CliCommandWithoutTerminalOutput("curl", path)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(output[0]), &apps)
	return apps, nil
}

func getSpaceGuid() string {
	configRepo := core_config.NewRepositoryFromFilepath(config_helpers.DefaultFilePath(), func(err error) {
		if err != nil {
			fmt.Printf("Config error: %s", err)
		}
	})

	return configRepo.SpaceFields().Guid
}

func filterOldApps(appName string, apps []Application) (oldApps []Application) {
	r := regexp.MustCompile(fmt.Sprintf("%s-[0-9]{14}-old", appName))
	oldApps = []Application{}
	for _, app := range apps {
		if r.MatchString(app.Name) {
			oldApps = append(oldApps, app)
		}
	}
	return
}

func main() {
	plugin.Start(&BlueGreenDeployPlugin{})
}
