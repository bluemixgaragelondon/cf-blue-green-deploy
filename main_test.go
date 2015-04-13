package main_test

import (
	"regexp"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	plugin "hub.jazz.net/git/bluemixgarage/cf-blue-green-deploy"
)

var _ = Describe("BGD Plugin", func() {
	Describe("smoke test script", func() {
		Context("when smoke test flag is not provided", func() {
			It("returns empty string", func() {
				args := []string{"blue-green-deploy", "appName"}
				Expect(plugin.ExtractIntegrationTestScript(args)).To(Equal(""))
			})
		})

		Context("when smoke test flag provided", func() {
			It("returns flag value", func() {
				args := []string{"blue-green-deploy", "appName", "--smoke-test=script/test"}
				Expect(plugin.ExtractIntegrationTestScript(args)).To(Equal("script/test"))
			})
		})
	})

	Describe("smoke test runner", func() {
		It("returns stdout", func() {
			out, _ := plugin.RunIntegrationTestScript("test/support/smoke-test-script", "app.mybluemix.net")
			Expect(out).To(ContainSubstring("STDOUT"))
		})

		It("returns stderr", func() {
			out, _ := plugin.RunIntegrationTestScript("test/support/smoke-test-script", "app.mybluemix.net")
			Expect(out).To(ContainSubstring("STDERR"))
		})

		It("passes app FQDN as first argument", func() {
			out, _ := plugin.RunIntegrationTestScript("test/support/smoke-test-script", "app.mybluemix.net")
			Expect(out).To(ContainSubstring("App FQDN is: app.mybluemix.net"))
		})

		Context("when script doesn't exist", func() {
			It("fails with useful error", func() {
				_, err := plugin.RunIntegrationTestScript("inexistent-smoke-test-script", "app.mybluemix.net")
				Expect(err.Error()).To(ContainSubstring("executable file not found"))
			})
		})

		Context("when script isn't executable", func() {
			It("fails with useful error", func() {
				_, err := plugin.RunIntegrationTestScript("test/support/nonexec-smoke-test-script", "app.mybluemix.net")
				Expect(err.Error()).To(ContainSubstring("permission denied"))
			})
		})
	})

	Describe("app filter", func() {
		Context("when there are 2 old versions and 1 non-old version", func() {
			var (
				appList    []plugin.Application
				currentApp *plugin.Application
				oldApps    []plugin.Application
			)

			BeforeEach(func() {
				appList = []plugin.Application{
					{Name: "foo-20150408114041-old"},
					{Name: "foo-20141234567348-old"},
					{Name: "foo-20163453473845"},
					{Name: "bar-foo-20141234567348-old"},
					{Name: "foo-20141234567348-older"},
				}
				currentApp, oldApps = plugin.FilterApps("foo", appList)
			})

			Describe("current app", func() {
				Context("when there is no current live app", func() {
					It("returns an empty struct", func() {
						app, _ := plugin.FilterApps("bar", appList)
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

	Describe("app name generator", func() {
		generated := plugin.GenerateAppName("foo")

		It("uses the passed name as a prefix", func() {
			Expect(generated).To(HavePrefix("foo"))
		})

		It("uses a timestamp as a suffix", func() {
			now, _ := strconv.Atoi(time.Now().Format("20060102150405"))
			genTimestamp, _ := strconv.Atoi(regexp.MustCompile(`\d{14}`).FindString(generated))

			Expect(now - genTimestamp).To(BeNumerically("<", 5))
		})
	})
})
