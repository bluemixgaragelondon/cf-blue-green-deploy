package main

import (
	"fmt"
	"os/exec"

	"github.com/cloudfoundry/cli/plugin"
)

type ErrorHandler func(string, error)

type BlueGreenDeploy struct {
	Connection plugin.CliConnection
	ErrorFunc  ErrorHandler
}

func (p *BlueGreenDeploy) DeleteAppVersions(apps []Application) {
	for _, app := range apps {
		if _, err := p.Connection.CliCommand("delete", app.Name, "-f", "-r"); err != nil {
			p.ErrorFunc("Could not delete old app version", err)
		}
	}
}

func (p *BlueGreenDeploy) PushNewAppVersion(appName string) (newAppName string) {
	newAppName = GenerateAppName(appName)
	if _, err := p.Connection.CliCommand("push", newAppName); err != nil {
		p.ErrorFunc("Could not push new version", err)
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
