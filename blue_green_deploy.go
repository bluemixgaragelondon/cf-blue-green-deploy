package main

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/cloudfoundry/cli/cf/configuration/config_helpers"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/plugin"
)

type ErrorHandler func(string, error)

type BlueGreen interface {
	Run() error
	PushNewAppVersion(string) string
	DeleteAppVersions([]Application)
	RunSmokeTests(string, string)
	RemapRoutesFromLiveAppToNewApp(Application, Application)
}

type BlueGreenDeploy struct {
	Connection plugin.CliConnection
	ErrorFunc  ErrorHandler
	AppLister
}

type AppLister interface {
	AppsInCurrentSpace() ([]Application, error)
}

type CfCurlAppLister struct {
	Connection plugin.CliConnection
}

func (p *BlueGreenDeploy) DeleteAppVersions(apps []Application) {
	for _, app := range apps {
		if _, err := p.Connection.CliCommand("delete", app.Name, "-f", "-r"); err != nil {
			p.ErrorFunc("Could not delete old app version", err)
		}
	}
}

func (p *BlueGreenDeploy) PushNewAppVersion(appName string) (newApp Application) {
	newApp.Name = GenerateAppName(appName)
	if _, err := p.Connection.CliCommand("push", newApp.Name); err != nil {
		p.ErrorFunc("Could not push new version", err)
	}

	apps, _ := p.AppsInCurrentSpace()
	for i, app := range apps {
		if app.Name == newApp.Name {
			newApp = apps[i]
			break
		}
	}

	return
}

func (p *BlueGreenDeploy) RunSmokeTests(script, appFQDN string) {
	out, err := exec.Command(script, appFQDN).CombinedOutput()
	fmt.Println(string(out))

	if err != nil {
		p.ErrorFunc("Smoke tests failed", err)
	}
}

func (p *BlueGreenDeploy) RemapRoutesFromLiveAppToNewApp(liveApp, newApp Application) {
	defaultRoute := liveApp.DefaultRoute()

	for _, route := range liveApp.Routes {
		if route != defaultRoute {
			p.mapRoute(newApp, route)
		}
		p.unmapRoute(liveApp, route)
	}
}

func (p *BlueGreenDeploy) mapRoute(a Application, r Route) {
	if _, err := p.Connection.CliCommand("map-route", a.Name, r.Domain.Name, "-n", r.Host); err != nil {
		p.ErrorFunc("Could not map route", err)
	}
}

func (p *BlueGreenDeploy) unmapRoute(a Application, r Route) {
	if _, err := p.Connection.CliCommand("unmap-route", a.Name, r.Domain.Name, "-n", r.Host); err != nil {
		p.ErrorFunc("Could not unmap route", err)
	}
}

func (l *CfCurlAppLister) AppsInCurrentSpace() ([]Application, error) {
	path := fmt.Sprintf("/v2/spaces/%s/summary", getSpaceGuid())

	output, err := l.Connection.CliCommandWithoutTerminalOutput("curl", path)
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
