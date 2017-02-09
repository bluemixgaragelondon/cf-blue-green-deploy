package main

import (
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
)

type ErrorHandler func(string, error)

type BlueGreenDeployer interface {
	Setup(plugin.CliConnection)
	PushNewApp(string, plugin_models.GetApp_RouteSummary, string)
	DeleteAllAppsExceptLiveApp(string)
	LiveApp(string) (string, []plugin_models.GetApp_RouteSummary)
	RunSmokeTests(string, string) bool
	UnmapRoutesFromApp(string, ...plugin_models.GetApp_RouteSummary)
	RenameApp(string, string)
	MapRoutesToApp(string, ...plugin_models.GetApp_RouteSummary)
}

type BlueGreenDeploy struct {
	Connection plugin.CliConnection
	Out        io.Writer
	ErrorFunc  ErrorHandler
}

func (p *BlueGreenDeploy) DeleteAppVersions(apps []plugin_models.GetAppsModel) {
	for _, app := range apps {
		if _, err := p.Connection.CliCommand("delete", app.Name, "-f", "-r"); err != nil {
			p.ErrorFunc("Could not delete old app version", err)
		}
	}
}

func (p *BlueGreenDeploy) DeleteAllAppsExceptLiveApp(appName string) {
	appsInSpace, err := p.Connection.GetApps()
	if err != nil {
		p.ErrorFunc("Could not load apps in space, are you logged in?", err)
	}
	oldAppVersions := p.GetOldApps(appName, appsInSpace)
	p.DeleteAppVersions(oldAppVersions)

}

func (p *BlueGreenDeploy) PushNewApp(appName string, route plugin_models.GetApp_RouteSummary, manifestPath string) {
	args := []string{"push", appName, "-n", route.Host, "-d", route.Domain.Name}
	if manifestPath != "" {
		args = append(args, "-f", manifestPath)
	}
	if _, err := p.Connection.CliCommand(args...); err != nil {
		p.ErrorFunc("Could not push new version", err)
	}
}

func (p *BlueGreenDeploy) GetOldApps(appName string, apps []plugin_models.GetAppsModel) (oldApps []plugin_models.GetAppsModel) {
	r := regexp.MustCompile(fmt.Sprintf("^%s(-old|-failed|-new)?$", appName))
	for _, app := range apps {
		if !r.MatchString(app.Name) {
			continue
		}

		// TODO (Rufus) - perhaps a change in the regex is needed.
		// - e.g. `^%s-(old|failed|new)$` (making the capture group not optional). This would mean that the live app, if that is the version
		// with no prefix, is not matched but others are. Equally, if the live app is the one without a suffix, perhaps it would be sufficient
		// to check for the existence of a hyphen, in which case we could use something like strings.Count for hyphen instead of the regex.
		// Then we would not need the if statement below.
		if strings.HasSuffix(app.Name, "-old") || strings.HasSuffix(app.Name, "-failed") || strings.HasSuffix(app.Name, "-new") {
			oldApps = append(oldApps, app)
		}
	}
	return
}

func (p *BlueGreenDeploy) LiveApp(appName string) (string, []plugin_models.GetApp_RouteSummary) {

	// Don't worry about error handling since earlier calls would have flushed out any errors
	// except for ones that the app doesn't exist (which isn't an error condition for us)
	liveApp, _ := p.Connection.GetApp(appName)
	return liveApp.Name, liveApp.Routes
}

func (p *BlueGreenDeploy) Setup(connection plugin.CliConnection) {
	p.Connection = connection
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

func (p *BlueGreenDeploy) UnmapRoutesFromApp(oldAppName string, routes ...plugin_models.GetApp_RouteSummary) {
	for _, route := range routes {
		p.unmapRoute(oldAppName, route)
	}
}

func (p *BlueGreenDeploy) mapRoute(appName string, r plugin_models.GetApp_RouteSummary) {
	if _, err := p.Connection.CliCommand("map-route", appName, r.Domain.Name, "-n", r.Host); err != nil {
		p.ErrorFunc("Could not map route", err)
	}
}

func (p *BlueGreenDeploy) unmapRoute(appName string, r plugin_models.GetApp_RouteSummary) {
	if _, err := p.Connection.CliCommand("unmap-route", appName, r.Domain.Name, "-n", r.Host); err != nil {
		p.ErrorFunc("Could not unmap route", err)
	}
}

func (p *BlueGreenDeploy) RenameApp(app string, newName string) {
	if _, err := p.Connection.CliCommand("rename", app, newName); err != nil {
		p.ErrorFunc("Could not rename app", err)
	}
}

func (p *BlueGreenDeploy) MapRoutesToApp(appName string, routes ...plugin_models.GetApp_RouteSummary) {
	for _, route := range routes {
		p.mapRoute(appName, route)
	}
}
