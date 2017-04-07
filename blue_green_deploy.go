package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/imdario/mergo"
)

type ErrorHandler func(string, error)

type BlueGreenDeployer interface {
	Setup(plugin.CliConnection)
	Push(*App)
	DeleteAllAppsExceptLiveApp(string)
	LiveApp(string) *App
	RunSmokeTests(string, string) error
	UnmapRoutesFromApp(string, ...plugin_models.GetApp_RouteSummary)
	RenameApp(string, string)
	MapRoutesToApp(string, ...plugin_models.GetApp_RouteSummary)
	DefaultCfDomain() (string, error)
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

func (p *BlueGreenDeploy) Setup(connection plugin.CliConnection) {
	p.Connection = connection
}

// DefaultCfDomain gets the default CF domain.
// While https://docs.cloudfoundry.org/devguide/deploy-apps/routes-domains.html#shared-domains
// shows that there is technically not a default shared domain,
// by default pushes go to the first shared domain created in the system.
// As long as the first created is the same as the first listed in our query
// below, our function is valid.
func (p *BlueGreenDeploy) DefaultCfDomain() (string, error) {
	var res []string
	var err error

	if res, err = p.Connection.CliCommandWithoutTerminalOutput("curl", "/v2/shared_domains"); err != nil {
		return "", err
	}

	response := struct {
		Description string `json:"description"`
		ErrorCode   string `json:"error_code"`
		Resources   []struct {
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

	if response.ErrorCode != "" {
		return "", fmt.Errorf("%s: %s", response.Description, response.ErrorCode)
	}

	if len(response.Resources) == 0 {
		return "", fmt.Errorf("No CF Domains found")
	}
	return response.Resources[0].Entity.Name, nil
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

// TODO generate this based on struct tags to avoid massive list of if statements
func (a *App) generatePushArgs() (string, error) {
	result := "push"
	if a.Name == "" {
		return "", fmt.Errorf("Expected app to have name, cannot push without name")
	}
	result += " " + a.Name

	if len(a.Routes) != 1 {
		// TODO support pushing apps with more than 1 route
		return "", fmt.Errorf("Expected new app to have exactly 1 route during push, got %v", len(a.Routes))
	}
	if a.Routes[0].Host == "" {
		return "", fmt.Errorf("Expected new app to have a host")
	}
	result += " -n " + a.Routes[0].Host

	if a.Routes[0].Domain.Name == "" {
		return "", fmt.Errorf("Expected new app to have a domain name")
	}
	result += " -d " + a.Routes[0].Domain.Name

	if a.InstanceCount != 0 {
		result += fmt.Sprintf(" -i %d", a.InstanceCount)
	}
	if a.Memory != 0 {
		result += fmt.Sprintf(" -m %dM", a.Memory)
	}
	if a.DiskQuota != 0 {
		result += fmt.Sprintf(" -k %dM", a.DiskQuota)
	}
	if a.ManifestPath != "" {
		result += " -f " + a.ManifestPath
	}
	return result, nil
}

// Merge uses a third party library, mergo, to merge app definitions.
// If a parameter is not equal to it's zero value in a, this is left.
// If a parameter is equal to it's zero value in a but defined in liveApp,
// the parameter is copied over to a.
func (a *App) Merge(liveApp *App) error {
	// Use serverside scale parameters if not defined in manifest
	if liveApp != nil {
		if err := mergo.Merge(a, *liveApp); err != nil {
			return err
		}
	}
	return nil
}

func (p *BlueGreenDeploy) Push(newApp *App) {
	if len(newApp.Routes) != 1 {
		// TODO support pushing apps with more than 1 route
		err := fmt.Errorf("Expected to be pushing an app with 1 route, got %v", len(newApp.Routes))
		p.ErrorFunc("", err)
	}

	args, err := newApp.generatePushArgs()
	if err != nil {
		p.ErrorFunc("", err)
	}

	if _, err := p.Connection.CliCommand(strings.Split(args, " ")...); err != nil {
		p.ErrorFunc("Could not run "+args, err)
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

func (p *BlueGreenDeploy) LiveApp(appName string) *App {

	// Don't worry about error handling since earlier calls would have flushed out any errors
	// except for ones that the app doesn't exist (which isn't an error condition for us)

	// TODO: We should capture the specific error for app not existing and handle all other errors

	liveApp, _ := p.Connection.GetApp(appName)
	return &App{GetAppModel: liveApp}
}

func (p *BlueGreenDeploy) RunSmokeTests(script, appFQDN string) error {
	out, err := exec.Command(script, appFQDN).CombinedOutput()
	fmt.Fprintln(p.Out, string(out))

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return err
		} else {
			p.ErrorFunc("Smoke tests failed", err)
		}
	}
	return nil
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
		p.ErrorFunc(fmt.Sprintf("Could not rename app %v to %v", app, newName), err)
	}
}

func (p *BlueGreenDeploy) MapRoutesToApp(appName string, routes ...plugin_models.GetApp_RouteSummary) {
	for _, route := range routes {
		p.mapRoute(appName, route)
	}
}
