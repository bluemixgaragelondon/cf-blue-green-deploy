package main_test

import (
	"bytes"
	"errors"
	"strings"

	"code.cloudfoundry.org/cli/plugin/pluginfakes"
	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BlueGreenDeploy", func() {
	var (
		bgdExitsWithErrors []error
		bgdOut             *bytes.Buffer
		connection         *pluginfakes.FakeCliConnection
		p                  BlueGreenDeploy
		testErrorFunc      func(message string, err error)
	)

	BeforeEach(func() {
		bgdExitsWithErrors = []error{}
		testErrorFunc = func(message string, err error) {
			bgdExitsWithErrors = append(bgdExitsWithErrors, err)
		}
		bgdOut = &bytes.Buffer{}

		connection = &pluginfakes.FakeCliConnection{}
		p = BlueGreenDeploy{Connection: connection, ErrorFunc: testErrorFunc, Out: bgdOut}
	})

	Describe("maps routes", func() {
		var (
			manifestApp plugin_models.GetAppModel
		)

		BeforeEach(func() {
			manifestApp = plugin_models.GetAppModel{
				Name: "new",
				Routes: []plugin_models.GetApp_RouteSummary{
					{Host: "host", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
					{Host: "host", Domain: plugin_models.GetApp_DomainFields{Name: "example.net"}},
				},
			}
		})

		It("maps all", func() {
			p.MapRoutesToApp(manifestApp.Name, manifestApp.Routes...)

			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(Equal([]string{
				"map-route new example.com -n host",
				"map-route new example.net -n host",
			}))
		})
	})

	Describe("remove routes from old app", func() {
		var (
			oldApp plugin_models.GetAppModel
		)

		BeforeEach(func() {
			oldApp = plugin_models.GetAppModel{
				Name: "old",
				Routes: []plugin_models.GetApp_RouteSummary{
					{Host: "live", Domain: plugin_models.GetApp_DomainFields{Name: "mybluemix.net"}},
					{Host: "live", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}},
				},
			}
		})

		It("unmaps all routes from the old app", func() {
			p.UnmapRoutesFromApp(oldApp.Name, oldApp.Routes...)

			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(Equal([]string{
				"unmap-route old mybluemix.net -n live",
				"unmap-route old example.com -n live",
			}))
		})
	})

	Describe("renaming an app", func() {
		var app string

		BeforeEach(func() {
			app = "foo"
		})

		It("renames the app", func() {
			p.RenameApp(app, "bar")
			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(ContainElement(
				"rename foo bar",
			))
		})

		Context("when renaming the app fails", func() {
			It("calls the error callback", func() {
				connection.CliCommandStub = func(args ...string) ([]string, error) {
					return nil, errors.New("failed to rename app")
				}
				p.RenameApp(app, "bar")

				Expect(bgdExitsWithErrors[0]).To(MatchError("failed to rename app"))
			})
		})
	})

	Describe("delete old apps", func() {
		var (
			apps []plugin_models.GetAppsModel
		)

		Context("with live and old apps", func() {
			BeforeEach(func() {
				apps = []plugin_models.GetAppsModel{
					{Name: "app-name-old"},
					{Name: "app-name"},
				}
				connection.GetAppsReturns(apps, nil)
			})

			It("only deletes the old apps", func() {
				p.DeleteAllAppsExceptLiveApp("app-name")
				cfCommands := getAllCfCommands(connection)

				Expect(cfCommands).To(Equal([]string{
					"delete app-name-old -f -r",
				}))
			})

			Context("when the deletion of an app fails", func() {
				BeforeEach(func() {
					connection.CliCommandStub = func(args ...string) ([]string, error) {
						return nil, errors.New("failed to delete app")
					}
				})

				It("returns an error", func() {
					p.DeleteAllAppsExceptLiveApp("app-name")
					Expect(bgdExitsWithErrors[0]).To(HaveOccurred())
				})
			})
		})

		Context("with live and failed apps", func() {
			BeforeEach(func() {
				apps = []plugin_models.GetAppsModel{
					{Name: "app-name-failed"},
					{Name: "app-name"},
				}
				connection.GetAppsReturns(apps, nil)
			})

			It("only deletes the failed apps", func() {
				p.DeleteAllAppsExceptLiveApp("app-name")
				cfCommands := getAllCfCommands(connection)

				Expect(cfCommands).To(Equal([]string{
					"delete app-name-failed -f -r",
				}))
			})
		})

		Context("with live and new apps", func() {
			BeforeEach(func() {
				apps = []plugin_models.GetAppsModel{
					{Name: "app-name-new"},
					{Name: "app-name"},
				}
				connection.GetAppsReturns(apps, nil)
			})

			It("only deletes the new apps", func() {
				p.DeleteAllAppsExceptLiveApp("app-name")
				cfCommands := getAllCfCommands(connection)

				Expect(cfCommands).To(Equal([]string{
					"delete app-name-new -f -r",
				}))
			})
		})

		Context("when there is no old version deployed", func() {
			BeforeEach(func() {
				apps = []plugin_models.GetAppsModel{
					{Name: "app-name"},
				}
				connection.GetAppsReturns(apps, nil)
			})

			It("succeeds", func() {
				p.DeleteAllAppsExceptLiveApp("app-name")
				Expect(bgdExitsWithErrors).To(HaveLen(0))
			})

			It("deletes nothing", func() {
				p.DeleteAllAppsExceptLiveApp("app-name")
				Expect(connection.CliCommandCallCount()).To(Equal(0))
			})
		})
	})

	Describe("deleting apps", func() {
		Context("when there is an old version deployed", func() {
			apps := []plugin_models.GetAppsModel{
				{Name: "app-name-old"},
				{Name: "app-name-old"},
			}

			It("deletes the apps", func() {
				p.DeleteAppVersions(apps)
				cfCommands := getAllCfCommands(connection)

				Expect(cfCommands).To(Equal([]string{
					"delete app-name-old -f -r",
					"delete app-name-old -f -r",
				}))
			})

			Context("when the deletion of an app fails", func() {
				BeforeEach(func() {
					connection.CliCommandStub = func(args ...string) ([]string, error) {
						return nil, errors.New("failed to delete app")
					}
				})

				It("returns an error", func() {
					p.DeleteAppVersions(apps)
					Expect(bgdExitsWithErrors[0]).To(HaveOccurred())
				})
			})
		})

		Context("when there is no old version deployed", func() {
			apps := []plugin_models.GetAppsModel{}

			It("succeeds", func() {
				p.DeleteAppVersions(apps)
				Expect(bgdExitsWithErrors).To(HaveLen(0))
			})

			It("deletes nothing", func() {
				p.DeleteAppVersions(apps)
				Expect(connection.CliCommandCallCount()).To(Equal(0))
			})
		})
	})

	Describe("pushing a new app", func() {
		newApp := "app-name-new"
		newRoute := plugin_models.GetApp_RouteSummary{Host: newApp, Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}}

		It("pushes an app with new appended to its name", func() {
			p.PushNewApp(newApp, newRoute)

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`^push app-name-new`))
		})

		It("uses the generated name for the route", func() {
			p.PushNewApp(newApp, newRoute)

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`-n app-name-new`))
		})

		It("pushes with the default cf domain", func() {
			p.PushNewApp(newApp, newRoute)

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`-d example.com`))
		})

		It("pushes with the specified manifest, if present in deployer", func() {
			manifestPath := "./manifest-tst.yml"
			p.PushNewApp(newApp, newRoute, manifestPath)

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`-f ./manifest-tst.yml`))
		})

		It("pushes without a manifest arg, if no manifest in deployer", func() {
			p.PushNewApp(newApp, newRoute)

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(Not(MatchRegexp(`-f `)))
		})

		Context("when the push fails", func() {
			BeforeEach(func() {
				connection.CliCommandStub = func(args ...string) ([]string, error) {
					return nil, errors.New("failed to push app")
				}
			})

			It("returns an error", func() {
				p.PushNewApp(newApp, newRoute)

				Expect(bgdExitsWithErrors[0]).To(MatchError("failed to push app"))
			})
		})
	})

	Describe("live app", func() {
		liveApp := plugin_models.GetAppModel{Name: "app-name"}

		Context("with live and old apps", func() {
			It("returns the live app", func() {
				connection.GetAppReturns(liveApp, nil)

				name, _ := p.LiveApp("app-name")
				Expect(name).To(Equal(liveApp.Name))
			})
		})

		Context("with no apps", func() {
			It("returns an empty app name", func() {
				connection.GetAppReturns(plugin_models.GetAppModel{}, errors.New("an error for no apps"))

				name, _ := p.LiveApp("app-name")
				Expect(name).To(BeEmpty())
			})
		})
	})

	Describe("app filter", func() {
		Context("when there are 2 old versions and 1 non-old version", func() {
			var (
				appList []plugin_models.GetAppsModel
				oldApps []plugin_models.GetAppsModel
			)

			BeforeEach(func() {
				appList = []plugin_models.GetAppsModel{
					{Name: "foo-old"},
					{Name: "foo-old"},
					{Name: "foo"},
					{Name: "bar-foo-old"},
					{Name: "foo-older"},
				}
				oldApps = p.GetOldApps("foo", appList)
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

	Describe("smoke test runner", func() {
		It("returns stdout", func() {
			_ = p.RunSmokeTests("test/support/smoke-test-script", "app.mybluemix.net")
			Expect(bgdOut.String()).To(ContainSubstring("STDOUT"))
		})

		It("returns stderr", func() {
			_ = p.RunSmokeTests("test/support/smoke-test-script", "app.mybluemix.net")
			Expect(bgdOut.String()).To(ContainSubstring("STDERR"))
		})

		It("passes app FQDN as first argument", func() {
			_ = p.RunSmokeTests("test/support/smoke-test-script", "app.mybluemix.net")
			Expect(bgdOut.String()).To(ContainSubstring("App FQDN is: app.mybluemix.net"))
		})

		Context("when script doesn't exist", func() {
			It("fails with useful error", func() {
				_ = p.RunSmokeTests("inexistent-smoke-test-script", "app.mybluemix.net")
				Expect(bgdExitsWithErrors[0].Error()).To(ContainSubstring("executable file not found"))
			})
		})

		Context("when script isn't executable", func() {
			It("fails with useful error", func() {
				_ = p.RunSmokeTests("test/support/nonexec-smoke-test-script", "app.mybluemix.net")
				Expect(bgdExitsWithErrors[0].Error()).To(ContainSubstring("permission denied"))
			})
		})

		Context("when script fails", func() {
			var passSmokeTest bool

			BeforeEach(func() {
				passSmokeTest = p.RunSmokeTests("test/support/smoke-test-script", "FORCE-SMOKE-TEST-FAILURE")
			})

			It("returns false", func() {
				Expect(passSmokeTest).To(Equal(false))
			})

			It("doesn't fail", func() {
				Expect(bgdExitsWithErrors).To(HaveLen(0))
			})
		})
	})

})

func getAllCfCommands(connection *pluginfakes.FakeCliConnection) (commands []string) {
	commands = []string{}
	for i := 0; i < connection.CliCommandCallCount(); i++ {
		args := connection.CliCommandArgsForCall(i)
		commands = append(commands, strings.Join(args, " "))
	}
	return
}
