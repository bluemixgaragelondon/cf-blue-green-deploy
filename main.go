package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/cloudfoundry/cli/plugin"
)

var PluginVersion string

type BlueGreenDeployPlugin struct {
	Connection      plugin.CliConnection
	BlueGreenDeploy BlueGreenDeploy
}

func (p *BlueGreenDeployPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	p.Connection = cliConnection
	p.BlueGreenDeploy.Connection = cliConnection

	if len(args) < 2 {
		fmt.Printf("App name must be provided")
		os.Exit(1)
	}

	appsInSpace, err := p.BlueGreenDeploy.AppLister.AppsInCurrentSpace()
	if err != nil {
		fmt.Printf("Could not load apps in space, are you logged in? - %s", err.Error())
		os.Exit(1)
	}

	appName := args[1]
	previousLiveApp, oldAppVersions := FilterApps(appName, appsInSpace)

	p.BlueGreenDeploy.DeleteAppVersions(oldAppVersions)

	newLiveApp := p.BlueGreenDeploy.PushNewAppVersion(appName)

	smokeTestScript := ExtractIntegrationTestScript(args)
	if smokeTestScript != "" {
		p.BlueGreenDeploy.RunSmokeTests(smokeTestScript, "google.co.uk")
	}

	if previousLiveApp != nil {
		p.BlueGreenDeploy.RemapRoutesFromLiveAppToNewApp(*previousLiveApp, newLiveApp)
	}

	fmt.Printf("Deployed %s", newLiveApp.Name)
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
					Usage: "blue-green-deploy APP_NAME [--smoke-test TEST_SCRIPT]",
					Options: map[string]string{
						"smoke-test": "The test script to run.",
					},
				},
			},
		},
	}
}

func FilterApps(appName string, apps []Application) (currentApp *Application, oldApps []Application) {
	r := regexp.MustCompile(fmt.Sprintf("^%s-[0-9]{14}(-old)?$", appName))
	for index, app := range apps {
		if !r.MatchString(app.Name) {
			continue
		}

		if strings.HasSuffix(app.Name, "-old") {
			oldApps = append(oldApps, app)
		} else {
			currentApp = &apps[index]
		}
	}
	return
}

func GenerateAppName(base string) string {
	return fmt.Sprintf("%s-%s", base, time.Now().Format("20060102150405"))
}

func ExtractIntegrationTestScript(args []string) string {
	f := flag.NewFlagSet("blue-green-deploy", flag.ExitOnError)
	script := f.String("smoke-test", "", "")
	f.Parse(args[2:])
	return *script
}

func RunIntegrationTestScript(script, appFQDN string) (string, error) {
	out, err := exec.Command(script, appFQDN).CombinedOutput()

	return string(out), err
}

func main() {
	p := BlueGreenDeployPlugin{
		BlueGreenDeploy: BlueGreenDeploy{
			ErrorFunc: func(message string, err error) {
				fmt.Printf("%v - %v\n", message, err)
				os.Exit(1)
			},
		},
	}

	plugin.Start(&p)
}
