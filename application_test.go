package main_test

import (
	"code.cloudfoundry.org/cli/plugin/models"
	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route", func() {
	Describe("fqdn", func() {
		It("returns the fqdn of the route", func() {
			route := plugin_models.GetApp_RouteSummary{Host: "testroute", Domain: plugin_models.GetApp_DomainFields{Name: "example.com"}}
			Expect(FQDN(route)).To(Equal("testroute.example.com"))
		})
	})
})
