// NOTICE: This is a derivative work of https://github.com/bluemixgaragelondon/cf-blue-green-deploy/blob/master/manifest.go.
package manifest

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cli/cf/formatters"
	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/Pallinder/go-randomdata"
)

type Manifest struct {
	Path string
	Data map[string]interface{}
}

type CfDomains struct {
	DefaultDomain  string
	SharedDomains  []string
	PrivateDomains []string
}

func (m Manifest) Applications(cfDomains CfDomains) ([]plugin_models.GetAppModel, error) {

	rawData, err := expandProperties(m.Data)
	data := rawData.(map[string]interface{})

	if err != nil {
		return []plugin_models.GetAppModel{}, err
	}

	appMaps, err := m.getAppMaps(data)
	if err != nil {
		return []plugin_models.GetAppModel{}, err
	}
	var apps []plugin_models.GetAppModel
	var mapToAppErrs []error
	for _, appMap := range appMaps {
		app, err := mapToAppParams(filepath.Dir(m.Path), appMap, cfDomains)
		if err != nil {
			mapToAppErrs = append(mapToAppErrs, err)
			continue
		}
		apps = append(apps, app)
	}

	if len(mapToAppErrs) > 0 {
		message := ""
		for i := range mapToAppErrs {
			message = message + fmt.Sprintf("%s\n", mapToAppErrs[i].Error())
		}
		return []plugin_models.GetAppModel{}, errors.New(message)
	}

	return apps, nil
}

func cloneWithExclude(data map[string]interface{}, excludedKey string) map[string]interface{} {
	otherMap := make(map[string]interface{})
	for key, value := range data {
		if excludedKey != key {
			otherMap[key] = value
		}
	}
	return otherMap
}

func (m Manifest) getAppMaps(data map[string]interface{}) ([]map[string]interface{}, error) {

	var apps []map[string]interface{}
	var errs []error
	// Check for presence
	unTypedAppMaps, ok := data["applications"]
	if ok {
		// Check for type
		appMaps, ok := unTypedAppMaps.([]interface{})

		if !ok {
			return []map[string]interface{}{}, errors.New("Expected applications to be a list")
		}

		globalProperties := cloneWithExclude(data, "applications")

		for _, appData := range appMaps {
			appDataAsMap, err := Mappify(appData)
			if err != nil {
				errs = append(errs, fmt.Errorf("Expected application to be a list of key/value pairs\nError occurred in manifest near:\n'{{.YmlSnippet}}'. Error was %v",
					map[string]interface{}{"YmlSnippet": appData}, err))
				continue
			}

			appMap, err := DeepMerge(globalProperties, appDataAsMap)
			if err != nil {
				return nil, err
			}

			apps = append(apps, appMap)
		}
	} else {
		// All properties in data are global, so just throw them in
		apps = append(apps, data)
	}

	if len(errs) > 0 {
		message := ""
		for i := range errs {
			message = message + fmt.Sprintf("%s\n", errs[i].Error())
		}
		return []map[string]interface{}{}, errors.New(message)
	}

	return apps, nil
}

var propertyRegex = regexp.MustCompile(`\${[\w-]+}`)

func expandProperties(input interface{}) (interface{}, error) {
	var errs []error
	var output interface{}

	switch input := input.(type) {
	case string:
		match := propertyRegex.FindStringSubmatch(input)
		if match != nil {
			if match[0] == "${random-word}" {
				// TODO we need a test for a manifest with ${random-word}
				output = strings.Replace(input, "${random-word}", strings.ToLower(randomdata.SillyName()), -1)
			} else {
				err := fmt.Errorf("Property '{{.PropertyName}}' found in manifest. This feature is no longer supported. Please remove it and try again.",
					map[string]interface{}{"PropertyName": match[0]})
				errs = append(errs, err)
			}
		} else {
			output = input
		}
	case []interface{}:
		outputSlice := make([]interface{}, len(input))
		for index, item := range input {
			itemOutput, itemErr := expandProperties(item)
			if itemErr != nil {
				errs = append(errs, itemErr)
				break
			}
			outputSlice[index] = itemOutput
		}
		output = outputSlice
	case map[interface{}]interface{}:

		outputMap := make(map[interface{}]interface{})
		for key, value := range input {
			itemOutput, itemErr := expandProperties(value)
			if itemErr != nil {
				errs = append(errs, itemErr)
				break
			}
			outputMap[key] = itemOutput
		}
		output = outputMap
	case map[string]interface{}:

		outputMap := make(map[string]interface{})
		for key, value := range input {

			itemOutput, itemErr := expandProperties(value)
			if itemErr != nil {
				errs = append(errs, itemErr)
				break
			}
			outputMap[key] = itemOutput
		}
		output = outputMap
	default:
		output = input
	}

	if len(errs) > 0 {
		message := ""
		for _, err := range errs {
			message = message + fmt.Sprintf("%s\n", err.Error())
		}
		return nil, errors.New(message)
	}

	return output, nil
}

