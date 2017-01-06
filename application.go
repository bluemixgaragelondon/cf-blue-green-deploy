package main

import (
	"code.cloudfoundry.org/cli/plugin/models"
	"fmt"
)

type Application struct {
	Name   string
	Routes []Route
}

type Route struct {
	Host   string
	Domain Domain
}

type Domain struct {
	Name string
}

func FQDN(r plugin_models.GetApp_RouteSummary) string {
	return fmt.Sprintf("%v.%v", r.Host, r.Domain.Name)
}
