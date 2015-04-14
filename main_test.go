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
