package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/cloudfoundry/cli/cf/configuration/config_helpers"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/plugin"
)

type ErrorHandler func(string, error)

type BlueGreenDeployer interface {
	Setup(plugin.CliConnection)
	PushNewApp(string) Application
	DeleteAllAppsExceptLiveApp(string)
	LiveApp(string) *Application
	RunSmokeTests(string, string)
	RemapRoutesFromLiveAppToNewApp(Application, Application)
	MarkAppAsOld(*Application)
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

func (p *BlueGreenDeploy) DeleteAllAppsExceptLiveApp(appName string) {
	appsInSpace, err := p.AppLister.AppsInCurrentSpace()
	if err != nil {
		p.ErrorFunc("Could not load apps in space, are you logged in?", err)
	}
	_, oldAppVersions := p.FilterApps(appName, appsInSpace)
	p.DeleteAppVersions(oldAppVersions)
}

func (p *BlueGreenDeploy) PushNewApp(appName string) (newApp Application) {
	newApp.Name = GenerateAppName(appName)
	if _, err := p.Connection.CliCommand("push", newApp.Name, "-n", newApp.Name); err != nil {
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

func (p *BlueGreenDeploy) FilterApps(appName string, apps []Application) (currentApp *Application, oldApps []Application) {
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

func (p *BlueGreenDeploy) LiveApp(appName string) (liveApp *Application) {
	appsInSpace, err := p.AppLister.AppsInCurrentSpace()
	if err != nil {
		p.ErrorFunc("Could not load apps in space, are you logged in?", err)
	}

	liveApp, _ = p.FilterApps(appName, appsInSpace)

	return
}

func (p *BlueGreenDeploy) Setup(connection plugin.CliConnection) {
	p.Connection = connection
	p.AppLister = &CfCurlAppLister{Connection: connection}
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
		if route == defaultRoute {
			continue
		}

		p.mapRoute(newApp, route)
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

func (p *BlueGreenDeploy) MarkAppAsOld(app *Application) {
	if _, err := p.Connection.CliCommand("rename", app.Name, fmt.Sprintf("%s-old", app.Name)); err != nil {
		p.ErrorFunc("Could not rename app", err)
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
