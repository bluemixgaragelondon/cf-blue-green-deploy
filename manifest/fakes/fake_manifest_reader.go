package fakes

import (
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
	"github.com/cloudfoundry-incubator/candiedyaml"
)

type FakeManifestReader struct {
	Yaml string
	Err  error
}

func (manifestReader *FakeManifestReader) Read() (*manifest.Manifest, error) {
	yamlMap := make(map[string]interface{})
	candiedyaml.Unmarshal([]byte(manifestReader.Yaml), &yamlMap)

	if manifestReader.Err != nil {
		return nil, manifestReader.Err
	} else {
		return &manifest.Manifest{Data: yamlMap}, nil
	}
}
