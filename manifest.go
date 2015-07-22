package main

import (
	"fmt"

	"github.com/cloudfoundry/cli/cf/manifest"
	"github.com/cloudfoundry/cli/cf/models"
)

type ManifestReader func(manifest.ManifestRepository, string) *models.AppParams

type ManifestAppFinder struct {
	Repo    manifest.ManifestRepository
	AppName string
}

func (f *ManifestAppFinder) RoutesFromManifest(defaultDomain string) []Route {
	if appParams := f.AppParams(); appParams != nil {

		manifestRoutes := make([]Route, 0)

		for _, host := range *appParams.Hosts {
			if appParams.Domains == nil {
				manifestRoutes = append(manifestRoutes, Route{Host: host, Domain: Domain{Name: defaultDomain}})
				continue
			}

			for _, domain := range *appParams.Domains {
				manifestRoutes = append(manifestRoutes, Route{Host: host, Domain: Domain{Name: domain}})
			}
		}

		return manifestRoutes
	}
	return nil
}

func (f *ManifestAppFinder) AppParams() *models.AppParams {
	manifest, err := f.Repo.ReadManifest("./")
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
