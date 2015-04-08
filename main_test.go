package main_test

import (
	"errors"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	plugin "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"

	"github.com/cloudfoundry/cli/plugin/fakes"
)

var _ = Describe("BGD Plugin", func() {
	Describe("the DeleteOldAppVersions function", func() {
		Context("when there is an old version deployed", func() {
			It("deletes the old version", func() {
				connection := &fakes.FakeCliConnection{}
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return []string{
							"{\"Apps\":[{\"Name\":\"app-name-20150326110000-old\"}]}",
						},
						nil
				}

				p := plugin.BlueGreenDeployPlugin{Connection: connection}
				p.DeleteOldAppVersions("app-name")

				Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
					To(Equal("delete app-name-20150326110000-old -f -r"))
			})
		})

		Context("when there is no old version deployed", func() {
			It("deletes nothing", func() {
				connection := &fakes.FakeCliConnection{}
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return []string{
							"{\"Apps\":[]}",
						},
						nil
				}

				p := plugin.BlueGreenDeployPlugin{Connection: connection}
				p.DeleteOldAppVersions("app-name")

				Expect(connection.CliCommandCallCount()).To(Equal(0))
			})
		})

		Context("when the list of apps in the current space can not be fetched", func() {
			It("returns an error", func() {
				connection := &fakes.FakeCliConnection{}
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return nil, errors.New("Failed retrieving app names")
				}

				p := plugin.BlueGreenDeployPlugin{Connection: connection}
				err := p.DeleteOldAppVersions("app-name")

				Expect(err).To(MatchError("Failed retrieving app names"))
			})
		})

		Context("when the deletion of an old app fails", func() {
			It("returns an error", func() {
				connection := &fakes.FakeCliConnection{}
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return []string{
							"{\"Apps\":[{\"Name\":\"app-name-20150326110000-old\"}]}",
						},
						nil
				}
				connection.CliCommandStub = func(args ...string) ([]string, error) {
					return nil, errors.New("failed to delete app")
				}

				p := plugin.BlueGreenDeployPlugin{Connection: connection}
				err := p.DeleteOldAppVersions("app-name")

				Expect(err).To(HaveOccurred())
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
})