func mapToAppParams(basePath string, yamlMap map[string]interface{}, cfDomains CfDomains) (plugin_models.GetAppModel, error) {
	err := checkForNulls(yamlMap)
	if err != nil {
		return plugin_models.GetAppModel{}, err
	}

	var appParams plugin_models.GetAppModel
	var errs []error

	if diskQuota := bytesVal(yamlMap, "disk_quota", &errs); diskQuota != nil {
		appParams.DiskQuota = *diskQuota
	}

	domainAry := sliceOrNil(yamlMap, "domains", &errs)
	if domain := stringVal(yamlMap, "domain", &errs); domain != nil {
		if domainAry == nil {
			domainAry = []string{*domain}
		} else {
			domainAry = append(domainAry, *domain)
		}
	}
	mytempDomainsObject := removeDuplicatedValue(domainAry)

	hostsArr := sliceOrNil(yamlMap, "hosts", &errs)
	if host := stringVal(yamlMap, "host", &errs); host != nil {
		hostsArr = append(hostsArr, *host)
	}
	myTempHostsObject := removeDuplicatedValue(hostsArr)

	routeRoutes := parseRoutes(cfDomains, yamlMap, &errs)
	compositeRoutes := RoutesFromManifest(cfDomains.DefaultDomain, myTempHostsObject, mytempDomainsObject)

	if routeRoutes == nil {
		appParams.Routes = compositeRoutes
	} else if compositeRoutes == nil || len(compositeRoutes) == 0 {
		appParams.Routes = routeRoutes
	} else {
		errs = append(errs, errors.New("Cannot have both a routes and a host or domain")) // TODO betrer message
	}
	if name := stringVal(yamlMap, "name", &errs); name != nil {
		appParams.Name = *name
	}

	if memory := bytesVal(yamlMap, "memory", &errs); memory != nil {
		appParams.Memory = *memory
	}

	if instanceCount := intVal(yamlMap, "instances", &errs); instanceCount != nil {
		appParams.InstanceCount = *instanceCount
	}

	if len(errs) > 0 {
		message := ""
		for _, err := range errs {
			message = message + fmt.Sprintf("%s\n", err.Error())
		}
		return plugin_models.GetAppModel{}, errors.New(message)
	}
	return appParams, nil
}

func removeDuplicatedValue(ary []string) []string {
	if ary == nil {
		return nil
	}

	m := make(map[string]bool)
	for _, v := range ary {
		m[v] = true
	}

	newAry := []string{}
	for _, val := range ary {
		if m[val] {
			newAry = append(newAry, val)
			m[val] = false
		}
	}
	return newAry
}

func checkForNulls(yamlMap map[string]interface{}) error {
	var errs []error
	for key, value := range yamlMap {
		if key == "command" || key == "buildpack" {
			break
		}
		if value == nil {
			errs = append(errs, fmt.Errorf("{{.PropertyName}} should not be null", map[string]interface{}{"PropertyName": key}))
		}
	}

	if len(errs) > 0 {
		message := ""
		for i := range errs {
			message = message + fmt.Sprintf("%s\n", errs[i].Error())
		}
		return errors.New(message)
	}

	return nil
}

func stringVal(yamlMap map[string]interface{}, key string, errs *[]error) *string {
	val := yamlMap[key]
	if val == nil {
		return nil
	}
	result, ok := val.(string)
	if !ok {
		*errs = append(*errs, fmt.Errorf("{{.PropertyName}} must be a string value", map[string]interface{}{"PropertyName": key}))
		return nil
	}
	return &result
}

func bytesVal(yamlMap map[string]interface{}, key string, errs *[]error) *int64 {
	yamlVal := yamlMap[key]
	if yamlVal == nil {
		return nil
	}

	stringVal := coerceToString(yamlVal)
	value, err := formatters.ToMegabytes(stringVal)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("Invalid value for '{{.PropertyName}}': {{.StringVal}}\n{{.Error}}",
			map[string]interface{}{
				"PropertyName": key,
				"Error":        err.Error(),
				"StringVal":    stringVal,
			}))
		return nil
	}
	return &value
}

func intVal(yamlMap map[string]interface{}, key string, errs *[]error) *int {
	var (
		intVal int
		err    error
	)

	switch val := yamlMap[key].(type) {
	case string:
		intVal, err = strconv.Atoi(val)
	case int:
		intVal = val
	case int64:
		intVal = int(val)
	case nil:
		return nil
	default:
		err = fmt.Errorf("Expected {{.PropertyName}} to be a number, but it was a {{.PropertyType}}.",
			map[string]interface{}{"PropertyName": key, "PropertyType": val})
	}

	if err != nil {
		*errs = append(*errs, err)
		return nil
	}

	return &intVal
}

