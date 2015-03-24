package main_test

import (
	. "github.com/onsi/ginkgo"
	plugin "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"

	"github.com/cloudfoundry/cli/plugin/fakes"
)

var _ = Describe("Main", func() {
	Describe("Plugin", func() {
		Describe("blue-green-deploy", func() {
			It("exists", func() {
				fake := &fakes.FakeCliConnection{}
				bgdPlugin := plugin.BgdPlugin{}
				bgdPlugin.Run(fake, []string{})
			})
		})
	})
})
