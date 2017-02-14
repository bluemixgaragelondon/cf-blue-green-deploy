package fakes

import (
	"fmt"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
	"github.com/cloudfoundry-incubator/candiedyaml"
)

type FakeManifestReader struct {
	Yaml string
	Err  error
}

func (manifestReader *FakeManifestReader) Read() *manifest.Manifest {
	yamlMap := make(map[string]interface{})
	candiedyaml.Unmarshal([]byte(manifestReader.Yaml), &yamlMap)

	if manifestReader.Err != nil {
		fmt.Println(manifestReader.Err)
		return nil
	} else {
		return &manifest.Manifest{Data: yamlMap}
	}
}
