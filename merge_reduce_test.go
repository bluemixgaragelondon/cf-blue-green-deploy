package main_test

import (
	. "github.com/bluemixgaragelondon/cf-blue-green-deploy/from-cf-codebase/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Map converter and merger", func() {

	Context("When converting an untyped map to one with string keys", func() {
		Context("when input map is empty", func() {
			It("Returns a map", func() {
				testmap := make(map[interface{}]interface{})
				mappedmap := Mappify(testmap)
				Expect(mappedmap).ToNot(BeNil())
			})
		})

		Context("when input map is not empty", func() {
			It("Correctly converts keys which are actually strings", func() {
				testmap := make(map[interface{}]interface{})
				testmap["foo"] = "bar"
				mappedmap := Mappify(testmap)
				Expect(mappedmap["foo"]).To(Equal("bar"))
			})
		})

	})

	Context("When doing a deep merge of a map", func() {
		Context("when both input maps are empty", func() {
			It("Returns a an empty map", func() {
				testmap1 := make(map[string]interface{})
				testmap2 := make(map[string]interface{})

				mappedmap := DeepMerge(testmap1, testmap2)
				Expect(len(mappedmap)).To(Equal(0))
			})
		})

		Context("when both input maps share a key", func() {
			testmap1 := make(map[string]interface{})
			testmap2 := make(map[string]interface{})
			testmap1["foo"] = "baz"
			testmap2["foo"] = "bar"

			It("Favours the value from the second argument", func() {
				mappedmap := DeepMerge(testmap1, testmap2)
				Expect(mappedmap["foo"]).To(Equal("bar"))
			})

			It("Favours the value from the second argument if the arguments are reversed", func() {
				mappedmap := DeepMerge(testmap2, testmap1)
				Expect(mappedmap["foo"]).To(Equal("baz"))
			})
		})
	})

})
