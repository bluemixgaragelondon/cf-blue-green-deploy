package main

import (
	"github.com/cloudfoundry/cli/cf/manifest"
	"github.com/cloudfoundry/cli/cf/models"
)

type ManifestReader func(manifest.ManifestRepository, string) *models.AppParams

func GetAppFromManifest(repo manifest.ManifestRepository, appName string) *models.AppParams {
	m, err := repo.ReadManifest("")
	if err != nil {
		return nil
	}

	apps, _ := m.Applications()

	if len(apps) == 1 {
		if apps[0].Name != nil && *apps[0].Name != appName {
			return nil
		} else {
			return &apps[0]
		}
	} else {
		return findApp(apps, appName)
	}
}

func findApp(apps []models.AppParams, appName string) *models.AppParams {
	for index, app := range apps {
		if *app.Name == appName {
			return &apps[index]
		}
	}
	return nil
}
