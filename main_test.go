package main_test

import (
	"errors"
	"fmt"

	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	"github.com/cloudfoundry/cli/plugin"
	"github.com/cloudfoundry/cli/plugin/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BGD Plugin", func() {
	Describe("smoke test script", func() {
		Context("when smoke test flag is not provided", func() {
			It("returns empty string", func() {
				args := []string{"blue-green-deploy", "appName"}
				Expect(ExtractIntegrationTestScript(args)).To(Equal(""))
			})
		})

		Context("when smoke test flag provided", func() {
			It("returns flag value", func() {
				args := []string{"blue-green-deploy", "appName", "--smoke-test=script/test"}
				Expect(ExtractIntegrationTestScript(args)).To(Equal("script/test"))
			})
		})
	})

	Describe("blue green flow", func() {
		Context("when there is a previous live app", func() {
			It("calls methods in correct order", func() {
				b := &BlueGreenDeployFake{liveApp: &Application{Name: "app-name-live"}}
				p := CfPlugin{
					Deployer: b,
				}

				p.Deploy("example.com", &FakeRepo{}, []string{"bgd", "app-name"})

				Expect(b.flow).To(Equal([]string{
					"delete old apps",
					"get current live app",
					"push app-name-new",
					"unmap temporary route from app-name-new",
					"copy routes from app-name-live to app-name-new",
					"rename app-name-live to app-name-old",
					"rename app-name-new to app-name",
					"mapped 1 routes",
					"unmap routes from app-name-live",
				}))
			})
		})

		Context("when there is no previous live app", func() {
			It("calls methods in correct order", func() {
				b := &BlueGreenDeployFake{liveApp: nil}
				p := CfPlugin{
					Deployer: b,
				}

				p.Deploy("example.com", &FakeRepo{}, []string{"bgd", "app-name"})

				Expect(b.flow).To(Equal([]string{
					"delete old apps",
					"get current live app",
					"push app-name-new",
					"unmap temporary route from app-name-new",
					"rename app-name-new to app-name",
					"mapped 1 routes",
				}))
			})
		})

		Context("when app has manifest", func() {
			It("maps manifest routes", func() {
				b := &BlueGreenDeployFake{liveApp: nil}
				p := CfPlugin{
					Deployer: b,
				}
				repo := &FakeRepo{yaml: `---
name: app-name
hosts:
- host1
- host2
domains:
- example.com
- example.net`}

				p.Deploy("example.com", repo, []string{"bgd", "app-name"})

				Expect(b.flow).To(Equal([]string{
					"delete old apps",
					"get current live app",
					"push app-name-new",
					"unmap temporary route from app-name-new",
					"rename app-name-new to app-name",
					"mapped 5 routes",
				}))
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
					p.Deploy("example.com", &FakeRepo{}, []string{"bgd", "app-name", "--smoke-test", "script/smoke-test"})

					Expect(b.flow).To(Equal([]string{
						"delete old apps",
						"get current live app",
						"push app-name-new",
						"script/smoke-test app-name-new.example.com",
						"unmap temporary route from app-name-new",
						"rename app-name-new to app-name",
						"mapped 1 routes",
					}))
				})

				It("returns true", func() {
					result := p.Deploy("example.com", &FakeRepo{}, []string{"bgd", "app-name", "--smoke-test", "script/smoke-test"})

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
					p.Deploy("example.com", &FakeRepo{}, []string{"bgd", "app-name", "--smoke-test", "script/smoke-test"})

					Expect(b.flow).To(Equal([]string{
						"delete old apps",
						"get current live app",
						"push app-name-new",
						"script/smoke-test app-name-new.example.com",
						"unmap temporary route from app-name-new",
						"rename app-name-new to app-name-failed",
					}))
				})

				It("returns false", func() {
					result := p.Deploy("example.com", &FakeRepo{}, []string{"bgd", "app-name", "--smoke-test", "script/smoke-test"})

					Expect(result).To(Equal(false))
				})
			})
		})

		Describe("DefaultCfDomain", func() {
			connection := &fakes.FakeCliConnection{}
			p := CfPlugin{Connection: connection}

			Context("when CF command succeeds", func() {
				It("returns CF default shared domain", func() {
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
              "name": "eu-gb.mybluemix.net"
           }
        }
     ]
  }`}, nil
					}
					domain, _ := p.DefaultCfDomain()
					Expect(domain).To(Equal("eu-gb.mybluemix.net"))
				})
			})

			Context("when CF command fails", func() {
				It("returns error", func() {
					connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
						return nil, errors.New("cf curl failed")
					}
					_, err := p.DefaultCfDomain()
					Expect(err).To(MatchError("cf curl failed"))
				})
			})

			Context("when CF command returns invalid JSON", func() {
				It("returns error", func() {
					connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
						return []string{`{"resources": { "entity": "foo" }}`}, nil
					}
					_, err := p.DefaultCfDomain()
					Expect(err).To(HaveOccurred())
				})
			})
		})

	})
})

type BlueGreenDeployFake struct {
	flow []string
	AppLister
	liveApp       *Application
	passSmokeTest bool
}

func (p *BlueGreenDeployFake) Setup(connection plugin.CliConnection) {
	p.flow = append(p.flow, "setup")
}

func (p *BlueGreenDeployFake) PushNewApp(app *Application) {
	p.flow = append(p.flow, fmt.Sprintf("push %s", app.Name))
}

func (p *BlueGreenDeployFake) DeleteAllAppsExceptLiveApp(string) {
	p.flow = append(p.flow, "delete old apps")
}

func (p *BlueGreenDeployFake) LiveApp(string) *Application {
	p.flow = append(p.flow, "get current live app")
	return p.liveApp
}
func (p *BlueGreenDeployFake) RunSmokeTests(script string, fqdn string) bool {
	p.flow = append(p.flow, fmt.Sprintf("%s %s", script, fqdn))
	return p.passSmokeTest
}

func (p *BlueGreenDeployFake) RemapRoutesFromLiveAppToNewApp(liveApp Application, newApp Application) {
	p.flow = append(p.flow, fmt.Sprintf("remap routes from %s to %s", liveApp.Name, newApp.Name))
}

func (p *BlueGreenDeployFake) UnmapTemporaryRouteFromNewApp(newApp Application) {
	p.flow = append(p.flow, fmt.Sprintf("unmap temporary route from %s", newApp.Name))
}

func (p *BlueGreenDeployFake) RenameApp(app *Application, newName string) {
	p.flow = append(p.flow, fmt.Sprintf("rename %s to %s", app.Name, newName))
}

func (p *BlueGreenDeployFake) MapAllRoutes(app *Application) {
	p.flow = append(p.flow, fmt.Sprintf("mapped %d routes", len(app.Routes)))
}

func (p *BlueGreenDeployFake) CopyLiveAppRoutesToNewApp(liveApp Application, newApp Application) {
	p.flow = append(p.flow, fmt.Sprintf("copy routes from %s to %s", liveApp.Name, newApp.Name))
}

func (p *BlueGreenDeployFake) UnmapRoutesFromOldApp(oldApp *Application) {
	p.flow = append(p.flow, fmt.Sprintf("unmap routes from %s", oldApp.Name))
}
