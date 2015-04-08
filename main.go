package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/cloudfoundry/cli/cf/configuration/config_helpers"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/plugin"
)

var PluginVersion string

type Application struct {
	Name   string
	Routes []struct {
		Host   string
		Domain struct {
			Name string
		}
	}
	Urls []string
}

type BlueGreenDeployPlugin struct {
	Connection plugin.CliConnection
}

func (p *BlueGreenDeployPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	p.Connection = cliConnection

	if len(args) < 2 {
		fmt.Printf("App name must be provided")
		os.Exit(1)
	}

	appsInSpace, err := p.appsInCurrentSpace()
	if err != nil {
		fmt.Printf("Could not load apps in space, are you logged in? - %s", err.Error())
		os.Exit(1)
	}

	appName := args[1]
	err = p.DeleteOldAppVersions(appName, appsInSpace)
	if err != nil {
		fmt.Printf("Could not delete old app version - %s", err.Error())
		os.Exit(1)
	}

	newAppName, err := p.PushNewAppVersion(appName)
	if err != nil {
		fmt.Printf("Could not push new version - %s", err.Error())
		os.Exit(1)
	}

	fmt.Println("Deployed %s", newAppName)
}

func (p *BlueGreenDeployPlugin) GetMetadata() plugin.PluginMetadata {
	var major, minor, build int
	fmt.Sscanf(PluginVersion, "%d.%d.%d", &major, &minor, &build)

	return plugin.PluginMetadata{
		Name: "blue-green-deploy",
		Version: plugin.VersionType{
			Major: major,
			Minor: minor,
			Build: build,
		},
		Commands: []plugin.Command{
			{
				Name:     "blue-green-deploy",
				Alias:    "bgd",
				HelpText: "Zero-downtime deploys with smoke tests",
				UsageDetails: plugin.Usage{
					Usage: "blue-green-deploy APP_NAME [--integration-test TEST_SCRIPT]",
					Options: map[string]string{
						"integration-test": "The test script to run.",
					},
				},
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

func (p *BlueGreenDeployPlugin) DeleteOldAppVersions(appName string, apps []Application) error {
	return p.deleteApps(FilterOldApps(appName, apps))
}

func (p *BlueGreenDeployPlugin) PushNewAppVersion(appName string) (newAppName string, err error) {
	newAppName = GenerateAppName(appName)
	_, err = p.Connection.CliCommand("push", newAppName)
	return
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

func ExtractIntegrationTestScript(args []string) string {
	f := flag.NewFlagSet("blue-green-deploy", flag.ExitOnError)
	script := f.String("integration-test", "", "")
	f.Parse(args[2:])
	return *script
}

func RunIntegrationTestScript(script, appFQDN string) (string, error) {
	out, err := exec.Command(script, appFQDN).CombinedOutput()

	return string(out), err
}

func main() {
	plugin.Start(&BlueGreenDeployPlugin{})
}
