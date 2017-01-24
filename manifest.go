package main

import (
	"fmt"

	"code.cloudfoundry.org/cli/cf/manifest"
	"code.cloudfoundry.org/cli/cf/models"
)

type ManifestReader func(manifest.Repository, string) *models.AppParams

type ManifestAppFinder struct {
	Repo    manifest.Repository
	ManifestPath string
	AppName string
}

func (f *ManifestAppFinder) RoutesFromManifest(defaultDomain string) []Route {
	if appParams := f.AppParams(); appParams != nil {

		manifestRoutes := make([]Route, 0)

		for _, host := range appParams.Hosts {
			if appParams.Domains == nil {
				manifestRoutes = append(manifestRoutes, Route{Host: host, Domain: Domain{Name: defaultDomain}})
				continue
			}

			for _, domain := range appParams.Domains {
				manifestRoutes = append(manifestRoutes, Route{Host: host, Domain: Domain{Name: domain}})
			}
		}

		return manifestRoutes
	}
	return nil
}

func (f *ManifestAppFinder) AppParams() *models.AppParams {
	var manifest *manifest.Manifest
	var err error
	if f.ManifestPath == "" {
		manifest, err = f.Repo.ReadManifest("./")
	} else {
		manifest, err = f.Repo.ReadManifest(f.ManifestPath)
	}

	if err != nil {
		return nil
	}

	apps, err := manifest.Applications()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	for index, app := range apps {
		if app.IsHostEmpty() {
			continue
		}

		if app.Name != nil && *app.Name != f.AppName {
			continue
		}

		return &apps[index]
	}

	return nil
}
