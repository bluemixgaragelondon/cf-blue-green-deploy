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

func (r Route) FQDN() string {
	return fmt.Sprintf("%v.%v", r.Host, r.Domain.Name)
}
