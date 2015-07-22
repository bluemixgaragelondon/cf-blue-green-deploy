package main_test

import (
	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route", func() {
	Describe("fqdn", func() {
		It("returns the fqdn of the route", func() {
			route := Route{Host: "testroute", Domain: Domain{Name: "example.com"}}
			Expect(route.FQDN()).To(Equal("testroute.example.com"))
		})
	})
})
