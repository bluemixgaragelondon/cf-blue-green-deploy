package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

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

	appName := args[1]

	p.Deployer.DeleteAllAppsExceptLiveApp(appName)
	liveApp := p.Deployer.LiveApp(appName)
	newApp := p.Deployer.PushNewApp(appName)

	smokeTestScript := ExtractIntegrationTestScript(args)
	if smokeTestScript != "" {
		p.Deployer.RunSmokeTests(smokeTestScript, newApp.DefaultRoute().FQDN())
	}

	if liveApp != nil {
		p.Deployer.RemapRoutesFromLiveAppToNewApp(*liveApp, newApp)
		p.Deployer.MarkAppAsOld(liveApp)
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
	return base
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
	p := CfPlugin{
		Deployer: &BlueGreenDeploy{
			ErrorFunc: func(message string, err error) {
				fmt.Printf("%v - %v\n", message, err)
				os.Exit(1)
			},
		},
	}

	plugin.Start(&p)
}
