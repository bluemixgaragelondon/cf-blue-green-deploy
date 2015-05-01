package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"
)

var _ = Describe("Application", func() {
	Describe("default route", func() {
		It("returns the route with host same as app name and default domain", func() {
			app := Application{
				DefaultDomain: "mybluemix.net",
				Name:          "app-new",
				Routes: []Route{
					{Host: "app-new", Domain: Domain{Name: "example.com"}},
					{Host: "app-new", Domain: Domain{Name: "mybluemix.net"}},
					{Host: "app", Domain: Domain{Name: "example.com"}},
				},
			}

			Expect(app.DefaultRoute()).To(Equal(Route{
				Host:   "app-new",
				Domain: Domain{Name: "mybluemix.net"},
			}))
		})
	})
})

var _ = Describe("Route", func() {
	Describe("fqdn", func() {
		It("returns the fqdn of the route", func() {
			route := Route{Host: "testroute", Domain: Domain{Name: "example.com"}}
			Expect(route.FQDN()).To(Equal("testroute.example.com"))
		})
	})
})