func coerceToString(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

func sliceOrNil(yamlMap map[string]interface{}, key string, errs *[]error) []string {
	if _, ok := yamlMap[key]; !ok {
		return nil
	}

	var err error
	stringSlice := []string{}

	sliceErr := fmt.Errorf("Expected {{.PropertyName}} to be a list of strings.", map[string]interface{}{"PropertyName": key})

	switch input := yamlMap[key].(type) {
	case []interface{}:
		for _, value := range input {
			stringValue, ok := value.(string)
			if !ok {
				err = sliceErr
				break
			}
			stringSlice = append(stringSlice, stringValue)
		}
	default:
		err = sliceErr
	}

	if err != nil {
		*errs = append(*errs, err)
		return []string{}
	}

	return stringSlice
}

func RoutesFromManifest(defaultDomain string, Hosts []string, Domains []string) []plugin_models.GetApp_RouteSummary {

	manifestRoutes := make([]plugin_models.GetApp_RouteSummary, 0)

	for _, host := range Hosts {
		if Domains == nil {
			manifestRoutes = append(manifestRoutes, plugin_models.GetApp_RouteSummary{Host: host, Domain: plugin_models.GetApp_DomainFields{Name: defaultDomain}})
			continue
		}

		for _, domain := range Domains {
			manifestRoutes = append(manifestRoutes, plugin_models.GetApp_RouteSummary{Host: host, Domain: plugin_models.GetApp_DomainFields{Name: domain}})
		}
	}

	return manifestRoutes
}

func parseRoutes(cfDomains CfDomains, input map[string]interface{}, errs *[]error) []plugin_models.GetApp_RouteSummary {
	if _, ok := input["routes"]; !ok {
		return nil
	}

	genericRoutes, ok := input["routes"].([]interface{})
	if !ok {
		*errs = append(*errs, fmt.Errorf("'routes' should be a list"))
		return nil
	}

	manifestRoutes := []plugin_models.GetApp_RouteSummary{}
	for _, genericRoute := range genericRoutes {
		route, ok := genericRoute.(map[interface{}]interface{})
		if !ok {
			*errs = append(*errs, fmt.Errorf("each route in 'routes' must have a 'route' property"))
			continue
		}

		if routeVal, exist := route["route"]; exist {
			routeWithoutPath, path := findPath(routeVal.(string))

			routeWithoutPathAndPort, port, err := findPort(routeWithoutPath)
			if err != nil {
				*errs = append(*errs, err)
			}
			hostname, domain, err := findDomain(cfDomains, routeWithoutPathAndPort)
			if err != nil {
				*errs = append(*errs, err)
			}
			manifestRoutes = append(manifestRoutes, plugin_models.GetApp_RouteSummary{

				// HTTP routes include a domain, an optional hostname, and an optional context path
				Host:   hostname,
				Domain: domain,
				Path:   path,
				Port:   port,
			})
		} else {
			*errs = append(*errs, fmt.Errorf("each route in 'routes' must have a 'route' property"))
		}
	}

	return manifestRoutes
}

func findPath(routeName string) (string, string) {
	routeSlice := strings.Split(routeName, "/")
	return routeSlice[0], strings.Join(routeSlice[1:], "/")
}

func findPort(routeName string) (string, int, error) {
	var err error
	routeSlice := strings.Split(routeName, ":")
	port := 0
	if len(routeSlice) == 2 {
		port, err = strconv.Atoi(routeSlice[1])
		if err != nil {
			return "", 0, errors.New(fmt.Sprintf("Invalid port for route %s", routeName))
		}
	}
	return routeSlice[0], port, nil
}

func findDomain(cfDomains CfDomains, routeName string) (string, plugin_models.GetApp_DomainFields, error) {
	host, domain := decomposeRoute(cfDomains.SharedDomains, routeName)
	if domain == nil {
		host, domain = decomposeRoute(cfDomains.PrivateDomains, routeName)
	}
	if domain == nil {

		return "", plugin_models.GetApp_DomainFields{}, fmt.Errorf(
			"The route %s did not match any existing domains",
			routeName,
		)
	}
	return host, *domain, nil
}

func decomposeRoute(allowedDomains []string, routeName string) (string, *plugin_models.GetApp_DomainFields) {

	var testDomain = func(routeName string) (*plugin_models.GetApp_DomainFields, bool) {

		domain := &plugin_models.GetApp_DomainFields{}
		for _, possibleDomain := range allowedDomains {
			if possibleDomain == routeName {
				domain.Name = routeName
				return domain, true
			}
		}
		return domain, false
	}

	domain, found := testDomain(routeName)
	if found {
		return "", domain
	}

	routeParts := strings.Split(routeName, ".")
	domain, found = testDomain(strings.Join(routeParts[1:], "."))
	if found {
		return routeParts[0], domain
	}

	return "", nil
}

func (manifest *Manifest) GetAppParams(appName string, cfDomains CfDomains) *plugin_models.GetAppModel {
	var err error
	apps, err := manifest.Applications(cfDomains)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	for index, app := range apps {
		if isHostOrDomainEmpty(app) {
			continue
		}
		if app.Name != "" && app.Name != appName {
			continue
		}
		return &apps[index]
	}

	return nil
}

func isHostOrDomainEmpty(app plugin_models.GetAppModel) bool {
	for _, route := range app.Routes {
		if route.Host != "" || route.Domain.Name != "" {
			return false
		}
	}
	return true
}
