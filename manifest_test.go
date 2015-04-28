package main_test

import (
	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/cloudfoundry/cli/cf/manifest"
	"github.com/cloudfoundry/cli/generic"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"
)

var _ = Describe("Manifest reader", func() {
	Context("When the manifest file is present", func() {
		Context("when the manifest contain a host but no app name", func() {
			repo := FakeRepo{yaml: `---
        host: foo`,
			}

			It("Returns params that contain the host", func() {
				Expect(*GetAppFromManifest(&repo, "foo").Hosts).To(ContainElement("foo"))
			})
		})

		Context("when the manifest contains a different app name", func() {
			repo := FakeRepo{yaml: `---
        name: bar
        host: foo`,
			}

			It("Returns nil", func() {
				Expect(GetAppFromManifest(&repo, "foo")).To(BeNil())
			})
		})

		Context("when the manifest contains multiple apps with 1 matching", func() {
			repo := FakeRepo{yaml: `---
        applications:
          - name: bar
            host: barhost
          - name: foo
            host: foohost`,
			}

			It("Returns the host for the passed app name", func() {
				Expect(*GetAppFromManifest(&repo, "foo").Hosts).To(ContainElement("foohost"))
			})
		})
	})
})

type FakeRepo struct {
	yaml string
	err  error
}

func (r *FakeRepo) ReadManifest(path string) (*manifest.Manifest, error) {
	yamlMap := generic.NewMap()
	candiedyaml.Unmarshal([]byte(r.yaml), yamlMap)
	return &manifest.Manifest{Data: yamlMap}, r.err
}
