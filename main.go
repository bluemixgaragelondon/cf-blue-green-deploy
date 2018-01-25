package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
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
		log.Fatalf("Failed to get default shared domain: %v", err)
	}

	p.Deployer.Setup(cliConnection)

	if argsStruct.AppName == "" {
		log.Fatal("App name was empty, must be provided.")
	}

	reader := manifest.FileManifestReader{argsStruct.ManifestPath}
	if !p.Deploy(defaultCfDomain, &reader, argsStruct) {
		log.Fatal("Smoke tests failed")
	}
}

func (p *CfPlugin) Deploy(defaultCfDomain string, manifestReader manifest.ManifestReader, args Args) bool {
	appName := args.AppName

	p.Deployer.DeleteAllAppsExceptLiveApp(appName)
	liveAppName, liveAppRoutes := p.Deployer.LiveApp(appName)

	manifestScaleParameters := p.GetScaleFromManifest(appName, defaultCfDomain, manifestReader)

	newAppName := appName + "-new"

	// Add route so that we can run the smoke tests
	tempRoute := plugin_models.GetApp_RouteSummary{Host: newAppName, Domain: plugin_models.GetApp_DomainFields{Name: defaultCfDomain}}

	// If deploy is unsuccessful, p.ErrorFunc will be called which exits.
	p.Deployer.PushNewApp(newAppName, tempRoute, args.ManifestPath, manifestScaleParameters)

	if liveAppName != "" {
		p.Deployer.SetSshAccess(newAppName, p.Deployer.CheckSshEnablement(appName))
	}
	promoteNewApp := true
	smokeTestScript := args.SmokeTestPath
	if smokeTestScript != "" {
		promoteNewApp = p.Deployer.RunSmokeTests(smokeTestScript, FQDN(tempRoute))
	}

	// TODO We're overloading 'new' here for both the staging app and the 'finished' app, which is confusing
	newAppRoutes := p.GetNewAppRoutes(args.AppName, defaultCfDomain, manifestReader, liveAppRoutes)

	p.Deployer.UnmapRoutesFromApp(newAppName, tempRoute)

	if promoteNewApp {
		// If there is a live app, we want to disassociate the routes with the old version of the app
		// and instead update the routes to use the new version.
		if liveAppName != "" {
			p.Deployer.MapRoutesToApp(newAppName, newAppRoutes...)
			p.Deployer.RenameApp(liveAppName, appName+"-old")
			p.Deployer.RenameApp(newAppName, appName)
			p.Deployer.UnmapRoutesFromApp(appName+"-old", liveAppRoutes...)
		} else {
			// If there is no live app, we only need to add our new routes.
			p.Deployer.MapRoutesToApp(newAppName, newAppRoutes...)
			p.Deployer.RenameApp(newAppName, appName)
		}
		return true
	} else {
		// We don't want to promote. Instead mark it as failed.
		p.Deployer.RenameApp(newAppName, appName+"-failed")
		return false
	}
}

func (p *CfPlugin) GetNewAppRoutes(appName string, defaultCfDomain string, manifestReader manifest.ManifestReader, liveAppRoutes []plugin_models.GetApp_RouteSummary) []plugin_models.GetApp_RouteSummary {
	newAppRoutes := []plugin_models.GetApp_RouteSummary{}

	manifest, err := manifestReader.Read()
	if err != nil {
		// This error should be handled properly
		fmt.Println(err)
	}

	if manifest != nil {
		if appParams := manifest.GetAppParams(appName, defaultCfDomain); appParams != nil && appParams.Routes != nil {
			newAppRoutes = appParams.Routes
		}
	}

	uniqueRoutes := p.UnionRouteLists(newAppRoutes, liveAppRoutes)

	if len(uniqueRoutes) == 0 {
		uniqueRoutes = append(uniqueRoutes, plugin_models.GetApp_RouteSummary{Host: appName, Domain: plugin_models.GetApp_DomainFields{Name: defaultCfDomain}})
	}
	return uniqueRoutes
}

func (p *CfPlugin) GetScaleFromManifest(appName string, defaultCfDomain string,
	manifestReader manifest.ManifestReader) (scaleParameters ScaleParameters) {
	manifest, err := manifestReader.Read()
	if err != nil {
		// TODO: Handle this error nicely
		fmt.Println(err)
	}
	if manifest != nil {
		manifestScaleParameters := manifest.GetAppParams(appName, defaultCfDomain)
		if manifestScaleParameters != nil {
			scaleParameters = ScaleParameters{
				Memory:        manifestScaleParameters.Memory,
				InstanceCount: manifestScaleParameters.InstanceCount,
				DiskQuota:     manifestScaleParameters.DiskQuota,
			}
		}
	}
	return
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
					// TODO for manifests with multiple apps, a different smoke test is needed. The approach below would not work.
					// Perhaps we could name the smoke test in the manifest?
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

	log.SetFlags(0)

	p := CfPlugin{
		Deployer: &BlueGreenDeploy{
			ErrorFunc: func(message string, err error) {
				log.Fatalf("%v - %v", message, err)
			},
			Out: os.Stdout,
		},
	}

	// TODO issue #24 - (Rufus) - not sure if I'm using the plugin correctly, but if I build (go build) and run without arguments
	// I expected to see available arguments but instead the code panics.
	plugin.Start(&p)
}
