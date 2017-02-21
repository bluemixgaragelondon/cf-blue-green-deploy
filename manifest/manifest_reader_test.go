package manifest_test

import (
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest reader", func() {

	Context("when a custom manifest file is provided", func() {

		It("should load that file rather than a default one", func() {
			reader := &manifest.FileManifestReader{ManifestPath: "../fixtures/custom-manifest.yml"}
			manifest, err := reader.Read()
			Expect(manifest).ToNot(BeNil())
			Expect(manifest.Data["name"]).To(Equal("my-app"))
			Expect(err).To(BeNil())
		})
	})

	Context("when a custom directory (but no file name)", func() {

		It("should load the default file from the custom directory", func() {
			reader := &manifest.FileManifestReader{ManifestPath: "../fixtures"}
			manifest, err := reader.Read()
			Expect(manifest).ToNot(BeNil())
			Expect(manifest.Data["name"]).To(Equal("plain-app"))
			Expect(err).To(BeNil())
		})
	})

	Context("When no manifest file is present", func() {

		It("Returns nil with error message", func() {
			reader := &manifest.FileManifestReader{ManifestPath: "../doesnotexist"}
			manifest, err := reader.Read()
			Expect(manifest).To(BeNil())
			Expect(err).ToNot(BeNil())
		})
	})

	Context("When manifest file is empty", func() {

		It("Returns nil with error message", func() {
			reader := &manifest.FileManifestReader{ManifestPath: "../fixtures/emptymanifest.yml"}
			manifest, err := reader.Read()
			Expect(manifest).To(BeNil())
			Expect(err).ToNot(BeNil())
		})
	})

	Context("When nothing is passed", func() {

		It("Returns nil with an error", func() {
			reader := &manifest.FileManifestReader{}
			manifest, err := reader.Read()
			Expect(manifest).To(BeNil())
			Expect(err).ToNot(BeNil())
		})
	})

	Context("When a manifest which inherits config from another manifest is passed", func() {

		It("Returns the configurations from the passed in manifest and all inherited manifests", func() {
			reader := &manifest.FileManifestReader{ManifestPath: "../fixtures/manifestwithinheritance.yml"}
			manifest, err := reader.Read()
			Expect(err).To(BeNil())
			Expect(manifest).ToNot(BeNil())
			Expect(manifest.Data["name"]).To(Equal("fancy-app"))
			Expect(manifest.Data["domain"]).To(Equal("shared-domain.example.com"))
		})
	})
})
