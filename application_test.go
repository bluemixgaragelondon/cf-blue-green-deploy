package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"
)

var _ = Describe("Application", func() {
	Describe("DefaultRoute", func() {
		Context("when there is one route that has a timestamp", func() {
			It("returns that route", func() {
				app := Application{
					Name: "app-20150410155216",
					Routes: []Route{
						{Host: "app-20150410155216", Domain: Domain{Name: "mybluemix.net"}},
						{Host: "app", Domain: Domain{Name: "example.com"}},
					},
				}

				Expect(app.DefaultRoute()).To(Equal(Route{
					Host:   "app-20150410155216",
					Domain: Domain{Name: "mybluemix.net"},
				}))
			})
		})
	})
})

var _ = Describe("Route", func() {
	Describe("FQDN()", func() {
		It("returns the fqdn of the route", func() {
			route := Route{Host: "testroute", Domain: Domain{Name: "example.com"}}
			Expect(route.FQDN()).To(Equal("testroute.example.com"))
		})
	})
})
