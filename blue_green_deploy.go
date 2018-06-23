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
	PushNewApp(string, plugin_models.GetApp_RouteSummary, string, ScaleParameters)
	DeleteAllAppsExceptLiveApp(string)
	GetScaleParameters(string) (ScaleParameters, error)
	LiveApp(string) (string, []plugin_models.GetApp_RouteSummary)
	RunSmokeTests(string, string) bool
	UnmapRoutesFromApp(string, ...plugin_models.GetApp_RouteSummary)
	DeleteRoutes(...plugin_models.GetApp_RouteSummary)
	RenameApp(string, string)
	MapRoutesToApp(string, ...plugin_models.GetApp_RouteSummary)
	CheckSshEnablement(string) bool
	SetSshAccess(string, bool)
}

type BlueGreenDeploy struct {
	Connection plugin.CliConnection
	Out        io.Writer
	ErrorFunc  ErrorHandler
}

type ScaleParameters struct {
	InstanceCount int
	Memory        int64
	DiskQuota     int64
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

func (p *BlueGreenDeploy) GetScaleParameters(appName string) (ScaleParameters, error) {
	appModel, err := p.Connection.GetApp(appName)
	if err != nil {
		return ScaleParameters{}, fmt.Errorf("Could not get scale parameters")
	}
	scaleParameters := ScaleParameters{
		InstanceCount: appModel.InstanceCount,
		Memory:        appModel.Memory,
		DiskQuota:     appModel.DiskQuota,
	}
	return scaleParameters, nil
}

func mergeScaleParameters(liveScale, manifestScale ScaleParameters) ScaleParameters {
	scaleParameters := liveScale
	if manifestScale.Memory != 0 {
		scaleParameters.Memory = manifestScale.Memory
	}
	if manifestScale.InstanceCount != 0 {
		scaleParameters.InstanceCount = manifestScale.InstanceCount
	}
	if manifestScale.DiskQuota != 0 {
		scaleParameters.DiskQuota = manifestScale.DiskQuota
	}
	return scaleParameters
}

func appendScaleArguments(args []string, scaleParameters ScaleParameters) []string {
	if scaleParameters.InstanceCount != 0 {
		instanceCount := fmt.Sprintf("%d", scaleParameters.InstanceCount)
		args = append(args, "-i", instanceCount)
	}
	if scaleParameters.Memory != 0 {
		memory := fmt.Sprintf("%dM", scaleParameters.Memory)
		args = append(args, "-m", memory)
	}
	if scaleParameters.DiskQuota != 0 {
		diskQuota := fmt.Sprintf("%dM", scaleParameters.DiskQuota)
		args = append(args, "-k", diskQuota)
	}
	return args
}

func (p *BlueGreenDeploy) PushNewApp(appName string, route plugin_models.GetApp_RouteSummary,
	manifestPath string, scaleParameters ScaleParameters) {
	args := []string{"push", appName, "-n", route.Host, "-d", route.Domain.Name}

	// Remove -new suffix of appname to get live app name
	newAppSuffix := "-new"
	liveAppName := appName[:len(appName)-len(newAppSuffix)]
	liveScaleParameters, _ := p.GetScaleParameters(liveAppName)
	scaleParameters = mergeScaleParameters(liveScaleParameters, scaleParameters)

	args = appendScaleArguments(args, scaleParameters)
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
	cmd := exec.Command(script, appFQDN)

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	stdout := io.MultiWriter(p.Out)
	stderr := io.MultiWriter(p.Out)

	err := cmd.Start()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false
		} else {
			p.ErrorFunc("Smoke tests failed", err)
		}
	}

	var errStdout, errStderr error

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	err = cmd.Wait()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false
		} else {
			p.ErrorFunc("Smoke tests failed", err)
		}
	}

	if errStdout != nil || errStderr != nil {
		if errStdout != nil {
			err = errStdout
		} else {
			err = errStderr
		}
		p.ErrorFunc("Failed to capture smoke test output", err)
	}

	return true
}

func (p *BlueGreenDeploy) UnmapRoutesFromApp(oldAppName string, routes ...plugin_models.GetApp_RouteSummary) {
	for _, route := range routes {
		p.unmapRoute(oldAppName, route)
	}
}

func (p *BlueGreenDeploy) DeleteRoutes(routes ...plugin_models.GetApp_RouteSummary) {
	for _, route := range routes {
		p.deleteRoute(route)
	}
}

func (p *BlueGreenDeploy) mapRoute(appName string, r plugin_models.GetApp_RouteSummary) {
	if _, err := p.Connection.CliCommand("map-route", appName, r.Domain.Name, "-n", r.Host); err != nil {
		p.ErrorFunc("Could not map route", err)
	}
}

func (p *BlueGreenDeploy) unmapRoute(appName string, r plugin_models.GetApp_RouteSummary) {
	command := []string{"unmap-route", appName, r.Domain.Name, "-n", r.Host}
	if len(r.Path) != 0 {
		command = append(command, "--path")
		command = append(command, r.Path)
	}
	if _, err := p.Connection.CliCommand(command...); err != nil {
		p.ErrorFunc("Could not unmap route", err)
	}
}

func (p *BlueGreenDeploy) deleteRoute(r plugin_models.GetApp_RouteSummary) {
	if _, err := p.Connection.CliCommand("delete-route", r.Domain.Name, "-n", r.Host, "-f"); err != nil {
		p.ErrorFunc("Could not delete route", err)
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

func (p *BlueGreenDeploy) CheckSshEnablement(app string) bool {
	if result, err := p.Connection.CliCommand("ssh-enabled", app); err != nil {
		p.ErrorFunc("Check ssh enabled status failed", err)
		return true
	} else {
		return (strings.Contains(result[0], "support is enabled"))
	}
}

func (p *BlueGreenDeploy) SetSshAccess(app string, enableSsh bool) {
	if enableSsh {
		if _, err := p.Connection.CliCommand("enable-ssh", app); err != nil {
			p.ErrorFunc("Could not enable ssh", err)
		}
	} else {
		if _, err := p.Connection.CliCommand("disable-ssh", app); err != nil {
			p.ErrorFunc("Could not disable ssh", err)
		}
	}
}
