package fakes

import (
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
	"github.com/cloudfoundry-incubator/candiedyaml"
)

type FakeRepository struct {
	Err  error
	Path string
	Yaml string
}

func (r *FakeRepository) ReadManifest(path string) (*manifest.Manifest, error) {
	r.Path = path
	yamlMap := make(map[string]interface{})
	candiedyaml.Unmarshal([]byte(r.Yaml), &yamlMap)
	return &manifest.Manifest{Data: yamlMap}, r.Err
}
