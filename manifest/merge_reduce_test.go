package manifest_test

import (
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Map converter and merger", func() {

	Context("When the input is not a map", func() {
		newMap, err := manifest.Mappify("hello")

		It("Returns nil", func() {
			Expect(newMap).To(BeNil())
		})

		It("Returns an error", func() {
			Expect(err).ToNot(BeNil())
		})
		It("Returns a descriptive error", func() {
			Expect(err.Error()).To(ContainSubstring("expected map"))
		})

	})

	Context("When converting an typed map to one with string keys", func() {
		testmap := make(map[string]string)
		mappedmap, err := manifest.Mappify(testmap)
		Context("when input map is empty", func() {
			It("Returns a map", func() {
				Expect(mappedmap).ToNot(BeNil())
			})

			It("Has no error", func() {
				Expect(err).To(BeNil())
			})
		})

		Context("when input map is not empty", func() {
			testmap := make(map[interface{}]interface{})
			testmap["foo"] = "bar"
			mappedmap, err := manifest.Mappify(testmap)
			It("Correctly handles keys", func() {
				Expect(mappedmap["foo"]).To(Equal("bar"))
			})

			It("Has no error", func() {
				Expect(err).To(BeNil())
			})
		})

	})

	Context("When converting an untyped map to one with string keys", func() {
		testmap := make(map[interface{}]interface{})
		mappedmap, err := manifest.Mappify(testmap)
		Context("when input map is empty", func() {
			It("Returns a map", func() {
				Expect(mappedmap).ToNot(BeNil())
			})

			It("Has no error", func() {
				Expect(err).To(BeNil())
			})
		})

		Context("when input map is not empty", func() {
			testmap := make(map[interface{}]interface{})
			testmap["foo"] = "bar"
			mappedmap, err := manifest.Mappify(testmap)
			It("Correctly converts keys which are actually strings", func() {
				Expect(mappedmap["foo"]).To(Equal("bar"))
			})

			It("Has no error", func() {
				Expect(err).To(BeNil())
			})
		})

	})

	Context("When doing a deep merge of a map", func() {
		testmap1 := make(map[string]interface{})
		testmap2 := make(map[string]interface{})
		Context("when both input maps are empty", func() {
			mappedmap, err := manifest.DeepMerge(testmap1, testmap2)
			It("Returns a an empty map", func() {
				Expect(len(mappedmap)).To(Equal(0))
			})

			It("Has no error", func() {
				Expect(err).To(BeNil())
			})
		})

		Context("when both input maps share a key", func() {
			testmap1["foo"] = "baz"
			testmap2["foo"] = "bar"

			mappedmap1, err := manifest.DeepMerge(testmap1, testmap2)
			It("Favours the value from the second argument", func() {
				Expect(mappedmap1["foo"]).To(Equal("bar"))
			})

			It("Has no error", func() {
				Expect(err).To(BeNil())
			})

			mappedmap2, err := manifest.DeepMerge(testmap2, testmap1)
			It("Favours the value from the second argument if the arguments are reversed", func() {
				Expect(mappedmap2["foo"]).To(Equal("baz"))
			})

			It("Has no error", func() {
				Expect(err).To(BeNil())
			})
		})
	})

})
