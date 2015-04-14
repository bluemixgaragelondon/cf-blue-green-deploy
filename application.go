package main

import "fmt"

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

func (r Route) FQDN() string {
	return fmt.Sprintf("%v.%v", r.Host, r.Domain.Name)
}
