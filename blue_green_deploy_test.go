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
				Name: "live-20150410155216",
				Routes: []Route{
					{Host: "live-20150410155216", Domain: Domain{Name: "mybluemix.net"}},
					{Host: "live", Domain: Domain{Name: "example.com"}},
				},
			}
			newApp = Application{
				Name: "new",
			}
		})

		It("map and unmaps routes from live app to the new app", func() {
			p.RemapRoutesFromLiveAppToNewApp(liveApp, newApp)

			cfCommands := getAllCfCommands(connection)

			Expect(cfCommands).To(Equal([]string{
				"unmap-route live-20150410155216 mybluemix.net -n live-20150410155216",
				"map-route new example.com -n live",
				"unmap-route live-20150410155216 example.com -n live",
			}))
		})
	})

	Describe("the DeleteAppVersions function", func() {
		Context("when there is an old version deployed", func() {
			apps := []Application{
				{Name: "app-name-20150326110000-old"},
				{Name: "app-name-20150325110000-old"},
			}

			It("deletes the apps", func() {
				p.DeleteAppVersions(apps)
				cfCommands := getAllCfCommands(connection)

				Expect(cfCommands).To(Equal([]string{
					"delete app-name-20150326110000-old -f -r",
					"delete app-name-20150325110000-old -f -r",
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

	Describe("the PushNewAppVersion function", func() {
		var (
			appLister *fakeAppLister
		)

		BeforeEach(func() {
			appLister = &fakeAppLister{Apps: []Application{}}
			p.AppLister = appLister
		})

		It("pushes an app with the timestamp appended to its name", func() {
			p.PushNewAppVersion("app-name")

			Expect(strings.Join(connection.CliCommandArgsForCall(0), " ")).
				To(MatchRegexp(`^push app-name-\d{14}$`))
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
			var newApp Application = p.PushNewAppVersion("app-name")

			Expect(newApp.Name).To(MatchRegexp(`^app-name-\d{14}$`))
			Expect(newApp.Routes[0].Host).To(Equal("testroute"))
		})

		Context("when the push fails", func() {
			BeforeEach(func() {
				connection.CliCommandStub = func(args ...string) ([]string, error) {
					return nil, errors.New("failed to push app")
				}
			})

			It("returns an error", func() {
				p.PushNewAppVersion("app-name")

				Expect(bgdErrors[0]).To(MatchError("failed to push app"))
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
