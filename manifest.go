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

func (f *ManifestAppFinder) Application(defaultDomain string) *Application {
	if appParams := f.AppParams(); appParams != nil {
		app := Application{Name: *appParams.Name, DefaultDomain: defaultDomain}

		for _, host := range *appParams.Hosts {
			if appParams.Domains == nil {
				app.Routes = append(app.Routes, Route{Host: host, Domain: Domain{Name: app.DefaultDomain}})
				continue
			}

			for _, domain := range *appParams.Domains {
				app.Routes = append(app.Routes, Route{Host: host, Domain: Domain{Name: domain}})
			}
		}

		return &app
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
