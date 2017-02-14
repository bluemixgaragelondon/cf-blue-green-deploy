package manifest_test

import (
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest reader", func() {
	// TODO We need to test
	// * manifest path params when nothing is passed
	// * manifest inheritance

	Context("when a custom manifest file is provided", func() {
		It("should load that file rather than a default one", func() {
			reader := &manifest.FileManifestReader{ManifestPath: "../fixtures/custom-manifest.yml"}
			manifest := reader.Read()
			Expect(manifest).ToNot(BeNil())
			Expect(manifest.Data["name"]).To(Equal("my-app"))
		})
	})

	Context("when a custom directory (but no file name)", func() {
		It("should load the default file from the custom directory", func() {
			reader := &manifest.FileManifestReader{ManifestPath: "../fixtures"}
			manifest := reader.Read()
			Expect(manifest).ToNot(BeNil())
			Expect(manifest.Data["name"]).To(Equal("plain-app"))
		})
	})

	Context("When no manifest file is present", func() {

		It("Returns nil", func() {
			reader := &manifest.FileManifestReader{ManifestPath: "../doesnotexist"}
			manifest := reader.Read()
			Expect(manifest).To(BeNil())
		})
	})

	Context("When manifest file is empty", func() {

		It("Returns nil", func() {
			reader := &manifest.FileManifestReader{ManifestPath: "../fixtures/emptymanifest.yml"}
			manifest := reader.Read()
			Expect(manifest).To(BeNil())
		})
	})
})
