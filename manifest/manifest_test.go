package manifest

import (
	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/cloudfoundry-incubator/candiedyaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	Context("For a manifest", func() {
		It("parses known manifest keys", func() {
			m := &Manifest{
				Path: "./manifest.yml",
				Data: map[string]interface{}{
					"disk_quota": "512M",
					"memory":     "256M",
					"instances":  1,
				},
			}
			apps, err := m.Applications(CfDomains{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(apps)).To(Equal(1))

			Expect(apps[0].DiskQuota).To(Equal(int64(512)))
			Expect(apps[0].Memory).To(Equal(int64(256)))
			Expect(apps[0].InstanceCount).To(Equal(1))
		})
	})

	Context("For a manifest with no applications section", func() {

		input := map[string]interface{}{
			"host": "bob",
			"routes": []interface{}{
				map[interface{}]interface{}{"route": "example.com"},
				map[interface{}]interface{}{"route": "www.example.com/foo"},
				map[interface{}]interface{}{"route": "tcp-example.com:1234"},
			},
		}
		m := &Manifest{}

		Context("the getAppMaps function", func() {
			appMaps, err := m.getAppMaps(input)
			It("does not error", func() {
				Expect(err).To(BeNil())
			})

			It("should return one entry", func() {
				Expect(len(appMaps)).To(Equal(1))
			})

			It("should return global properties", func() {
				Expect(appMaps).To(Equal([]map[string]interface{}{input}))
			})
		})

		Context("the parseRoutes function", func() {
			errs := []error{}
			routeStuff := parseRoutes(CfDomains{SharedDomains: []string{"example.com"}, PrivateDomains: []string{"tcp-example.com"}}, input, &errs)

			It("does not error", func() {
				Expect(len(errs)).To(Equal(0))
			})

			It("should return three routes", func() {
				Expect(len(routeStuff)).To(Equal(3))
			})

			It("should return global properties", func() {
				// We're only testing for domain because of limitations in the route struct
				Expect(routeStuff[0].Domain.Name).To(Equal("example.com"))
			})
		})
	})

	Context("For a manifest with an applications section", func() {
		applicationsContents := []interface{}{map[string]string{
			"fred": "hello",
		}}
		input := map[string]interface{}{
			"applications": applicationsContents,
			"host":         "bob",
		}

		m := &Manifest{}
		appMaps, err := m.getAppMaps(input)

		Context("the AppMaps function", func() {
			It("does not error", func() {
				Expect(err).To(BeNil())
			})

			It("should not alter what gets passed in", func() {

				Expect(input["applications"]).To(Equal(applicationsContents))
				// Make sure this doesn't change what's passed in
				Expect(input["applications"]).To(Equal(applicationsContents))

			})

			It("should return one entry", func() {
				Expect(len(appMaps)).To(Equal(1))
			})

			It("should merge global properties with application-level properties", func() {

				Expect(appMaps[0]["host"]).To(Equal("bob"))
				Expect(appMaps[0]["fred"]).To(Equal("hello"))

			})
		})
	})

	Context("For a manifest with two applications in the applications section", func() {
		applicationsContents := []interface{}{map[string]string{
			"fred": "hello",
		},
			map[string]string{
				"george": "goodbye",
			}}
		input := map[string]interface{}{
			"applications": applicationsContents,
			"host":         "bob",
		}

		m := &Manifest{}
		appMaps, err := m.getAppMaps(input)

		Context("the AppMaps function", func() {
			It("does not error", func() {
				Expect(err).To(BeNil())
			})

			It("should not alter what gets passed in", func() {

				Expect(input["applications"]).To(Equal(applicationsContents))
				// Make sure this doesn't change what's passed in
				Expect(input["applications"]).To(Equal(applicationsContents))

			})

			It("should return two entry", func() {
				Expect(len(appMaps)).To(Equal(2))
			})

			It("should merge global properties with application-level properties", func() {

				Expect(appMaps[0]["host"]).To(Equal("bob"))
				Expect(appMaps[0]["fred"]).To(Equal("hello"))
				Expect(appMaps[0]["george"]).To(BeNil())

				Expect(appMaps[1]["host"]).To(Equal("bob"))
				Expect(appMaps[1]["george"]).To(Equal("goodbye"))
				Expect(appMaps[1]["fred"]).To(BeNil())

			})
		})
	})

})

