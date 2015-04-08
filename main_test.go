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

	Describe("old app filter", func() {
		Context("when there are 2 old versions and 1 non-old version", func() {
			appList := []plugin.Application{
				{Name: "foo-20150408114041-old"},
				{Name: "foo-20141234567348-old"},
				{Name: "foo-20163453473845"},
				{Name: "bar-foo-20141234567348-old"},
				{Name: "foo-20141234567348-older"},
			}

			It("returns all apps that have the same name, with a valid timestamp and -old suffix", func() {
				Expect(plugin.FilterOldApps("foo", appList)).To(ContainElement(appList[0]))
				Expect(plugin.FilterOldApps("foo", appList)).To(ContainElement(appList[1]))
			})

			It("doesn't return any apps that don't have a -old suffix", func() {
				Expect(plugin.FilterOldApps("foo", appList)).ToNot(ContainElement(appList[2]))
			})

			It("doesn't return elements that have an additional prefix before the app name", func() {
				Expect(plugin.FilterOldApps("foo", appList)).ToNot(ContainElement(appList[3]))
			})

			It("doesn't return elements that have an additional suffix after -old", func() {
				Expect(plugin.FilterOldApps("foo", appList)).ToNot(ContainElement(appList[4]))
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
