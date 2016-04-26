package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/cloudfoundry/cli/cf/configuration/confighelpers"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	"github.com/cloudfoundry/cli/plugin"
)

type ErrorHandler func(string, error)

type BlueGreenDeployer interface {
	Setup(plugin.CliConnection)
	PushNewApp(string, Route)
	DeleteAllAppsExceptLiveApp(string)
	LiveApp(string) (string, []Route)
	RunSmokeTests(string, string) bool
	UnmapRoutesFromApp(string, ...Route)
	RenameApp(string, string)
	MapRoutesToApp(string, ...Route)
}

type BlueGreenDeploy struct {
	Connection plugin.CliConnection
	Out        io.Writer
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

func (p *BlueGreenDeploy) PushNewApp(appName string, route Route) {
	if _, err := p.Connection.CliCommand("push", appName, "-n", route.Host, "-d", route.Domain.Name); err != nil {
		p.ErrorFunc("Could not push new version", err)
	}
}

func (p *BlueGreenDeploy) FilterApps(appName string, apps []Application) (currentApp *Application, oldApps []Application) {
	r := regexp.MustCompile(fmt.Sprintf("^%s(-old|-failed|-new)?$", appName))
	for index, app := range apps {
		if !r.MatchString(app.Name) {
			continue
		}

		if strings.HasSuffix(app.Name, "-old") || strings.HasSuffix(app.Name, "-failed") || strings.HasSuffix(app.Name, "-new") {
			oldApps = append(oldApps, app)
		} else {
			currentApp = &apps[index]
		}
	}
	return
}

func (p *BlueGreenDeploy) LiveApp(appName string) (string, []Route) {
	appsInSpace, err := p.AppLister.AppsInCurrentSpace()
	if err != nil {
		p.ErrorFunc("Could not load apps in space, are you logged in?", err)
	}

	liveApp, _ := p.FilterApps(appName, appsInSpace)

	if liveApp == nil {
		return "", nil
	} else {
		return liveApp.Name, liveApp.Routes
	}
}

func (p *BlueGreenDeploy) Setup(connection plugin.CliConnection) {
	p.Connection = connection
	p.AppLister = &CfCurlAppLister{Connection: connection}
}

func (p *BlueGreenDeploy) RunSmokeTests(script, appFQDN string) bool {
	out, err := exec.Command(script, appFQDN).CombinedOutput()
	fmt.Fprintln(p.Out, string(out))

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false
		} else {
			p.ErrorFunc("Smoke tests failed", err)
		}
	}
	return true
}

func (p *BlueGreenDeploy) UnmapRoutesFromApp(oldAppName string, routes ...Route) {
	for _, route := range routes {
		p.unmapRoute(oldAppName, route)
	}
}

func (p *BlueGreenDeploy) mapRoute(appName string, r Route) {
	if _, err := p.Connection.CliCommand("map-route", appName, r.Domain.Name, "-n", r.Host); err != nil {
		p.ErrorFunc("Could not map route", err)
	}
}

func (p *BlueGreenDeploy) unmapRoute(appName string, r Route) {
	if _, err := p.Connection.CliCommand("unmap-route", appName, r.Domain.Name, "-n", r.Host); err != nil {
		p.ErrorFunc("Could not unmap route", err)
	}
}

func (p *BlueGreenDeploy) RenameApp(app string, newName string) {
	if _, err := p.Connection.CliCommand("rename", app, newName); err != nil {
		p.ErrorFunc("Could not rename app", err)
	}
}

func (p *BlueGreenDeploy) MapRoutesToApp(appName string, routes ...Route) {
	for _, route := range routes {
		p.mapRoute(appName, route)
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
	configRepo := coreconfig.NewRepositoryFromFilepath(confighelpers.DefaultFilePath(), func(err error) {
		if err != nil {
			fmt.Printf("Config error: %s", err)
		}
	})

	return configRepo.SpaceFields().Guid
}
