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
	liveAppName, liveAppRoutes := p.Deployer.LiveApp(appName)

	newAppName := appName + "-new"
	tempRoute := Route{Host: newAppName, Domain: Domain{Name: defaultCfDomain}}
	p.Deployer.PushNewApp(newAppName, tempRoute)

	promoteNewApp := true
	smokeTestScript := ExtractIntegrationTestScript(args)
	if smokeTestScript != "" {
		promoteNewApp = p.Deployer.RunSmokeTests(smokeTestScript, tempRoute.FQDN())
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

func (p *CfPlugin) GetNewAppRoutes(appName string, defaultCfDomain string, repo manifest.ManifestRepository, liveAppRoutes []Route) []Route{
	newAppRoutes := []Route{}
	f := ManifestAppFinder{AppName: appName, Repo: repo}
	if manifestRoutes := f.RoutesFromManifest(defaultCfDomain); manifestRoutes != nil {
		newAppRoutes = append(newAppRoutes, manifestRoutes...)
	}
	uniqueRoutes := p.UnionRouteLists(newAppRoutes, liveAppRoutes)

	if len(uniqueRoutes) == 0 {
		uniqueRoutes = append(uniqueRoutes, Route{Host: appName, Domain: Domain{Name: defaultCfDomain}})
	}
	return uniqueRoutes
}

func (p *CfPlugin) UnionRouteLists(listA []Route, listB []Route) []Route {
	duplicateList := append(listA, listB...)

	routesSet := make(map[Route]struct{})

	for _, route := range duplicateList {
		routesSet[route] = struct{}{}
	}

	uniqueRoutes := []Route{}
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
					Usage: "blue-green-deploy APP_NAME [--smoke-test TEST_SCRIPT] [--manifest MANIFEST_FILE]",
					Options: map[string]string{
						"-smoke-test": "The test script to run.",
						"-manifest": "manifest file to use instead of manifest.yml.",
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
	f := flag.NewFlagSet("blue-green-deploy", flag.ContinueOnError)
	f.String("manifest", "", "") // Necessary, otherwise it will complain "provided but not defined"
	script := f.String("smoke-test", "", "")
	if err := f.Parse(args[2:]); err != nil {
		fmt.Println(err)
		os.Exit(3)
	}
	return *script
}

func main() {
	// T needs to point to a translate func, otherwise cf internals blow up
	i18n.T, _ = go_i18n.Tfunc("")

	flag := flag.NewFlagSet("blue-green-deploy", flag.ContinueOnError)
	manifest := flag.String("manifest", "", "")
	flag.String("smoke-test", "", "") // Necessary, otherwise it will complain "provided but not defined"

	// Args format is <exec_name> <pid> <command> <args>...
	if len(os.Args) > 3 {
		if err := flag.Parse(os.Args[4:]); err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
	}

	p := CfPlugin{
		Deployer: &BlueGreenDeploy{
			ErrorFunc: func(message string, err error) {
				fmt.Printf("%v - %v\n", message, err)
				os.Exit(1)
			},
			Out: os.Stdout,
			ManifestPath: *manifest,
		},
	}

	plugin.Start(&p)
}
