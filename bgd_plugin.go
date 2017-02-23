package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
)

// PluginVersion is a global
var PluginVersion string

type CfPlugin struct {
	Connection plugin.CliConnection
	Deployer   BlueGreenDeployer
	DeploymentInfo
}

type DeploymentInfo struct {
	DefaultCfDomain string
	LiveApp         *App
	StagingApp      *App
	NewApp          *App
}

// Embed the GetAppModel struct so we can define
// our own functions on this struct as well as
// running custom ones.
type App struct {
	plugin_models.GetAppModel
}

// GetMetadata must be defined so that CfPlugin satisfies the cf cli plugin interface
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

// Run is called after some processing done by the plugin library during plugin.Start
// Run must be defined so that CfPlugin satisfies the cf cli plugin interface
func (p *CfPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) > 0 && args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	argsStruct, err := NewArgs(args)
	if err != nil {
		log.Fatal(err)
	}

	defaultCfDomain, err := p.DefaultCfDomain()
	if err != nil {
		log.Fatalf("Failed to get default shared domain: %v", err)
	}

	p.Deployer.Setup(cliConnection)

	if argsStruct.AppName == "" {
		log.Fatal("App name was empty, must be provided.")
	}

	if err := p.Deploy(defaultCfDomain, &manifest.FileManifestReader{}, *argsStruct); err != nil {
		log.Fatal("Deploy failed: %v", err)
	}
}

// DefaultCfDomain gets the default CF domain.
// While https://docs.cloudfoundry.org/devguide/deploy-apps/routes-domains.html#shared-domains
// shows that there is technically not a default shared domain,
// by default pushes go to the first shared domain created in the system.
// As long as the first created is the same as the first listed in our query
// below, our function is valid.
func (p *CfPlugin) DefaultCfDomain() (string, error) {
	var res []string
	var err error

	if res, err = p.Connection.CliCommandWithoutTerminalOutput("curl", "/v2/shared_domains"); err != nil {
		return "", err
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
		return "", err
	}
	return response.Resources[0].Entity.Name, nil
}

func (p *CfPlugin) Deploy(defaultCfDomain string, manifestReader manifest.ManifestReader, args Args) error {
	appName := args.AppName

	p.Deployer.DeleteAllAppsExceptLiveApp(appName)
	p.LiveApp = p.Deployer.LiveApp(appName)

	manifestScaleParameters := p.GetScaleFromManifest(appName, defaultCfDomain, manifestReader)

	newAppName := appName + "-new"

	// Add route so that we can run the smoke tests
	tempRoute := plugin_models.GetApp_RouteSummary{Host: newAppName, Domain: plugin_models.GetApp_DomainFields{Name: defaultCfDomain}}

	// If deploy is unsuccessful, p.ErrorFunc will be called which exits.
	p.Deployer.PushNewApp(newAppName, tempRoute, args.ManifestPath, manifestScaleParameters)

	// If we have a smoke test, run it
	if args.SmokeTestPath != "" {
		if err := p.Deployer.RunSmokeTests(args.SmokeTestPath, FQDN(tempRoute)); err != nil {
			// If smoke test errors, return error
			p.Deployer.UnmapRoutesFromApp(newAppName, tempRoute)
			p.Deployer.RenameApp(newAppName, appName+"-failed")
			return err
		}
	}

	// TODO We're overloading 'new' here for both the staging app and the 'finished' app, which is confusing
	newAppRoutes := p.GetNewAppRoutes(appName, defaultCfDomain, manifestReader, p.LiveApp)

	p.Deployer.UnmapRoutesFromApp(newAppName, tempRoute)

	// If there is a live app, we want to disassociate the routes with the old version of the app
	// and instead update the routes to use the new version.
	if p.LiveApp != nil {
		p.Deployer.MapRoutesToApp(newAppName, newAppRoutes...)
		p.Deployer.RenameApp(p.LiveApp.Name, appName+"-old")
		p.Deployer.RenameApp(newAppName, appName)
		p.Deployer.UnmapRoutesFromApp(appName+"-old", p.LiveApp.Routes...)
	} else {
		// If there is no live app, we only need to add our new routes.
		p.Deployer.MapRoutesToApp(newAppName, newAppRoutes...)
		p.Deployer.RenameApp(newAppName, appName)
	}

	return nil

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

func FQDN(r plugin_models.GetApp_RouteSummary) string {
	return fmt.Sprintf("%v.%v", r.Host, r.Domain.Name)
}

func (p *CfPlugin) GetNewAppRoutes(appName string, defaultCfDomain string, manifestReader manifest.ManifestReader, liveApp *App) []plugin_models.GetApp_RouteSummary {
	var uniqueRoutes []plugin_models.GetApp_RouteSummary
	var routesFromManifest []plugin_models.GetApp_RouteSummary

	manifest, err := manifestReader.Read()
	if err != nil {
		// This error should be handled properly
		fmt.Println(err)
	}

	if manifest != nil {
		if appParams := manifest.GetAppParams(appName, defaultCfDomain); appParams != nil && appParams.Routes != nil {
			routesFromManifest = appParams.Routes
		}
	}

	if liveApp != nil && len(liveApp.Routes) != 0 {
		uniqueRoutes = p.UnionRouteLists(routesFromManifest, liveApp.Routes)
	} else {
		uniqueRoutes = routesFromManifest
	}

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