var _ = Describe("CloneWithExclude", func() {

	Context("When the map contains some values and excludeKey exists", func() {

		input := map[string]interface{}{
			"one":   1,
			"two":   2138,
			"three": 1908,
		}

		excludeKey := "two"

		actual := cloneWithExclude(input, excludeKey)

		It("should return a new map without the excludeKey", func() {

			expected := map[string]interface{}{
				"one":   1,
				"three": 1908,
			}

			Expect(actual).To(Equal(expected))
		})

		It("should not alter the original map", func() {
			Expect(input["two"]).To(Equal(2138))
		})
	})

	Context("When the map contains some values and excludeKey does not exist", func() {
		It("should return a new map with the same contents as the original", func() {
			input := map[string]interface{}{
				"one":   1,
				"two":   2138,
				"three": 1908,
			}

			excludeKey := "four"

			actual := cloneWithExclude(input, excludeKey)

			Expect(actual).To(Equal(input))
		})
	})

	Context("When the map contains a key that includes the excludeKey", func() {
		It("should return a new map with the same contents as the original", func() {
			input := map[string]interface{}{
				"one":       1,
				"two":       2138,
				"threefour": 1908,
			}

			excludeKey := "four"

			actual := cloneWithExclude(input, excludeKey)

			Expect(actual).To(Equal(input))
		})
	})

	Context("When the map is empty", func() {
		It("should return a new empty map", func() {
			input := map[string]interface{}{}

			excludeKey := "one"

			actual := cloneWithExclude(input, excludeKey)

			Expect(actual).To(Equal(input))
		})
	})

	Context("when the manifest contains a different app name", func() {
		manifest := manifestFromYamlString(`---
	      name: bar
	      host: foo`)

		It("Returns nil", func() {
			Expect(manifest.GetAppParams("appname", CfDomains{DefaultDomain: "domain"})).To(BeNil())
		})

		Context("when the manifest contain a host but no app name", func() {
			manifest := manifestFromYamlString(`---
host: foo`)

			It("Returns params that contain the host", func() {

				routes := manifest.GetAppParams("foo", CfDomains{DefaultDomain: "something.com"}).Routes
				Expect(routes).To(ConsistOf(
					plugin_models.GetApp_RouteSummary{Host: "foo", Domain: plugin_models.GetApp_DomainFields{Name: "something.com"}},
				))
			})
		})

		Describe("Route Lister", func() {
			It("returns a list of Routes from the manifest", func() {
				manifest := manifestFromYamlString(`---
name: foo
hosts:
 - host1
 - host2
domains:
 - example.com
 - example.net`)

				params := manifest.GetAppParams("foo", CfDomains{DefaultDomain: "example.com"})

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
					manifest := manifestFromYamlString(`---
name: foo
hosts:
 - host1
 - host2`)

					params := manifest.GetAppParams("foo", CfDomains{DefaultDomain: "example.com"})
					Expect(params).ToNot(BeNil())
					Expect(params.Routes).ToNot(BeNil())

					routes := params.Routes
					Expect(routes).To(ConsistOf(
						plugin_models.GetApp_RouteSummary{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
						plugin_models.GetApp_RouteSummary{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
					))
				})
			})

			Context("when app has just routes, no hosts or domains", func() {
				It("returns those routes", func() {
					manifest := manifestFromYamlString(`---
name: foo
routes:
 - route: route1.domain1
 - route: route2.domain2`)

					params := manifest.GetAppParams("foo", CfDomains{DefaultDomain: "example.com", SharedDomains: []string{"route1.domain1", "route2.domain2"}})
					Expect(params).ToNot(BeNil())
					Expect(params.Routes).ToNot(BeNil())

					routes := params.Routes
					Expect(routes).To(ConsistOf(
						plugin_models.GetApp_RouteSummary{Domain: plugin_models.GetApp_DomainFields{Name: "route1.domain1"}},
						plugin_models.GetApp_RouteSummary{Domain: plugin_models.GetApp_DomainFields{Name: "route2.domain2"}},
					))
				})
			})

			Context("when app has routes, and an app name, but no domain", func() {
				It("correctly identifies the host and domain components", func() {

					manifest := manifestFromYamlString(`---
name: my-app
routes:
 - route: my-app.example.io`)

					params := manifest.GetAppParams("my-app", CfDomains{DefaultDomain: "defaultdomain.com", PrivateDomains: []string{"example.io"}})
					Expect(params).ToNot(BeNil())
					Expect(params.Routes).ToNot(BeNil())

					routes := params.Routes
					Expect(routes).To(ConsistOf(
						plugin_models.GetApp_RouteSummary{Host: "my-app", Domain: plugin_models.GetApp_DomainFields{Name: "example.io"}},
					))
				})

				Context("the app name is repeated in the domain", func() {
					It("correctly identifies the host and domain components", func() {

						manifest := manifestFromYamlString(`---
name: my-app
routes:
 - route: my-app.example.my-app.io`)

						params := manifest.GetAppParams("my-app", CfDomains{DefaultDomain: "defaultdomain.com", PrivateDomains: []string{"example.my-app.io"}})
						Expect(params).ToNot(BeNil())
						Expect(params.Routes).ToNot(BeNil())

						routes := params.Routes
						Expect(routes).To(ConsistOf(
							plugin_models.GetApp_RouteSummary{Host: "my-app", Domain: plugin_models.GetApp_DomainFields{Name: "example.my-app.io"}},
						))
					})
				})
			})

			Context("when no matching application", func() {
				It("returns nil", func() {
					manifest := manifestFromYamlString(``)

					Expect(manifest.GetAppParams("foo", CfDomains{DefaultDomain: "example.com"})).To(BeNil())
				})
			})
		})

	})

	Context("when the manifest contains multiple apps with 1 matching", func() {
		manifest := manifestFromYamlString(`---
applications:
 - name: bar
   host: barhost
 - name: foo
   hosts:
    - host1
    - host2
   domains:
    - example1.com
    - example2.com`)
		It("Returns the correct app", func() {

			var hostNames []string
			var domainNames []string

			appParams := manifest.GetAppParams("foo", CfDomains{})
			Expect(appParams).ToNot(BeNil())

			routes := appParams.Routes
			Expect(routes).ToNot(BeNil())
			for _, route := range routes {
				hostNames = append(hostNames, route.Host)
				domainNames = append(domainNames, route.Domain.Name)
			}

			hostNames = deDuplicate(hostNames)
			domainNames = deDuplicate(domainNames)

			Expect(manifest.GetAppParams("foo", CfDomains{}).Name).To(Equal("foo"))
			Expect(hostNames).To(ConsistOf("host1", "host2"))
			Expect(domainNames).To(ConsistOf("example1.com", "example2.com"))
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

func manifestFromYamlString(yamlString string) *Manifest {
	yamlMap := make(map[string]interface{})
	candiedyaml.Unmarshal([]byte(yamlString), &yamlMap)
	return &Manifest{Data: yamlMap}
}
