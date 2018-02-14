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

	cfDomains := manifest.CfDomains{}
	var err error

	cfDomains.SharedDomains, err = p.SharedDomains()
	if err != nil {
		log.Fatalf("Failed to get shared domains: %v", err)
	}

	if len(cfDomains.SharedDomains) < 1 {
		log.Fatalf("Failed to get default shared domain (no shared domains defined)")
	} else {
		cfDomains.DefaultDomain = cfDomains.SharedDomains[0]
	}

	cfDomains.PrivateDomains, err = p.PrivateDomains()
	if err != nil {
		log.Fatalf("Failed to get private domains: %v", err)
	}

	p.Deployer.Setup(cliConnection)

	if argsStruct.AppName == "" {
		log.Fatal("App name was empty, must be provided.")
	}

	reader := manifest.FileManifestReader{argsStruct.ManifestPath}
	if !p.Deploy(cfDomains, &reader, argsStruct) {
		log.Fatal("Smoke tests failed")
	}
}

func (p *CfPlugin) Deploy(cfDomains manifest.CfDomains, manifestReader manifest.ManifestReader, args Args) bool {
	appName := args.AppName

	p.Deployer.DeleteAllAppsExceptLiveApp(appName)
	liveAppName, liveAppRoutes := p.Deployer.LiveApp(appName)

	manifestScaleParameters := p.GetScaleFromManifest(appName, cfDomains, manifestReader)

	newAppName := appName + "-new"

	// Add route so that we can run the smoke tests
	tempRoute := plugin_models.GetApp_RouteSummary{Host: newAppName, Domain: plugin_models.GetApp_DomainFields{Name: cfDomains.DefaultDomain}}

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
	newAppRoutes := p.GetNewAppRoutes(args.AppName, cfDomains, manifestReader, liveAppRoutes)

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

func (p *CfPlugin) GetNewAppRoutes(appName string, cfDomains manifest.CfDomains, manifestReader manifest.ManifestReader, liveAppRoutes []plugin_models.GetApp_RouteSummary) []plugin_models.GetApp_RouteSummary {
	newAppRoutes := []plugin_models.GetApp_RouteSummary{}

	parsedManifest, err := manifestReader.Read()
	if err != nil {
		// This error should be handled properly
		fmt.Println(err)
	}

	if parsedManifest != nil {
		if appParams := parsedManifest.GetAppParams(appName, cfDomains); appParams != nil && appParams.Routes != nil {
			newAppRoutes = appParams.Routes
		}
	}

	uniqueRoutes := p.UnionRouteLists(newAppRoutes, liveAppRoutes)

	if len(uniqueRoutes) == 0 {
		uniqueRoutes = append(uniqueRoutes, plugin_models.GetApp_RouteSummary{Host: appName, Domain: plugin_models.GetApp_DomainFields{Name: cfDomains.DefaultDomain}})
	}
	return uniqueRoutes
}

func (p *CfPlugin) GetScaleFromManifest(appName string, cfDomains manifest.CfDomains,
	manifestReader manifest.ManifestReader) (scaleParameters ScaleParameters) {
	parsedManifest, err := manifestReader.Read()
	if err != nil {
		// TODO: Handle this error nicely
		fmt.Println(err)
	}
	if parsedManifest != nil {
		manifestScaleParameters := parsedManifest.GetAppParams(appName, cfDomains)
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

func (p *CfPlugin) PrivateDomains() (domains []string, apiErr error) {
	path := "/v2/private_domains"
	return p.listCfDomains(path)
}

func (p *CfPlugin) SharedDomains() (domains []string, apiErr error) {
	path := "/v2/shared_domains"
	return p.listCfDomains(path)
}

func (p *CfPlugin) listCfDomains(cfPath string) (domains []string, err error) {
	var res []string
	if res, err = p.Connection.CliCommandWithoutTerminalOutput("curl", cfPath); err != nil {
		return
	}

	response := struct {
		Resources []struct {
			Entity struct {
				Name string
			}
		}
	}{}

	var jsonString string
	jsonString = strings.Join(res, "\n")

	if err = json.Unmarshal([]byte(jsonString), &response); err != nil {
		return
	}

	for i, _ := range response.Resources {
		domains = append(domains, response.Resources[i].Entity.Name)
	}
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
