package main_test

import (
	. "github.com/bluemixgaragelondon/cf-blue-green-deploy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strings"
)

var _ = Describe("Args", func() {
	Context("With an appname only", func() {
		in := strings.Split("bgd appname", " ")
		args, err := NewArgs(in)

		It("does not error", func() {
			Expect(err).To(BeNil())
		})

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
		in := strings.Split("bgd appname --smoke-test script/smoke-test", " ")
		args, err := NewArgs(in)

		It("does not error", func() {
			Expect(err).To(BeNil())
		})

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
		in := strings.Split("bgd appname --smoke-test smokey -f manifest.yml", " ")
		args, err := NewArgs(in)

		It("does not error", func() {
			Expect(err).To(BeNil())
		})

		It("sets the smoke test file", func() {
			Expect(args.SmokeTestPath).To(Equal("smokey"))
		})

		It("sets the app name", func() {
			Expect(args.AppName).To(Equal("appname"))
		})

		It("sets a manifest", func() {
			Expect(args.ManifestPath).To(Equal("manifest.yml"))
		})
	})
})
