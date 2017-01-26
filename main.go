package main

import (
	"encoding/json"
	"fmt"
	"os"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/from-cf-codebase/manifest"
	"strings"
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

	argsStruct := NewArgs(args)

	p.Connection = cliConnection

	defaultCfDomain, err := p.DefaultCfDomain()
	if err != nil {
		fmt.Println("Failed to get default shared domain")
		os.Exit(1)
	}

	p.Deployer.Setup(cliConnection)

	if argsStruct.AppName == "" {
		fmt.Println("App name must be provided")
		os.Exit(1)
	}

	if !p.Deploy(defaultCfDomain, manifest.DiskRepository{}, argsStruct) {
		fmt.Println("Smoke tests failed")
		os.Exit(1)
	}
}

func (p *CfPlugin) Deploy(defaultCfDomain string, repo manifest.Repository, args Args) bool {
	appName := args.AppName

	p.Deployer.DeleteAllAppsExceptLiveApp(appName)
	liveAppName, liveAppRoutes := p.Deployer.LiveApp(appName)

	newAppName := appName + "-new"
	tempRoute := plugin_models.GetApp_RouteSummary{Host: newAppName, Domain: plugin_models.GetApp_DomainFields{Name: defaultCfDomain}}
	p.Deployer.PushNewApp(newAppName, tempRoute, args.ManifestPath)

	promoteNewApp := true
	smokeTestScript := args.SmokeTestPath
	if smokeTestScript != "" {
		promoteNewApp = p.Deployer.RunSmokeTests(smokeTestScript, FQDN(tempRoute))
	}

	newAppRoutes := p.GetNewAppRoutes(appName, defaultCfDomain, repo, liveAppRoutes)

	p.Deployer.UnmapRoutesFromApp(newAppName, tempRoute)

	if promoteNewApp {
		if liveAppName != "" {
			p.Deployer.MapRoutesToApp(newAppName, newAppRoutes...)
			p.Deployer.RenameApp(liveAppName, appName+"-old")
			p.Deployer.RenameApp(newAppName, appName)
			p.Deployer.UnmapRoutesFromApp(appName+"-old", liveAppRoutes...)
		} else {
			p.Deployer.MapRoutesToApp(newAppName, newAppRoutes...)
			p.Deployer.RenameApp(newAppName, appName)
		}
		return true
	} else {
		p.Deployer.RenameApp(newAppName, appName+"-failed")
		return false
	}
}

func (p *CfPlugin) GetNewAppRoutes(appName string, defaultCfDomain string, repo manifest.Repository, liveAppRoutes []plugin_models.GetApp_RouteSummary) []plugin_models.GetApp_RouteSummary {
	newAppRoutes := []plugin_models.GetApp_RouteSummary{}
	f := ManifestAppFinder{AppName: appName, Repo: repo}
	if manifestRoutes := f.RoutesFromManifest(defaultCfDomain); manifestRoutes != nil {
		newAppRoutes = append(newAppRoutes, manifestRoutes...)
	}
	uniqueRoutes := p.UnionRouteLists(newAppRoutes, liveAppRoutes)

	if len(uniqueRoutes) == 0 {
		uniqueRoutes = append(uniqueRoutes, plugin_models.GetApp_RouteSummary{Host: appName, Domain: plugin_models.GetApp_DomainFields{Name: defaultCfDomain}})
	}
	return uniqueRoutes
}

func (p *CfPlugin) UnionRouteLists(listA []plugin_models.GetApp_RouteSummary, listB []plugin_models.GetApp_RouteSummary) []plugin_models.GetApp_RouteSummary {
	duplicateList := append(listA, listB...)

	routesSet := make(map[plugin_models.GetApp_RouteSummary]struct{})

	for _, route := range duplicateList {
		routesSet[route] = struct{}{}
	}

	uniqueRoutes := []plugin_models.GetApp_RouteSummary{}
	for route := range routesSet {
		uniqueRoutes = append(uniqueRoutes, route)
	}
	return uniqueRoutes
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
					Usage: "blue-green-deploy APP_NAME [--smoke-test TEST_SCRIPT] [-f MANIFEST_FILE]",
					Options: map[string]string{
						"smoke-test": "The test script to run.",
						"f":          "Path to manifest",
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

	var json_string string
	json_string = strings.Join(res, "\n")

	if err = json.Unmarshal([]byte(json_string), &response); err != nil {
		return
	}

	domain = response.Resources[0].Entity.Name
	return
}

func FQDN(r plugin_models.GetApp_RouteSummary) string {
	return fmt.Sprintf("%v.%v", r.Host, r.Domain.Name)
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
