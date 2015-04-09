package main_test

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	plugin "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"

	"github.com/cloudfoundry/cli/plugin/fakes"
)

var _ = Describe("BGD Plugin", func() {
	Describe("the DeleteOldAppVersions function", func() {
		Context("when there is an old version deployed", func() {
			var connection *fakes.FakeCliConnection
			apps := []plugin.Application{{Name: "app-name-20150326110000-old"}}

			BeforeEach(func() {
				connection = &fakes.FakeCliConnection{}
			})

			It("deletes the old version", func() {
				p := plugin.BlueGreenDeployPlugin{Connection: connection}
				p.DeleteOldAppVersions("app-name", apps)

				Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
					To(Equal("delete app-name-20150326110000-old -f -r"))
			})

			Context("when the deletion of an old app fails", func() {
				BeforeEach(func() {
					connection.CliCommandStub = func(args ...string) ([]string, error) {
						return nil, errors.New("failed to delete app")
					}
				})

				It("returns an error", func() {
					p := plugin.BlueGreenDeployPlugin{Connection: connection}
					err := p.DeleteOldAppVersions("app-name", apps)

					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when there is no old version deployed", func() {
			var connection *fakes.FakeCliConnection
			apps := []plugin.Application{}

			BeforeEach(func() {
				connection = &fakes.FakeCliConnection{}
			})

			It("succeeds", func() {
				p := plugin.BlueGreenDeployPlugin{Connection: connection}
				err := p.DeleteOldAppVersions("app-name", apps)
				Expect(err).ToNot(HaveOccurred())
			})

			It("deletes nothing", func() {
				p := plugin.BlueGreenDeployPlugin{Connection: connection}
				p.DeleteOldAppVersions("app-name", apps)
				Expect(connection.CliCommandCallCount()).To(Equal(0))
			})
		})
	})

	Describe("the PushNewAppVersion function", func() {
		It("pushes an app with the timestamp appended to its name", func() {
			connection := &fakes.FakeCliConnection{}

			p := plugin.BlueGreenDeployPlugin{Connection: connection}
			p.PushNewAppVersion("app-name")

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`^push app-name-\d{14}$`))
		})

		It("returns the new app name", func() {
			connection := &fakes.FakeCliConnection{}
			p := plugin.BlueGreenDeployPlugin{Connection: connection}
			newAppName, _ := p.PushNewAppVersion("app-name")

			Expect(newAppName).To(MatchRegexp(`^app-name-\d{14}$`))
		})

		Context("when the push fails", func() {
			var connection *fakes.FakeCliConnection

			BeforeEach(func() {
				connection = &fakes.FakeCliConnection{}
				connection.CliCommandStub = func(args ...string) ([]string, error) {
					return nil, errors.New("failed to push app")
				}
			})

			It("returns an error", func() {
				p := plugin.BlueGreenDeployPlugin{Connection: connection}
				_, err := p.PushNewAppVersion("app-name")

				Expect(err).To(MatchError("failed to push app"))
			})
		})
	})

	Describe("the MapRoutesFromPreviousApp function", func() {
		Context("when there was an app previously pushed", func() {
			previousApp := plugin.Application{
				Name: "foo",
				Routes: []plugin.Route{
					{Host: "foo", Domain: plugin.Domain{Name: "example.com"}},
					{Host: "bar", Domain: plugin.Domain{Name: "mybluemix.net"}},
				},
			}

			It("maps the routes of the previous app to the new app", func() {
				connection := &fakes.FakeCliConnection{}
				p := plugin.BlueGreenDeployPlugin{Connection: connection}
				p.MapRoutesFromPreviousApp("foo-12345", previousApp)

				Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
					To(Equal("map-route foo-12345 example.com -n foo"))
				Expect(strings.Join(connection.CliCommandArgsForCall(1), " ")).
					To(Equal("map-route foo-12345 mybluemix.net -n bar"))
			})
		})
	})

	Describe("the UnmapAllRoutes function", func() {
		It("unmaps all routes from the app", func() {
			app := plugin.Application{
				Name: "foo",
				Routes: []plugin.Route{
					{Host: "foo", Domain: plugin.Domain{Name: "example.com"}},
					{Host: "bar", Domain: plugin.Domain{Name: "mybluemix.net"}},
				},
			}

			connection := &fakes.FakeCliConnection{}
			p := plugin.BlueGreenDeployPlugin{Connection: connection}
			p.UnmapAllRoutes(app)

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(Equal("unmap-route foo example.com -n foo"))
			Expect(strings.Join(connection.CliCommandArgsForCall(1), " ")).
				To(Equal("unmap-route foo mybluemix.net -n bar"))
		})
	})

	Describe("integration test script", func() {
		Context("when integration test flag is not provided", func() {
			It("returns empty string", func() {
				args := []string{"blue-green-deploy", "appName"}
				Expect(plugin.ExtractIntegrationTestScript(args)).To(Equal(""))
			})
		})

		Context("when integration test flag provided", func() {
			It("returns flag value", func() {
				args := []string{"blue-green-deploy", "appName", "--integration-test=script/test"}
				Expect(plugin.ExtractIntegrationTestScript(args)).To(Equal("script/test"))
			})
		})
	})

	Describe("integration test runner", func() {
		It("returns stdout", func() {
			out, _ := plugin.RunIntegrationTestScript("test/support/integration-test-script", "app.mybluemix.net")
			Expect(out).To(ContainSubstring("STDOUT"))
		})

		It("returns stderr", func() {
			out, _ := plugin.RunIntegrationTestScript("test/support/integration-test-script", "app.mybluemix.net")
			Expect(out).To(ContainSubstring("STDERR"))
		})

		It("passes app FQDN as first argument", func() {
			out, _ := plugin.RunIntegrationTestScript("test/support/integration-test-script", "app.mybluemix.net")
			Expect(out).To(ContainSubstring("App FQDN is: app.mybluemix.net"))
		})

		Context("when script doesn't exist", func() {
			It("fails with useful error", func() {
				_, err := plugin.RunIntegrationTestScript("inexistent-integration-test-script", "app.mybluemix.net")
				Expect(err.Error()).To(ContainSubstring("executable file not found"))
			})
		})

		Context("when script isn't executable", func() {
			It("fails with useful error", func() {
				_, err := plugin.RunIntegrationTestScript("test/support/nonexec-integration-test-script", "app.mybluemix.net")
				Expect(err.Error()).To(ContainSubstring("permission denied"))
			})
		})
	})

	Describe("app filter", func() {
		Context("when there are 2 old versions and 1 non-old version", func() {
			var (
				appList    []plugin.Application
				currentApp *plugin.Application
				oldApps    []plugin.Application
			)

			BeforeEach(func() {
				appList = []plugin.Application{
					{Name: "foo-20150408114041-old"},
					{Name: "foo-20141234567348-old"},
					{Name: "foo-20163453473845"},
					{Name: "bar-foo-20141234567348-old"},
					{Name: "foo-20141234567348-older"},
				}
				currentApp, oldApps = plugin.FilterApps("foo", appList)
			})

			Describe("current app", func() {
				Context("when there is no current live app", func() {
					It("returns an empty struct", func() {
						app, _ := plugin.FilterApps("bar", appList)
						Expect(app).To(BeNil())
					})
				})

				Context("when there is a current live app", func() {
					It("returns the current live app", func() {
						Expect(*currentApp).To(Equal(appList[2]))
					})
				})
			})

			Describe("old app list", func() {
				It("returns all apps that have the same name, with a valid timestamp and -old suffix", func() {
					Expect(oldApps).To(ContainElement(appList[0]))
					Expect(oldApps).To(ContainElement(appList[1]))
				})

				It("doesn't return any apps that don't have a -old suffix", func() {
					Expect(oldApps).ToNot(ContainElement(appList[2]))
				})

				It("doesn't return elements that have an additional prefix before the app name", func() {
					Expect(oldApps).ToNot(ContainElement(appList[3]))
				})

				It("doesn't return elements that have an additional suffix after -old", func() {
					Expect(oldApps).ToNot(ContainElement(appList[4]))
				})
			})
		})
	})

	Describe("app name generator", func() {
		generated := plugin.GenerateAppName("foo")

		It("uses the passed name as a prefix", func() {
			Expect(generated).To(HavePrefix("foo"))
		})

		It("uses a timestamp as a suffix", func() {
			now, _ := strconv.Atoi(time.Now().Format("20060102150405"))
			genTimestamp, _ := strconv.Atoi(regexp.MustCompile(`\d{14}`).FindString(generated))

			Expect(now - genTimestamp).To(BeNumerically("<", 5))
		})
	})
})
