package main_test

import (
	"errors"
	"fmt"

	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
	"code.cloudfoundry.org/cli/plugin/pluginfakes"
	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BGD Plugin", func() {

	Describe("blue green flow", func() {
		Context("when there is a previous live app", func() {
			It("calls methods in correct order", func() {
				b := &BlueGreenDeployFake{liveApp: &plugin_models.GetAppModel{Name: "app-name-live"}, appSshEnabled: false}
				p := CfPlugin{
					Deployer: b,
				}

				p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, &fakes.FakeManifestReader{}, NewArgs([]string{"bgd", "app-name"}))

				Expect(b.flow).To(Equal([]string{
					"delete old apps",
					"get current live app",
					"push app-name-new",
					"check ssh enablement for 'app-name'",
					"set ssh enablement for 'app-name-new' to 'false'",
					"unmap 1 routes from app-name-new",
					"mapped 1 routes",
					"rename app-name-live to app-name-old",
					"rename app-name-new to app-name",
					"unmap 0 routes from app-name-old",
				}))
			})

			Context("with an existing live route", func() {
				It("maps the live app routes to the new app", func() {

					liveAppRoutes := []plugin_models.GetApp_RouteSummary{
						{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
						{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
					}

					b := &BlueGreenDeployFake{
						liveApp: &plugin_models.GetAppModel{Name: "app-name-live",
							Routes: liveAppRoutes},
					}
					p := CfPlugin{
						Deployer: b,
					}

					p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, &fakes.FakeManifestReader{}, NewArgs([]string{"bgd", "app-name"}))

					Expect(b.mappedRoutes).To(ConsistOf(liveAppRoutes))
				})
			})

			Context("with an existing live route and manifest", func() {
				It("maps both manifest & live app routes", func() {
					liveAppRoutes := []plugin_models.GetApp_RouteSummary{
						{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
						{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
					}

					b := &BlueGreenDeployFake{
						liveApp: &plugin_models.GetAppModel{Name: "app-name-live",
							Routes: liveAppRoutes},
					}
					p := CfPlugin{
						Deployer: b,
					}
					repo := &fakes.FakeManifestReader{Yaml: `---
          name: app-name
          hosts:
           - man1
          domains:
           - example.com
        `}

					p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, repo, NewArgs([]string{"bgd", "app-name"}))

					expectedAppRoutes := append(liveAppRoutes, plugin_models.GetApp_RouteSummary{Host: "man1", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}})

					Expect(b.mappedRoutes).To(ConsistOf(expectedAppRoutes))
				})

				It("maps unique routes", func() {
					liveAppRoutes := []plugin_models.GetApp_RouteSummary{
						{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
						{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
					}

					b := &BlueGreenDeployFake{
						liveApp: &plugin_models.GetAppModel{Name: "app-name-live",
							Routes: liveAppRoutes},
					}
					p := CfPlugin{
						Deployer: b,
					}
					repo := &fakes.FakeManifestReader{Yaml: `---
          name: app-name
          hosts:
           - man1
           - host1
           - host2
          domains:
           - example.com
        `}

					p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, repo, NewArgs([]string{"bgd", "app-name"}))

					expectedAppRoutes := append(liveAppRoutes, plugin_models.GetApp_RouteSummary{Host: "man1", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}})

					Expect(b.mappedRoutes).To(ConsistOf(expectedAppRoutes))

				})
			})
		})

		Context("when there is no previous live app", func() {
			It("calls methods in correct order", func() {
				b := &BlueGreenDeployFake{liveApp: nil}
				p := CfPlugin{
					Deployer: b,
				}

				p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, &fakes.FakeManifestReader{}, NewArgs([]string{"bgd", "app-name"}))

				Expect(b.flow).To(Equal([]string{
					"delete old apps",
					"get current live app",
					"push app-name-new",
					"unmap 1 routes from app-name-new",
					"mapped 1 routes",
					"rename app-name-new to app-name",
				}))
			})
		})

		Context("when app has manifest", func() {
			Context("when manifest uses hosts and domains", func() {
				It("maps manifest routes", func() {
					b := &BlueGreenDeployFake{liveApp: nil}
					p := CfPlugin{
						Deployer: b,
					}
					repo := &fakes.FakeManifestReader{Yaml: `---
          name: app-name
          hosts:
           - host1
           - host2
          domains:
           - example.com
           - example.net
        `}

					p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, repo, NewArgs([]string{"bgd", "app-name"}))

					Expect(b.flow).To(Equal([]string{
						"delete old apps",
						"get current live app",
						"push app-name-new",
						"unmap 1 routes from app-name-new",
						"mapped 4 routes",
						"rename app-name-new to app-name",
					}))

					expectedRoutes := []plugin_models.GetApp_RouteSummary{
						{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
						{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
						{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "example.net"}},
						{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "example.net"}},
					}

					Expect(b.mappedRoutes).To(ConsistOf(expectedRoutes))
				})
			})
			Context("when manifest uses routes", func() {
				It("maps manifest routes", func() {
					b := &BlueGreenDeployFake{liveApp: nil}
					p := CfPlugin{
						Deployer: b,
					}
					repo := &fakes.FakeManifestReader{Yaml: `---
name: app-name
routes:
  - route: host1.something.com
  - route: host2.mine.com
  - route: host3.common.com
`}

					p.Deploy(manifest.CfDomains{DefaultDomain: "example.com", SharedDomains: []string{"common.com"}, PrivateDomains: []string{"mine.com", "something.com"}}, repo, NewArgs([]string{"bgd", "app-name"}))

					Expect(b.flow).To(Equal([]string{
						"delete old apps",
						"get current live app",
						"push app-name-new",
						"unmap 1 routes from app-name-new",
						"mapped 3 routes",
						"rename app-name-new to app-name",
					}))

					expectedRoutes := []plugin_models.GetApp_RouteSummary{
						{Host: "host1", Domain: plugin_models.GetApp_DomainFields{Name: "something.com"}},
						{Host: "host2", Domain: plugin_models.GetApp_DomainFields{Name: "mine.com"}},
						{Host: "host3", Domain: plugin_models.GetApp_DomainFields{Name: "common.com"}},
					}

					Expect(b.mappedRoutes).To(ConsistOf(expectedRoutes))
				})
			})

			Context("when scale parameters are defined", func() {
				It("Uses the scale values", func() {
					b := &BlueGreenDeployFake{liveApp: nil}
					p := CfPlugin{
						Deployer: b,
					}
					repo := &fakes.FakeManifestReader{Yaml: `---
            name: app-name
            memory: 16M
            disk_quota: 500M
            instances: 3
            hosts:
            - host1
            `}
					p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, repo, NewArgs([]string{"bgd", "app-name"}))
					Expect(b.flow).To(Equal([]string{
						"delete old apps",
						"get current live app",
						"push app-name-new",
						"unmap 1 routes from app-name-new",
						"mapped 1 routes",
						"rename app-name-new to app-name",
					}))
					scaleParameters := ScaleParameters{
						Memory:        int64(16),
						DiskQuota:     int64(500),
						InstanceCount: 3,
					}
					Expect(*b.usedScale).To(Equal(scaleParameters))
				})
			})
			Context("when no routes are specified in the manifest", func() {
				It("maps the app name as the only route", func() {
					b := &BlueGreenDeployFake{liveApp: nil}
					p := CfPlugin{
						Deployer: b,
					}
					repo := &fakes.FakeManifestReader{Yaml: `---
						name: app-name
						hosts:
							- host1
					`}

					p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, repo, NewArgs([]string{"bgd", "app-name"}))

					Expect(b.mappedRoutes).To(Equal([]plugin_models.GetApp_RouteSummary{
						{Host: "app-name", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
					}))
				})
			})
		})

		Context("when there is a smoke test defined", func() {
			Context("when it succeeds", func() {
				var (
					b *BlueGreenDeployFake
					p CfPlugin
				)

				BeforeEach(func() {
					b = &BlueGreenDeployFake{liveApp: nil, passSmokeTest: true}
					p = CfPlugin{
						Deployer: b,
					}
				})

				It("calls methods in correct order", func() {
					p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, &fakes.FakeManifestReader{}, NewArgs([]string{"bgd", "app-name", "--smoke-test", "script/smoke-test"}))

					Expect(b.flow).To(Equal([]string{
						"delete old apps",
						"get current live app",
						"push app-name-new",
						"script/smoke-test app-name-new.example.com",
						"unmap 1 routes from app-name-new",
						"mapped 1 routes",
						"rename app-name-new to app-name",
					}))
				})

				It("returns true", func() {
					result := p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, &fakes.FakeManifestReader{}, NewArgs([]string{"bgd", "app-name", "--smoke-test", "script/smoke-test"}))

					Expect(result).To(Equal(true))
				})
			})

			Context("when it fails", func() {
				var (
					b *BlueGreenDeployFake
					p CfPlugin
				)

				BeforeEach(func() {
					b = &BlueGreenDeployFake{liveApp: nil, passSmokeTest: false}
					p = CfPlugin{
						Deployer: b,
					}
				})

				It("calls methods in correct order", func() {
					p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, &fakes.FakeManifestReader{}, NewArgs([]string{"bgd", "app-name", "--smoke-test", "script/smoke-test"}))

					Expect(b.flow).To(Equal([]string{
						"delete old apps",
						"get current live app",
						"push app-name-new",
						"script/smoke-test app-name-new.example.com",
						"unmap 1 routes from app-name-new",
						"rename app-name-new to app-name-failed",
					}))
				})

				It("returns false", func() {
					result := p.Deploy(manifest.CfDomains{DefaultDomain: "example.com"}, &fakes.FakeManifestReader{}, NewArgs([]string{"bgd", "app-name", "--smoke-test", "script/smoke-test"}))

					Expect(result).To(Equal(false))
				})
			})
		})

		Describe("GetScaleFromManifest", func() {
			p := CfPlugin{}
			Context("when the manifest is valid", func() {
				It("returns the scale parameters", func() {
					fakeManifestReader := &fakes.FakeManifestReader{Yaml: `---
            name: app-name
            memory: 16M
            disk_quota: 500M
            hosts:
            - man1
            `,
					}
					actualScale := p.GetScaleFromManifest("app-name", manifest.CfDomains{DefaultDomain: "example.com"}, fakeManifestReader)
					expectedScale := ScaleParameters{Memory: int64(16), DiskQuota: int64(500)}
					Expect(actualScale).To(Equal(expectedScale))
				})
			})
			Context("the manifest is invalid", func() {
				It("returns an empty manifest", func() {
					failingFakeManifestReader := &fakes.FakeManifestReader{Err: errors.New("")}
					actualScale := p.GetScaleFromManifest("app-name", manifest.CfDomains{DefaultDomain: "example.com"}, failingFakeManifestReader)
					expectedScale := ScaleParameters{}
					Expect(actualScale).To(Equal(expectedScale))
				})
			})
		})
	})

	Describe("SharedDomains", func() {
		connection := &pluginfakes.FakeCliConnection{}
		p := CfPlugin{Connection: connection}

		Context("when CF command succeeds", func() {
			It("returns all CF shared domains", func() {
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return []string{`{
 "total_results": 2,
 "total_pages": 1,
 "prev_url": null,
 "next_url": null,
 "resources": [
	{
	   "metadata": {
		  "guid": "75049093-13e9-4520-80a6-2d6fea6542bc",
		  "url": "/v2/shared_domains/75049093-13e9-4520-80a6-2d6fea6542bc",
		  "created_at": "2014-10-20T09:21:39+00:00",
		  "updated_at": null
	   },
	   "entity": {
		  "name": "my.cool.com"
	   }},
	   {
		"metadata": {
		   "guid": "75049093-13e9-4520-80a6-2d6fea6542bc",
		   "url": "/v2/shared_domains/75049093-13e9-4520-80a6-2d6fea6542bc",
		   "created_at": "2014-10-20T09:21:39+00:00",
		   "updated_at": null
		},
		"entity": {
		   "name": "another.com"
		}
	}
 ]
}`}, nil
				}
				domain, _ := p.SharedDomains()
				Expect(domain).To(Equal([]string{"my.cool.com", "another.com"}))
			})
		})

		Context("when CF command fails", func() {
			It("returns error", func() {
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return nil, errors.New("cf curl failed")
				}
				_, err := p.SharedDomains()
				Expect(err).To(MatchError("cf curl failed"))
			})
		})

		Context("when CF command returns invalid JSON", func() {
			It("returns error", func() {
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return []string{`{"resources": { "entity": "foo" }}`}, nil
				}
				_, err := p.SharedDomains()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("PrivateDomains", func() {
		connection := &pluginfakes.FakeCliConnection{}
		p := CfPlugin{Connection: connection}

		Context("when CF command succeeds", func() {
			It("returns all private domains", func() {
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return []string{`{
 "total_results": 2,
 "total_pages": 1,
 "prev_url": null,
 "next_url": null,
 "resources": [
	{
	   "metadata": {
		  "guid": "75049093-13e9-4520-80a6-2d6fea6542bc",
		  "url": "/v2/private_domains/75049093-13e9-4520-80a6-2d6fea6542bc",
		  "created_at": "2014-10-20T09:21:39+00:00",
		  "updated_at": null
	   },
	   "entity": {
		  "name": "eu-gb.mypaas.net"
	   }},
	   {
		"metadata": {
		   "guid": "75049093-13e9-4520-80a6-2d6fea6542bc",
		   "url": "/v2/private_domains/75049093-13e9-4520-80a6-2d6fea6542bc",
		   "created_at": "2014-10-20T09:21:39+00:00",
		   "updated_at": null
		},
		"entity": {
		   "name": "eu-de.mypaas.net"
		}
	}
 ]
}`}, nil
				}
				domain, _ := p.PrivateDomains()
				Expect(domain).To(Equal([]string{"eu-gb.mypaas.net", "eu-de.mypaas.net"}))
			})
		})

		Context("when CF command fails", func() {
			It("returns error", func() {
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return nil, errors.New("cf curl failed")
				}
				_, err := p.PrivateDomains()
				Expect(err).To(MatchError("cf curl failed"))
			})
		})

		Context("when CF command returns invalid JSON", func() {
			It("returns error", func() {
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return []string{`{"resources": { "entity": "foo" }}`}, nil
				}
				_, err := p.PrivateDomains()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Unique list of routes", func() {
		p := CfPlugin{}

		Context("when listA and ListB are empty", func() {
			It("returns an empty list", func() {
				listA := []plugin_models.GetApp_RouteSummary{}
				listB := []plugin_models.GetApp_RouteSummary{}

				Expect(p.UnionRouteLists(listA, listB)).To(BeEmpty())
			})
		})
		Context("when listA is Empty", func() {
			It("returns listB", func() {
				listA := []plugin_models.GetApp_RouteSummary{}
				listB := []plugin_models.GetApp_RouteSummary{{Host: "foo"}}

				Expect(p.UnionRouteLists(listA, listB)).To(Equal(listB))
			})
		})
		Context("when listB is Empty", func() {
			It("returns listA", func() {
				listA := []plugin_models.GetApp_RouteSummary{{Host: "foo"}}
				listB := []plugin_models.GetApp_RouteSummary{}

				Expect(p.UnionRouteLists(listA, listB)).To(ConsistOf(listA))
			})
		})
		Context("when listB and listA contain the same routes", func() {
			It("returns a list equal in contents to listB", func() {
				listA := []plugin_models.GetApp_RouteSummary{{Host: "foo"}}
				listB := []plugin_models.GetApp_RouteSummary{{Host: "foo"}}

				Expect(p.UnionRouteLists(listA, listB)).To(ConsistOf(listA))
			})
		})
		Context("when listB and listA contain different routes", func() {
			It("returns a union of both routes", func() {
				listA := []plugin_models.GetApp_RouteSummary{{Host: "foo"}}
				listB := []plugin_models.GetApp_RouteSummary{{Host: "bar"}}

				Expect(p.UnionRouteLists(listA, listB)).To(ConsistOf(append(listA, listB...)))
			})
		})
		Context("when listA contains some routes not in listB", func() {
			It("returns a union of both routes", func() {
				listA := []plugin_models.GetApp_RouteSummary{{Host: "foo"}, {Host: "bar"}}
				listB := []plugin_models.GetApp_RouteSummary{{Host: "foo"}}

				Expect(p.UnionRouteLists(listA, listB)).To(ConsistOf(listA))
			})
		})
		Context("when listB contains some routes not in listA", func() {
			It("returns a union of both routes", func() {
				listA := []plugin_models.GetApp_RouteSummary{{Host: "foo"}}
				listB := []plugin_models.GetApp_RouteSummary{{Host: "foo"}, {Host: "bar"}}

				Expect(p.UnionRouteLists(listA, listB)).To(ConsistOf(listB))
			})
		})
		Context("when list A and List B contain both shared and non-shared routes", func() {
			It("returns a union of both routes", func() {
				listA := []plugin_models.GetApp_RouteSummary{{Host: "shared"}, {Host: "listAOnly"}}
				listB := []plugin_models.GetApp_RouteSummary{{Host: "shared"}, {Host: "listBOnly"}}

				expectedRoutes := []plugin_models.GetApp_RouteSummary{{Host: "shared"}, {Host: "listAOnly"}, {Host: "listBOnly"}}

				Expect(p.UnionRouteLists(listA, listB)).To(ConsistOf(expectedRoutes))
			})
		})
		Context("when list A is nil", func() {
			It("returns list B", func() {
				listB := []plugin_models.GetApp_RouteSummary{{Host: "foo"}}

				Expect(p.UnionRouteLists(nil, listB)).To(ConsistOf(listB))
			})
		})
		Context("when list B is nil", func() {
			It("returns list A", func() {
				listA := []plugin_models.GetApp_RouteSummary{{Host: "foo"}}

				Expect(p.UnionRouteLists(listA, nil)).To(ConsistOf(listA))
			})
		})
		Context("when list A & list B are nil", func() {
			It("returns an empty Array", func() {
				Expect(p.UnionRouteLists(nil, nil)).To(BeEmpty())
			})
		})
	})

	Describe("FQDN", func() {
		It("returns the fqdn of the route", func() {
			route := plugin_models.GetApp_RouteSummary{Host: "testroute", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}}
			Expect(FQDN(route)).To(Equal("testroute.example.com"))
		})
	})
})

type BlueGreenDeployFake struct {
	flow          []string
	liveApp       *plugin_models.GetAppModel
	appSshEnabled bool
	passSmokeTest bool
	mappedRoutes  []plugin_models.GetApp_RouteSummary
	scale         *ScaleParameters
	usedScale     *ScaleParameters
}

func (p *BlueGreenDeployFake) Setup(connection plugin.CliConnection) {
	p.flow = append(p.flow, "setup")
}

func (p *BlueGreenDeployFake) GetScaleParameters(appName string) (ScaleParameters, error) {
	return ScaleParameters{}, nil
}

func (p *BlueGreenDeployFake) PushNewApp(appName string, route plugin_models.GetApp_RouteSummary,
	manifestPath string, scaleParameters ScaleParameters) {
	p.usedScale = &scaleParameters
	p.flow = append(p.flow, fmt.Sprintf("push %s", appName))
}

func (p *BlueGreenDeployFake) DeleteAllAppsExceptLiveApp(string) {
	p.flow = append(p.flow, "delete old apps")
}

func (p *BlueGreenDeployFake) LiveApp(string) (string, []plugin_models.GetApp_RouteSummary) {
	p.flow = append(p.flow, "get current live app")
	if p.liveApp == nil {
		return "", nil
	} else {
		return p.liveApp.Name, p.liveApp.Routes
	}
}
func (p *BlueGreenDeployFake) RunSmokeTests(script string, fqdn string) bool {
	p.flow = append(p.flow, fmt.Sprintf("%s %s", script, fqdn))
	return p.passSmokeTest
}

func (p *BlueGreenDeployFake) RemapRoutesFromLiveAppToNewApp(liveApp plugin_models.GetAppModel, newApp plugin_models.GetAppModel) {
	p.flow = append(p.flow, fmt.Sprintf("remap routes from %s to %s", liveApp.Name, newApp.Name))
}

func (p *BlueGreenDeployFake) RenameApp(app string, newName string) {
	p.flow = append(p.flow, fmt.Sprintf("rename %s to %s", app, newName))
}

func (p *BlueGreenDeployFake) MapRoutesToApp(appName string, routes ...plugin_models.GetApp_RouteSummary) {
	p.mappedRoutes = routes
	p.flow = append(p.flow, fmt.Sprintf("mapped %d routes", len(routes)))
}

func (p *BlueGreenDeployFake) UnmapRoutesFromApp(oldAppName string, routes ...plugin_models.GetApp_RouteSummary) {
	p.flow = append(p.flow, fmt.Sprintf("unmap %d routes from %s", len(routes), oldAppName))
}

func (p *BlueGreenDeployFake) CheckSshEnablement(app string) bool {
	p.flow = append(p.flow, fmt.Sprintf("check ssh enablement for '%s'", app))
	return strings.Contains(app, "ssh-enabled-app")
}

func (p *BlueGreenDeployFake) SetSshAccess(app string, enableSsh bool) {
	p.flow = append(p.flow, fmt.Sprintf("set ssh enablement for '%s' to '%v'", app, enableSsh))
}
