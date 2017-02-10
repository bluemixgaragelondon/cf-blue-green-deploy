package main

import (
	"code.cloudfoundry.org/cli/plugin/models"
	"fmt"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
)

type ManifestReader func(manifest.Repository, string) *plugin_models.GetAppModel

type ManifestAppFinder struct {
	Repo          manifest.Repository
	ManifestPath  string
	AppName       string
	DefaultDomain string
}

func IsHostEmpty(app plugin_models.GetAppModel) bool {
	for _, route := range app.Routes {
		if route.Host != "" {
			return false
		}
	}
	return true
}

func (f *ManifestAppFinder) AppParams() *plugin_models.GetAppModel {
	var manifest *manifest.Manifest
	var err error
	if f.ManifestPath == "" {
		manifest, err = f.Repo.ReadManifest("./")
	} else {
		manifest, err = f.Repo.ReadManifest(f.ManifestPath)
	}

	if err != nil {
		fmt.Println(err)
		return nil
	}

	apps, err := manifest.Applications(f.DefaultDomain)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	for index, app := range apps {
		if IsHostEmpty(app) {
			continue
		}

		if app.Name != "" && app.Name != f.AppName {
			continue
		}

		return &apps[index]
	}
	return nil
}
