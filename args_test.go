package main_test

import (
	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strings"
)

var _ = Describe("Args", func() {
	Context("With a smoke test and an appname", func() {
		args := NewArgs(bgdArgs("--smoke-test foo appname"))

		It("sets the smoke test file", func() {
			Expect(args.SmokeTestPath).To(Equal("foo"))
		})

		It("sets the app name", func() {
			Expect(args.AppName).To(Equal("appname"))
		})

		It("does not set a manifest", func() {
			Expect(args.ManifestPath).To(BeZero())
		})
	})

	Context("With a smoke test and a manifest", func() {
		args := NewArgs(bgdArgs("--smoke-test smokey -f manifest.yml"))

		It("sets the smoke test file", func() {
			Expect(args.SmokeTestPath).To(Equal("smokey"))
		})

		It("sets the app name", func() {
			Expect(args.AppName).To(BeZero())
		})

		It("does not set a manifest", func() {
			Expect(args.ManifestPath).To(Equal("manifest.yml"))
		})
	})
})

func bgdArgs(argString string) []string {
	args := strings.Split(argString, " ")
	return append([]string{"cf", "blue-green-deploy"}, args...)
}
