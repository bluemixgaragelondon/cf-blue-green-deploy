package main_test

import (
	"errors"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	plugin "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"

	"github.com/cloudfoundry/cli/plugin/fakes"
)

var _ = Describe("Main", func() {
	Describe("Plugin", func() {
		Describe("blue-green-deploy", func() {
			It("exists", func() {
				connection := &fakes.FakeCliConnection{}
				connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
					return []string{
							"[{\"Name\":\"wqe\"}]",
						},
						nil
				}
				p := plugin.BlueGreenDeployPlugin{}
				p.Run(connection, []string{"blue-green-deploy", "appname"})
			})

			Describe("OldAppVersionList", func() {
				It("returns list of application names", func() {
					connection := &fakes.FakeCliConnection{}
					connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
						return []string{
								"[{\"Name\":\"app-name-20150326110000-old\"}]",
							},
							nil
					}
					p := plugin.BlueGreenDeployPlugin{Connection: connection}
					appList, _ := p.OldAppVersionList("app-name")

					Expect(appList).To(Equal([]plugin.Application{{Name: "app-name-20150326110000-old"}}))
				})

				Context("when cli command fails", func() {
					It("returns error", func() {
						connection := &fakes.FakeCliConnection{}
						connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
							return nil, errors.New("Failed retrieving app names")
						}
						p := plugin.BlueGreenDeployPlugin{Connection: connection}
						_, err := p.OldAppVersionList("app-name")
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})

		Describe("DeleteApps", func() {
			Context("when app fails to delete", func() {
				It("returns error", func() {
					connection := &fakes.FakeCliConnection{}
					connection.CliCommandStub = func(args ...string) ([]string, error) {
						return nil, errors.New("Failed deleting app")
					}
					p := plugin.BlueGreenDeployPlugin{Connection: connection}
					Expect(p.DeleteApps([]plugin.Application{{Name: "app-name"}})).To(MatchError("Failed deleting app"))
				})
			})

			Context("when no app delete fails", func() {
				It("deletes all apps and mapped routes in list", func() {
					connection := &fakes.FakeCliConnection{}
					p := plugin.BlueGreenDeployPlugin{Connection: connection}
					p.DeleteApps([]plugin.Application{{Name: "app1"}, {Name: "app2"}})

					Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).To(Equal("delete app1 -f -r"))
					Expect(strings.Join(connection.CliCommandArgsForCall(1), " ")).To(Equal("delete app2 -f -r"))
				})
			})
		})

		Describe("DeleteOldAppVersions", func() {
			Context("when getting old app versions fails", func() {
				It("returns error", func() {
					connection := &fakes.FakeCliConnection{}
					connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
						return nil, errors.New("Failed retrieving app names")
					}
					p := plugin.BlueGreenDeployPlugin{Connection: connection}
					Expect(p.DeleteOldAppVersions("app-name")).To(MatchError("Failed retrieving app names"))
				})
			})

			Context("when getting old app versions succeeds", func() {
				It("deletes all old app versions", func() {
					connection := &fakes.FakeCliConnection{}
					connection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
						return []string{
								"[{\"Name\":\"app-20150326120000\"},{\"Name\":\"app-20150326110000-old\"}]",
							},
							nil
					}
					p := plugin.BlueGreenDeployPlugin{Connection: connection}
					p.DeleteOldAppVersions("app")
					Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).To(Equal("delete app-20150326110000-old -f -r"))
				})
			})
		})
	})
})
