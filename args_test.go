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

	Context("With a smoke test and a manifest and app name at the end", func() {
		in := strings.Split("bgd --smoke-test smokey -f manifest.yml appname", " ")
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

	Context("With a manifest only", func() {
		in := strings.Split("bgd -f myManifest.yml", " ")
		args, err := NewArgs(in)

		It("does not error", func() {
			Expect(err).To(BeNil())
		})

		It("does not set an app name", func() {
			Expect(args.AppName).To(BeZero())
		})

		It("Sets a manifest", func() {
			Expect(args.ManifestPath).To(Equal("myManifest.yml"))
		})

		It("does not set the smoke test file", func() {
			Expect(args.SmokeTestPath).To(BeZero())
		})
	})

	Context("With a smoke test only", func() {
		in := strings.Split("bgd --smoke-test smokey", " ")
		args, err := NewArgs(in)

		It("does not error", func() {
			Expect(err).To(BeNil())
		})

		It("sets the smoke test file", func() {
			Expect(args.SmokeTestPath).To(Equal("smokey"))
		})

		It("does not set an app name", func() {
			Expect(args.AppName).To(BeZero())
		})

		It("does not set a manifest", func() {
			Expect(args.ManifestPath).To(BeZero())
		})
	})
})
