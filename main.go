package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

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
	err := p.DeleteOldAppVersions(appName)
	if err != nil {
		fmt.Printf("Could not delete old app version - %s", err.Error())
		os.Exit(1)
	}

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

func (p *BlueGreenDeployPlugin) deleteApps(apps []Application) error {
	for _, app := range apps {
		if _, err := p.Connection.CliCommand("delete", app.Name, "-f", "-r"); err != nil {
			return err
		}
	}

	return nil
}

func (p *BlueGreenDeployPlugin) DeleteOldAppVersions(appName string) error {
	apps, err := p.appsInCurrentSpace()
	if err != nil {
		return err
	}
	return p.deleteApps(FilterOldApps(appName, apps))
}

func (p *BlueGreenDeployPlugin) PushNewAppVersion(appName string) error {
	_, err := p.Connection.CliCommand("push", fmt.Sprintf("%v-%v", appName, "12345678901234"))
	return err
}

func (p *BlueGreenDeployPlugin) appsInCurrentSpace() ([]Application, error) {
	path := fmt.Sprintf("/v2/spaces/%s/summary", getSpaceGuid())

	output, err := p.Connection.CliCommandWithoutTerminalOutput("curl", path)
	if err != nil {
		return nil, err
	}

	apps := struct {
		Apps []Application
	}{}

	json.Unmarshal([]byte(output[0]), &apps)
	return apps.Apps, nil
}

func getSpaceGuid() string {
	configRepo := core_config.NewRepositoryFromFilepath(config_helpers.DefaultFilePath(), func(err error) {
		if err != nil {
			fmt.Printf("Config error: %s", err)
		}
	})

	return configRepo.SpaceFields().Guid
}

func FilterOldApps(appName string, apps []Application) (oldApps []Application) {
	r := regexp.MustCompile(fmt.Sprintf("^%s-[0-9]{14}-old$", appName))
	oldApps = []Application{}
	for _, app := range apps {
		if r.MatchString(app.Name) {
			oldApps = append(oldApps, app)
		}
	}
	return
}

func GenerateAppName(base string) string {
	return fmt.Sprintf("%s-%s", base, time.Now().Format("20060102150405"))
}

func main() {
	plugin.Start(&BlueGreenDeployPlugin{})
}
