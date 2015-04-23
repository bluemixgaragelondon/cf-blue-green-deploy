package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry/cli/plugin"
)

var PluginVersion string

type CfPlugin struct {
	Connection plugin.CliConnection
	Deployer   BlueGreenDeployer
}

func (p *CfPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	p.Connection = cliConnection
	p.Deployer.Setup(cliConnection)

	if len(args) < 2 {
		fmt.Printf("App name must be provided")
		os.Exit(1)
	}

	if !p.Deploy(args) {
		fmt.Println("Smoke tests failed")
		os.Exit(1)
	}
}

func (p *CfPlugin) Deploy(args []string) bool {
	appName := args[1]

	p.Deployer.DeleteAllAppsExceptLiveApp(appName)
	liveApp := p.Deployer.LiveApp(appName)
	newApp := p.Deployer.PushNewApp(appName)

	promoteNewApp := true
	smokeTestScript := ExtractIntegrationTestScript(args)
	if smokeTestScript != "" {
		promoteNewApp = p.Deployer.RunSmokeTests(smokeTestScript, newApp.DefaultRoute().FQDN())
	}

	if promoteNewApp {
		if liveApp != nil {
			p.Deployer.RemapRoutesFromLiveAppToNewApp(*liveApp, newApp)
			p.Deployer.UnmapTemporaryRouteFromNewApp(newApp)
			p.Deployer.UpdateAppNames(liveApp, &newApp)
		} else {
			p.Deployer.UnmapTemporaryRouteFromNewApp(newApp)
			// p.Deployer.UpdateAppName(null, newApp)
		}
		return true
	} else {
		p.Deployer.UnmapTemporaryRouteFromNewApp(newApp)
		p.Deployer.RenameApp(&newApp, appName+"-failed")
		return false
	}
}

func (p *CfPlugin) GetMetadata() plugin.PluginMetadata {
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

func GenerateAppName(base string) string {
	return base + "-new"
}

func ExtractIntegrationTestScript(args []string) string {
	f := flag.NewFlagSet("blue-green-deploy", flag.ExitOnError)
	script := f.String("smoke-test", "", "")
	f.Parse(args[2:])
	return *script
}

func main() {
	p := CfPlugin{
		Deployer: &BlueGreenDeploy{
			ErrorFunc: func(message string, err error) {
				fmt.Printf("%v - %v\n", message, err)
				os.Exit(1)
			},
			Out: os.Stdout,
		},
	}

	plugin.Start(&p)
}
