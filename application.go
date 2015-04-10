package main

import (
	"fmt"
	"regexp"
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

func (a *Application) DefaultRoute() Route {
	for _, route := range a.Routes {
		if route.Host == a.Name {
			return route
		}
	}

	return Route{}
}

func (a *Application) DefaultRouteWhenWeWillCorrectlySetTheAppNameFromTheCommandLineNotFromCloudController() Route {
	r := regexp.MustCompile(fmt.Sprintf("^%s-[0-9]{14}$", a.Name))

	for _, route := range a.Routes {
		if r.MatchString(route.Host) {
			return route
		}
	}

	return Route{}
}
