package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/manifest"
	"github.com/cloudfoundry/cli/plugin"
	go_i18n "github.com/nicksnyder/go-i18n/i18n"
)

var PluginVersion string

type CfPlugin struct {
	Connection plugin.CliConnection
	Deployer   BlueGreenDeployer
}

func (p *CfPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) > 0 && args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	p.Connection = cliConnection

	defaultCfDomain, err := p.DefaultCfDomain()
	if err != nil {
		fmt.Println("Failed to get default shared domain")
		os.Exit(1)
	}

	p.Deployer.Setup(cliConnection)

	if len(args) < 2 {
		fmt.Println("App name must be provided")
		os.Exit(1)
	}

	if !p.Deploy(defaultCfDomain, manifest.ManifestDiskRepository{}, args) {
		fmt.Println("Smoke tests failed")
		os.Exit(1)
	}
}

func (p *CfPlugin) Deploy(defaultCfDomain string, repo manifest.ManifestRepository, args []string) bool {
	appName := args[1]

	p.Deployer.DeleteAllAppsExceptLiveApp(appName)
	liveApp := p.Deployer.LiveApp(appName)

	newAppName := appName + "-new"

	tempRoute := Route{Host: newAppName, Domain: Domain{Name: defaultCfDomain}}

	newApp := Application{
		Name:   newAppName,
		Routes: []Route{tempRoute},
	}

	p.Deployer.PushNewApp(&newApp, tempRoute)

	f := ManifestAppFinder{AppName: appName, Repo: repo}
	if manifestRoutes := f.RoutesFromManifest(defaultCfDomain); manifestRoutes != nil {
		newApp.Routes = append(newApp.Routes, manifestRoutes...)
	}

	promoteNewApp := true
	smokeTestScript := ExtractIntegrationTestScript(args)
	if smokeTestScript != "" {
		promoteNewApp = p.Deployer.RunSmokeTests(smokeTestScript, tempRoute.FQDN())
	}

	p.Deployer.UnmapTemporaryRouteFromNewApp(newApp.Name, tempRoute)

	if promoteNewApp {
		if liveApp != nil {
			p.Deployer.CopyLiveAppRoutesToNewApp(liveApp.Name, newApp.Name, liveApp.Routes)
			p.Deployer.MapAllRoutes(newAppName, newApp.Routes)
			p.Deployer.RenameApp(liveApp.Name, appName+"-old")
			p.Deployer.RenameApp(newAppName, appName)
			p.Deployer.UnmapRoutesFromOldApp(appName+"-old", liveApp.Routes)
		} else {
			p.Deployer.MapAllRoutes(newAppName, newApp.Routes)
			p.Deployer.RenameApp(newAppName, appName)
		}
		return true
	} else {
		p.Deployer.RenameApp(newAppName, appName+"-failed")
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

func (p *CfPlugin) DefaultCfDomain() (domain string, err error) {
	var res []string
	if res, err = p.Connection.CliCommandWithoutTerminalOutput("curl", "/v2/shared_domains"); err != nil {
		return
	}

	response := struct {
		Resources []struct {
			Entity struct {
				Name string
			}
		}
	}{}

	if err = json.Unmarshal([]byte(res[0]), &response); err != nil {
		return
	}

	domain = response.Resources[0].Entity.Name
	return
}

func ExtractIntegrationTestScript(args []string) string {
	f := flag.NewFlagSet("blue-green-deploy", flag.ExitOnError)
	script := f.String("smoke-test", "", "")
	f.Parse(args[2:])
	return *script
}

func main() {
	// T needs to point to a translate func, otherwise cf internals blow up
	i18n.T, _ = go_i18n.Tfunc("")
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
