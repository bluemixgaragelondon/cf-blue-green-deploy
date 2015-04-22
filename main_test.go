package main_test

import (
	"fmt"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/cloudfoundry/cli/plugin/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"
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

	Describe("smoke test runner", func() {
		It("returns stdout", func() {
			out, _ := RunIntegrationTestScript("test/support/smoke-test-script", "app.mybluemix.net")
			Expect(out).To(ContainSubstring("STDOUT"))
		})

		It("returns stderr", func() {
			out, _ := RunIntegrationTestScript("test/support/smoke-test-script", "app.mybluemix.net")
			Expect(out).To(ContainSubstring("STDERR"))
		})

		It("passes app FQDN as first argument", func() {
			out, _ := RunIntegrationTestScript("test/support/smoke-test-script", "app.mybluemix.net")
			Expect(out).To(ContainSubstring("App FQDN is: app.mybluemix.net"))
		})

		Context("when script doesn't exist", func() {
			It("fails with useful error", func() {
				_, err := RunIntegrationTestScript("inexistent-smoke-test-script", "app.mybluemix.net")
				Expect(err.Error()).To(ContainSubstring("executable file not found"))
			})
		})

		Context("when script isn't executable", func() {
			It("fails with useful error", func() {
				_, err := RunIntegrationTestScript("test/support/nonexec-smoke-test-script", "app.mybluemix.net")
				Expect(err.Error()).To(ContainSubstring("permission denied"))
			})
		})
	})

	Describe("app name generator", func() {
		generated := GenerateAppName("foo")

		It("uses the passed name with -new appended", func() {
			Expect(generated).To(Equal("foo-new"))
		})
	})

	Describe("blue green flow", func() {
		Context("when there is a previous live app", func() {
			It("calls methods in correct order", func() {
				b := &BlueGreenDeployFake{liveApp: &Application{Name: "app-name-live"}}
				p := CfPlugin{
					Deployer: b,
				}

				p.Run(&fakes.FakeCliConnection{}, []string{"bgd", "app-name"})

				Expect(b.flow).To(Equal([]string{
					"setup",
					"delete old apps",
					"get current live app",
					"push app-name",
					"remap routes from app-name-live to app-name-new",
					"unmap temporary route from app-name-new",
					"update app-name-live to old and app-name-new to live",
				}))
			})
		})

		Context("when there is no previous live app", func() {
			It("calls methods in correct order", func() {
				b := &BlueGreenDeployFake{liveApp: nil}
				p := CfPlugin{
					Deployer: b,
				}

				p.Run(&fakes.FakeCliConnection{}, []string{"bgd", "app-name"})

				Expect(b.flow).To(Equal([]string{
					"setup",
					"delete old apps",
					"get current live app",
					"push app-name",
				}))
			})
		})

		Context("when there is a smoke test defined", func() {
			It("calls methods in correct order", func() {
				b := &BlueGreenDeployFake{liveApp: nil}
				p := CfPlugin{
					Deployer: b,
				}

				p.Run(&fakes.FakeCliConnection{}, []string{"bgd", "app-name", "--smoke-test", "script/smoke-test"})

				Expect(b.flow).To(Equal([]string{
					"setup",
					"delete old apps",
					"get current live app",
					"push app-name",
					"script/smoke-test app-name-new.example.com",
				}))
			})
		})
	})
})

type BlueGreenDeployFake struct {
	flow []string
	AppLister
	liveApp *Application
}

func (p *BlueGreenDeployFake) Setup(connection plugin.CliConnection) {
	p.flow = append(p.flow, "setup")
}

func (p *BlueGreenDeployFake) PushNewApp(appName string) Application {
	p.flow = append(p.flow, fmt.Sprintf("push %s", appName))
	return Application{Name: "app-name-new", Routes: []Route{{Host: "app-name-new", Domain: Domain{Name: "example.com"}}}}
}

func (p *BlueGreenDeployFake) DeleteAllAppsExceptLiveApp(string) {
	p.flow = append(p.flow, "delete old apps")
}

func (p *BlueGreenDeployFake) LiveApp(string) *Application {
	p.flow = append(p.flow, "get current live app")
	return p.liveApp
}
func (p *BlueGreenDeployFake) RunSmokeTests(script string, fqdn string) {
	p.flow = append(p.flow, fmt.Sprintf("%s %s", script, fqdn))
}

func (p *BlueGreenDeployFake) RemapRoutesFromLiveAppToNewApp(liveApp Application, newApp Application) {
	p.flow = append(p.flow, fmt.Sprintf("remap routes from %s to %s", liveApp.Name, newApp.Name))
}

func (p *BlueGreenDeployFake) UnmapTemporaryRouteFromNewApp(newApp Application) {
	p.flow = append(p.flow, fmt.Sprintf("unmap temporary route from %s", newApp.Name))
}

func (p *BlueGreenDeployFake) UpdateAppNames(oldApp, newApp *Application) {
	p.flow = append(p.flow, fmt.Sprintf("update %s to old and %s to live", oldApp.Name, newApp.Name))
}
