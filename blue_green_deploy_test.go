package main_test

import (
	"strings"

	"github.com/cloudfoundry/cli/plugin/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"
)

var _ = Describe("BlueGreenDeploy", func() {
	Describe("RemapRoutesFromLiveappToNewApp", func() {
		var (
			connection      *fakes.FakeCliConnection
			liveApp, newApp Application
			errors          []error
			p               BlueGreenDeploy
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

			connection = &fakes.FakeCliConnection{}
			errors = []error{}
			testErrorFunc := func(message string, err error) {
				errors = append(errors)
			}
			p = BlueGreenDeploy{Connection: connection, ErrorFunc: testErrorFunc}
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
})

func getAllCfCommands(connection *fakes.FakeCliConnection) (commands []string) {
	commands = []string{}
	for i := 0; i < connection.CliCommandCallCount(); i++ {
		args := connection.CliCommandArgsForCall(i)
		commands = append(commands, strings.Join(args, " "))
	}
	return
}
