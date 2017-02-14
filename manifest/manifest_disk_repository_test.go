package manifest_test

import (
	"code.cloudfoundry.org/cli/plugin/models"
	"errors"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest reader", func() {

	Context("when a custom manifest file is provided", func() {
		It("should load that file from the repository", func() {
			repo := fakes.FakeRepository{Yaml: `---
        host: foo`,
			}
			manifestAppFinder := manifest.ManifestAppFinder{
				AppName:       "foo",
				Repo:          &repo,
				ManifestPath:  "manifest-tst.yml",
				DefaultDomain: "example.com",
			}

			manifestAppFinder.AppParams()

			Expect(repo.Path).To(Equal("manifest-tst.yml"))
		})
	})

	Context("when a custom manifest file is not provided", func() {
		It("should load that file from the repository", func() {
			repo := fakes.FakeRepository{Yaml: `---
        host: foo`,
			}
			manifestAppFinder := manifest.ManifestAppFinder{
				AppName:       "foo",
				Repo:          &repo,
				DefaultDomain: "example.com",
			}

			manifestAppFinder.AppParams()

			Expect(repo.Path).To(Equal("./"))
		})
	})

	Context("When the manifest file is present", func() {
		Context("when the manifest contain a host but no app name", func() {
			repo := fakes.FakeRepository{Yaml: `---
        host: foo`,
			}
			manifestAppFinder := manifest.ManifestAppFinder{AppName: "foo", Repo: &repo}

			It("Returns params that contain the host", func() {

				var hostNames []string

				for _, route := range manifestAppFinder.AppParams().Routes {
					hostNames = append(hostNames, route.Host)
				}

				Expect(hostNames).To(ContainElement("foo"))
			})
		})

		Context("when the manifest contains a different app name", func() {
			repo := fakes.FakeRepository{Yaml: `---
        name: bar
        host: foo`,
			}
			manifestAppFinder := manifest.ManifestAppFinder{AppName: "foo", Repo: &repo}

			It("Returns nil", func() {
				Expect(manifestAppFinder.AppParams()).To(BeNil())
			})
		})

		Context("when the manifest contains multiple apps with 1 matching", func() {
			repo := fakes.FakeRepository{Yaml: `---
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
			manifestAppFinder := manifest.ManifestAppFinder{AppName: "foo", Repo: &repo}

			var hostNames []string
			var domainNames []string

			for _, route := range manifestAppFinder.AppParams().Routes {
				hostNames = append(hostNames, route.Host)
				domainNames = append(domainNames, route.Domain.Name)
			}

			hostNames = deDuplicate(hostNames)
			domainNames = deDuplicate(domainNames)

			It("Returns the correct app", func() {
				Expect(manifestAppFinder.AppParams().Name).To(Equal("foo"))
				Expect(hostNames).To(ConsistOf("host1", "host2"))
				Expect(domainNames).To(ConsistOf("example1.com", "example2.com"))
			})
		})
	})

	Context("When no manifest file is present", func() {
		repo := fakes.FakeRepository{Err: errors.New("Error finding manifest")}
		manifestAppFinder := manifest.ManifestAppFinder{AppName: "foo", Repo: &repo}

		It("Returns nil", func() {
			Expect(manifestAppFinder.AppParams()).To(BeNil())
		})
	})

	Context("When manifest file is empty", func() {
		repo := fakes.FakeRepository{Yaml: ``}
		manifestAppFinder := manifest.ManifestAppFinder{AppName: "foo", Repo: &repo}

		It("Returns nil", func() {
			Expect(manifestAppFinder.AppParams()).To(BeNil())
		})
	})

	Describe("Route Lister", func() {
		It("returns a list of Routes from the manifest", func() {
			repo := fakes.FakeRepository{Yaml: `---
          name: foo
          hosts:
          - host1
          - host2
          domains:
          - example.com
          - example.net`,
			}
			manifestAppFinder := manifest.ManifestAppFinder{
				AppName:       "foo",
				Repo:          &repo,
				DefaultDomain: "example.com",
			}

			params := manifestAppFinder.AppParams()

			Expect(params).ToNot(BeNil())
			Expect(params.Routes).ToNot(BeNil())

			routes := params.Routes
			Expect(routes).To(ConsistOf(
				plugin_models.GetApp_RouteSummary{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
				plugin_models.GetApp_RouteSummary{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "example.net"}},
				plugin_models.GetApp_RouteSummary{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
				plugin_models.GetApp_RouteSummary{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "example.net"}},
			))
		})

		Context("when app has just hosts, no domains", func() {
			It("returns Application", func() {
				repo := fakes.FakeRepository{Yaml: `---
          name: foo
          hosts:
          - host1
          - host2`,
				}
				manifestAppFinder := manifest.ManifestAppFinder{
					AppName:       "foo",
					Repo:          &repo,
					DefaultDomain: "example.com",
				}
				params := manifestAppFinder.AppParams()
				Expect(params).ToNot(BeNil())
				Expect(params.Routes).ToNot(BeNil())

				routes := params.Routes
				Expect(routes).To(ConsistOf(
					plugin_models.GetApp_RouteSummary{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
					plugin_models.GetApp_RouteSummary{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
				))
			})
		})

		PContext("when app has just routes, no hosts or domains", func() {
			It("returns those routes", func() {
				repo := fakes.FakeRepository{Yaml: `---
          name: foo
          routes:
          - route1.domain1
          - route2.domain2`,
				}
				manifestAppFinder := manifest.ManifestAppFinder{
					AppName:       "foo",
					Repo:          &repo,
					DefaultDomain: "example.com",
				}
				routes := manifestAppFinder.AppParams()

				Expect(routes).To(ConsistOf(
					plugin_models.GetApp_RouteSummary{Host: "route1", Domain: plugin_models.GetApp_DomainFields{Name: "domain1"}},
					plugin_models.GetApp_RouteSummary{Host: "route2", Domain: plugin_models.GetApp_DomainFields{Name: "domain2"}},
				))
			})
		})

		Context("when no matching application", func() {
			It("returns nil", func() {
				repo := fakes.FakeRepository{Yaml: ``}
				manifestAppFinder := manifest.ManifestAppFinder{
					AppName:       "foo",
					Repo:          &repo,
					DefaultDomain: "example.com",
				}

				Expect(manifestAppFinder.AppParams()).To(BeNil())
			})
		})
	})
})

func deDuplicate(ary []string) []string {
	if ary == nil {
		return nil
	}

	m := make(map[string]bool)
	for _, v := range ary {
		m[v] = true
	}

	newAry := []string{}
	for _, val := range ary {
		if m[val] {
			newAry = append(newAry, val)
			m[val] = false
		}
	}
	return newAry
}
