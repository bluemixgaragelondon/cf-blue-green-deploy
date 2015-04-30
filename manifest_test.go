package main_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/manifest"
	"github.com/cloudfoundry/cli/generic"
	go_i18n "github.com/nicksnyder/go-i18n/i18n"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"
)

var _ = Describe("Manifest reader", func() {
	// testing code that calls into cf cli requires T to point to a translate func
	i18n.T, _ = go_i18n.Tfunc("")

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
  hosts:
  - host1
  - host2
  domains:
  - example1.com
  - example2.com`,
			}

			It("Returns the correct app", func() {
				Expect(*GetAppFromManifest(&repo, "foo").Name).To(Equal("foo"))
				Expect(*GetAppFromManifest(&repo, "foo").Hosts).To(ConsistOf("host1", "host2"))
				Expect(*GetAppFromManifest(&repo, "foo").Domains).To(ConsistOf("example1.com", "example2.com"))
			})
		})
	})

	Context("When no manifest file is present", func() {
		repo := FakeRepo{err: errors.New("Error finding manifest")}

		It("Returns nil", func() {
			Expect(GetAppFromManifest(&repo, "foo")).To(BeNil())
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
