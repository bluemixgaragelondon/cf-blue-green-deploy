package main_test

import (
	"errors"

	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/manifest"
	"github.com/cloudfoundry/cli/generic"
	go_i18n "github.com/nicksnyder/go-i18n/i18n"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest reader", func() {
	// testing code that calls into cf cli requires T to point to a translate func
	i18n.T, _ = go_i18n.Tfunc("")

	Context("when a custom manifest file is provided", func() {
		It("should load that file from the repository", func() {
			repo := FakeRepo{yaml: `---
        host: foo`,
			}
			manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo, ManifestPath: "manifest-tst.yml"}

			manifestAppFinder.RoutesFromManifest("example.com")

			Expect(repo.path).To(Equal("manifest-tst.yml"))
		})
	})

	Context("when a custom manifest file is not provided", func() {
		It("should load that file from the repository", func() {
			repo := FakeRepo{yaml: `---
        host: foo`,
			}
			manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo}

			manifestAppFinder.RoutesFromManifest("example.com")

			Expect(repo.path).To(Equal("./"))
		})
	})

	Context("When the manifest file is present", func() {
		Context("when the manifest contain a host but no app name", func() {
			repo := FakeRepo{yaml: `---
        host: foo`,
			}
			manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo}

			It("Returns params that contain the host", func() {
				Expect(*manifestAppFinder.AppParams().Hosts).To(ContainElement("foo"))
			})
		})

		Context("when the manifest contains a different app name", func() {
			repo := FakeRepo{yaml: `---
        name: bar
        host: foo`,
			}
			manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo}

			It("Returns nil", func() {
				Expect(manifestAppFinder.AppParams()).To(BeNil())
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
			manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo}

			It("Returns the correct app", func() {
				Expect(*manifestAppFinder.AppParams().Name).To(Equal("foo"))
				Expect(*manifestAppFinder.AppParams().Hosts).To(ConsistOf("host1", "host2"))
				Expect(*manifestAppFinder.AppParams().Domains).To(ConsistOf("example1.com", "example2.com"))
			})
		})
	})

	Context("When no manifest file is present", func() {
		repo := FakeRepo{err: errors.New("Error finding manifest")}
		manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo}

		It("Returns nil", func() {
			Expect(manifestAppFinder.AppParams()).To(BeNil())
		})
	})

	Context("When manifest file is empty", func() {
		repo := FakeRepo{yaml: ``}
		manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo}

		It("Returns nil", func() {
			Expect(manifestAppFinder.AppParams()).To(BeNil())
		})
	})

	Describe("Route Lister", func() {
		It("returns a list of Routes from the manifest", func() {
			repo := FakeRepo{yaml: `---
          name: foo
          hosts:
          - host1
          - host2
          domains:
          - example.com
          - example.net`,
			}
			manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo}

			routes := manifestAppFinder.RoutesFromManifest("example.com")

			Expect(routes).To(ConsistOf(
				Route{Host: "host1", Domain: Domain{Name: "example.com"}},
				Route{Host: "host1", Domain: Domain{Name: "example.net"}},
				Route{Host: "host2", Domain: Domain{Name: "example.com"}},
				Route{Host: "host2", Domain: Domain{Name: "example.net"}},
			))
		})

		Context("when app has just hosts, no domains", func() {
			It("returns Application", func() {
				repo := FakeRepo{yaml: `---
          name: foo
          hosts:
          - host1
          - host2`,
				}
				manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo}
				routes := manifestAppFinder.RoutesFromManifest("example.com")

				Expect(routes).To(ConsistOf(
					Route{Host: "host1", Domain: Domain{Name: "example.com"}},
					Route{Host: "host2", Domain: Domain{Name: "example.com"}},
				))
			})
		})

		Context("when no matching application", func() {
			It("returns nil", func() {
				repo := FakeRepo{yaml: ``}
				manifestAppFinder := ManifestAppFinder{AppName: "foo", Repo: &repo}

				Expect(manifestAppFinder.RoutesFromManifest("example.com")).To(BeNil())
			})
		})
	})
})

type FakeRepo struct {
	yaml string
	err  error
	path string
}

func (r *FakeRepo) ReadManifest(path string) (*manifest.Manifest, error) {
	r.path = path
	yamlMap := generic.NewMap()
	candiedyaml.Unmarshal([]byte(r.yaml), yamlMap)
	return &manifest.Manifest{Data: yamlMap}, r.err
}
