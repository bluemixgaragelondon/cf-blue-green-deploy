package main_test

import (
	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strings"
)

var _ = Describe("Args", func() {
	Context("With an appname only", func() {
		args := NewArgs(bgdArgs("appname"))

		It("sets the app name", func() {
			Expect(args.AppName).To(Equal("appname"))
		})

		It("does not set the smoke test file", func() {
			Expect(args.SmokeTestPath).To(BeZero())
		})

		It("does not set a manifest", func() {
			Expect(args.ManifestPath).To(BeZero())
		})
	})

	Context("With a smoke test and an appname", func() {
		args := NewArgs(bgdArgs("appname --smoke-test script/smoke-test"))

		It("sets the smoke test file", func() {
			Expect(args.SmokeTestPath).To(Equal("script/smoke-test"))
		})

		It("sets the app name", func() {
			Expect(args.AppName).To(Equal("appname"))
		})

		It("does not set a manifest", func() {
			Expect(args.ManifestPath).To(BeZero())
		})
	})

	Context("With an appname smoke test and a manifest", func() {
		args := NewArgs(bgdArgs("appname --smoke-test smokey -f custommanifest.yml"))

		It("sets the smoke test file", func() {
			Expect(args.SmokeTestPath).To(Equal("smokey"))
		})

		It("sets the app name", func() {
			Expect(args.AppName).To(Equal("appname"))
		})

		It("sets a manifest", func() {
			Expect(args.ManifestPath).To(Equal("custommanifest.yml"))
		})
	})

	Context("With an appname and a manifest", func() {
		args := NewArgs(bgdArgs("appname -f custommanifest.yml"))

		It("sets the app name", func() {
			Expect(args.AppName).To(Equal("appname"))
		})

		It("sets a manifest", func() {
			Expect(args.ManifestPath).To(Equal("custommanifest.yml"))
		})
	})

	Context("When a global cf flag is set with an app name", func() {
		args := NewArgs([]string{"cf", "-v", "blue-green-deploy", "app"})

		It("sets the app name", func() {
			Expect(args.AppName).To(Equal("app"))
		})
	})

	Context("When the bgd abbreviation is used", func() {
		args := NewArgs([]string{"cf", "bgd", "app"})

		It("sets the app name", func() {
			Expect(args.AppName).To(Equal("app"))
		})
	})
})

func bgdArgs(argString string) []string {
	args := strings.Split(argString, " ")
	return append([]string{"blue-green-deploy"}, args...)
}
