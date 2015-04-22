package main_test

import (
	"errors"
	"strings"

	"github.com/cloudfoundry/cli/plugin/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"
)

var _ = Describe("BlueGreenDeploy", func() {
	var (
		bgdErrors     []error
		connection    *fakes.FakeCliConnection
		p             BlueGreenDeploy
		testErrorFunc func(message string, err error)
	)

	BeforeEach(func() {
		bgdErrors = []error{}
		testErrorFunc = func(message string, err error) {
			bgdErrors = append(bgdErrors, err)
		}

		connection = &fakes.FakeCliConnection{}
		p = BlueGreenDeploy{Connection: connection, ErrorFunc: testErrorFunc}
	})

	Describe("RemapRoutesFromLiveappToNewApp", func() {
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

		It("map and unmap all routes from live app to the new app except the default route", func() {
			p.RemapRoutesFromLiveAppToNewApp(liveApp, newApp)

			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(Equal([]string{
				"map-route new example.com -n live",
				"unmap-route live example.com -n live",
			}))
		})
	})

	Describe("marking apps as old", func() {
		app := Application{Name: "app"}

		It("appends -old to app name", func() {
			p.MarkAppAsOld(&app)

			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(Equal([]string{
				"rename app app-old",
			}))
		})

		Context("when renaming the app fails", func() {
			It("calls the error callback", func() {
				connection.CliCommandStub = func(args ...string) ([]string, error) {
					return nil, errors.New("failed to rename app")
				}

				p.MarkAppAsOld(&app)
				Expect(bgdErrors[0]).To(HaveOccurred())
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
					Expect(bgdErrors[0]).To(HaveOccurred())
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
				Expect(bgdErrors).To(HaveLen(0))
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
					Expect(bgdErrors[0]).To(HaveOccurred())
				})
			})
		})

		Context("when there is no old version deployed", func() {
			apps := []Application{}

			It("succeeds", func() {
				p.DeleteAppVersions(apps)
				Expect(bgdErrors).To(HaveLen(0))
			})

			It("deletes nothing", func() {
				p.DeleteAppVersions(apps)
				Expect(connection.CliCommandCallCount()).To(Equal(0))
			})
		})
	})

	Describe("pushing a new app", func() {
		var (
			appLister *fakeAppLister
		)

		BeforeEach(func() {
			appLister = &fakeAppLister{Apps: []Application{}}
			p.AppLister = appLister
		})

		It("pushes an app with the timestamp appended to its name", func() {
			p.PushNewApp("app-name")

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`^push app-name`))
		})

		It("uses the generated name for the route", func() {
			p.PushNewApp("app-name")

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`-n app-name`))
		})

		It("returns the new app as an Application", func() {
			// stubbing cf push so it appends the newly pushed app to the list of
			// fixtures for testing subsequent operations
			connection.CliCommandStub = func(args ...string) ([]string, error) {
				appLister.Apps = append(appLister.Apps, Application{
					Name: args[1],
					Routes: []Route{
						{
							Host: "testroute",
						},
					}})
				return nil, nil
			}
			var newApp Application = p.PushNewApp("app-name")

			Expect(newApp.Name).To(MatchRegexp(`^app-name$`))
			Expect(newApp.Routes[0].Host).To(Equal("testroute"))
		})

		Context("when the push fails", func() {
			BeforeEach(func() {
				connection.CliCommandStub = func(args ...string) ([]string, error) {
					return nil, errors.New("failed to push app")
				}
			})

			It("returns an error", func() {
				p.PushNewApp("app-name")

				Expect(bgdErrors[0]).To(MatchError("failed to push app"))
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
