package main_test

import (
	"bytes"
	"errors"
	"strings"

	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	"github.com/cloudfoundry/cli/plugin/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BlueGreenDeploy", func() {
	var (
		bgdExitsWithErrors []error
		bgdOut             *bytes.Buffer
		connection         *fakes.FakeCliConnection
		p                  BlueGreenDeploy
		testErrorFunc      func(message string, err error)
	)

	BeforeEach(func() {
		bgdExitsWithErrors = []error{}
		testErrorFunc = func(message string, err error) {
			bgdExitsWithErrors = append(bgdExitsWithErrors, err)
		}
		bgdOut = &bytes.Buffer{}

		connection = &fakes.FakeCliConnection{}
		p = BlueGreenDeploy{Connection: connection, ErrorFunc: testErrorFunc, Out: bgdOut}
	})

	Describe("map all routes", func() {
		var (
			manifestApp Application
		)

		BeforeEach(func() {
			manifestApp = Application{
				Name: "new",
				Routes: []Route{
					{Host: "host", Domain: Domain{Name: "example.com"}},
					{Host: "host", Domain: Domain{Name: "example.net"}},
				},
			}
		})

		It("maps all", func() {
			p.MapAllRoutes(manifestApp.Name, manifestApp.Routes)

			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(Equal([]string{
				"map-route new example.com -n host",
				"map-route new example.net -n host",
			}))
		})
	})

	Describe("copy routes from live app to new app", func() {
		var (
			liveApp, newApp Application
		)

		BeforeEach(func() {
			liveApp = Application{
				Name: "live",
				Routes: []Route{
					{Host: "live", Domain: Domain{Name: "mybluemix.net"}},
					{Host: "live", Domain: Domain{Name: "example.com"}},
				},
			}
			newApp = Application{
				Name: "new",
			}
		})

		It("copies all routes from live app to new app including the default route", func() {
			p.CopyLiveAppRoutesToNewApp(liveApp.Name, newApp.Name, liveApp.Routes)

			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(Equal([]string{
				"map-route new mybluemix.net -n live",
				"map-route new example.com -n live",
			}))
		})
	})

	Describe("remove routes from old app", func() {
		var (
			oldApp Application
		)

		BeforeEach(func() {
			oldApp = Application{
				Name: "old",
				Routes: []Route{
					{Host: "live", Domain: Domain{Name: "mybluemix.net"}},
					{Host: "live", Domain: Domain{Name: "example.com"}},
				},
			}
		})

		It("unmaps all routes from the old app", func() {
			p.UnmapRoutesFromOldApp(oldApp.Name, oldApp.Routes)

			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(Equal([]string{
				"unmap-route old mybluemix.net -n live",
				"unmap-route old example.com -n live",
			}))
		})
	})

	Describe("unmapping temporary route from new app", func() {
		newApp := Application{
			Name: "app-new",
			Routes: []Route{
				{Host: "app-new", Domain: Domain{Name: "mybluemix.net"}},
				{Host: "app", Domain: Domain{Name: "mybluemix.net"}},
			},
		}

		tempRoute := Route{Host: "app-new", Domain: Domain{Name: "mybluemix.net"}}

		It("unmaps the temporary route", func() {
			p.UnmapTemporaryRouteFromNewApp(newApp.Name, tempRoute)

			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(Equal([]string{
				"unmap-route app-new mybluemix.net -n app-new",
			}))
		})

		Context("when the unmapping fails", func() {
			BeforeEach(func() {
				connection.CliCommandStub = func(args ...string) ([]string, error) {
					return nil, errors.New("failed to unmap route")
				}
			})

			It("returns an error", func() {
				p.UnmapTemporaryRouteFromNewApp(newApp.Name, tempRoute)

				Expect(bgdExitsWithErrors[0]).To(HaveOccurred())
			})
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
			apps []Application
		)

		Context("with live and old apps", func() {
			BeforeEach(func() {
				apps = []Application{
					{Name: "app-name-old"},
					{Name: "app-name"},
				}
				appLister := &fakeAppLister{Apps: apps}
				p.AppLister = appLister
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
				apps = []Application{
					{Name: "app-name-failed"},
					{Name: "app-name"},
				}
				appLister := &fakeAppLister{Apps: apps}
				p.AppLister = appLister
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
				apps = []Application{
					{Name: "app-name-new"},
					{Name: "app-name"},
				}
				appLister := &fakeAppLister{Apps: apps}
				p.AppLister = appLister
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
				apps = []Application{
					{Name: "app-name"},
				}
				appLister := &fakeAppLister{Apps: apps}
				p.AppLister = appLister
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
			apps := []Application{
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
			apps := []Application{}

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
		newApp := Application{Name: "app-name-new"}
		newRoute := Route{Host: newApp.Name, Domain: Domain{Name: "example.com"}}

		It("pushes an app with new appended to its name", func() {
			p.PushNewApp(&newApp, newRoute)

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`^push app-name-new`))
		})

		It("uses the generated name for the route", func() {
			p.PushNewApp(&newApp, newRoute)

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`-n app-name-new`))
		})

		It("pushes with the default cf domain", func() {
			p.PushNewApp(&newApp, newRoute)

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`-d example.com`))
		})

		Context("when the push fails", func() {
			BeforeEach(func() {
				connection.CliCommandStub = func(args ...string) ([]string, error) {
					return nil, errors.New("failed to push app")
				}
			})

			It("returns an error", func() {
				p.PushNewApp(&newApp, newRoute)

				Expect(bgdExitsWithErrors[0]).To(MatchError("failed to push app"))
			})
		})
	})

	Describe("live app", func() {
		oldApp := Application{Name: "app-name-old"}
		liveApp := Application{Name: "app-name"}
		newerLiveApp := Application{Name: "app-name"}

		Context("with live and old apps", func() {
			It("returns the live app", func() {
				p.AppLister = &fakeAppLister{Apps: []Application{oldApp, liveApp}}

				Expect(p.LiveApp("app-name")).To(Equal(&liveApp))
			})
		})

		Context("with multiple live apps", func() {
			It("returns the last pushed app", func() {
				p.AppLister = &fakeAppLister{Apps: []Application{liveApp, newerLiveApp}}

				Expect(p.LiveApp("app-name")).To(Equal(&newerLiveApp))
			})
		})

		Context("with no apps", func() {
			It("returns no app", func() {
				p.AppLister = &fakeAppLister{Apps: []Application{}}

				Expect(p.LiveApp("app-name")).To(BeNil())
			})
		})
	})

	Describe("app filter", func() {
		Context("when there are 2 old versions and 1 non-old version", func() {
			var (
				appList    []Application
				currentApp *Application
				oldApps    []Application
			)

			BeforeEach(func() {
				appList = []Application{
					{Name: "foo-old"},
					{Name: "foo-old"},
					{Name: "foo"},
					{Name: "bar-foo-old"},
					{Name: "foo-older"},
				}
				currentApp, oldApps = p.FilterApps("foo", appList)
			})

			Describe("current app", func() {
				Context("when there is no current live app", func() {
					It("returns an empty struct", func() {
						app, _ := p.FilterApps("bar", appList)
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

func getAllCfCommands(connection *fakes.FakeCliConnection) (commands []string) {
	commands = []string{}
	for i := 0; i < connection.CliCommandCallCount(); i++ {
		args := connection.CliCommandArgsForCall(i)
		commands = append(commands, strings.Join(args, " "))
	}
	return
}

type fakeAppLister struct {
	Apps []Application
}

func (l *fakeAppLister) AppsInCurrentSpace() ([]Application, error) {
	return l.Apps, nil
}
