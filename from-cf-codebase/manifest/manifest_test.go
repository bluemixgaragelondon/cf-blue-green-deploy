package manifest

import (
	//	"code.cloudfoundry.org/cli/plugin/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {

	Context("For a manifest with no applications section", func() {

		input := map[string]interface{}{
			"host": "bob",
			"routes": []interface{}{
				map[interface{}]interface{}{"route": "example.com"},
				map[interface{}]interface{}{"route": "www.example.com/foo"},
				map[interface{}]interface{}{"route": "tcp-example.com:1234"},
			},
		}
		m := NewEmptyManifest()

		Context("the getAppMaps function", func() {
			appMaps, err := m.getAppMaps(input)
			It("does not error", func() {
				Expect(err).To(BeNil())
			})

			It("should return one entry", func() {
				Expect(len(appMaps)).To(Equal(1))
			})

			It("should return global properties", func() {
				Expect(appMaps).To(Equal([]map[string]interface{}{input}))
			})
		})

		Context("the parseRoutes function", func() {
			errs := []error{}
			routeStuff := parseRoutes(input, &errs)

			It("does not error", func() {
				Expect(len(errs)).To(Equal(0))
			})

			It("should return three routes", func() {
				Expect(len(routeStuff)).To(Equal(3))
			})

			It("should return global properties", func() {
				// We're only testing for domain because of limitations in the route struct
				Expect(routeStuff[0].Domain.Name).To(Equal("example.com"))
			})
		})
	})

	Context("For a manifest with an applications section", func() {
		applicationsContents := []interface{}{map[string]string{
			"fred": "hello",
		}}
		input := map[string]interface{}{
			"applications": applicationsContents,
			"host":         "bob",
		}

		m := NewEmptyManifest()
		appMaps, err := m.getAppMaps(input)

		Context("the AppMaps function", func() {
			It("does not error", func() {
				Expect(err).To(BeNil())
			})

			It("should not alter what gets passed in", func() {

				Expect(input["applications"]).To(Equal(applicationsContents))
				// Make sure this doesn't change what's passed in
				Expect(input["applications"]).To(Equal(applicationsContents))

			})

			It("should return one entry", func() {
				Expect(len(appMaps)).To(Equal(1))
			})

			It("should merge global properties with application-level properties", func() {

				Expect(appMaps[0]["host"]).To(Equal("bob"))
				Expect(appMaps[0]["fred"]).To(Equal("hello"))

			})
		})
	})

	Context("For a manifest with two applications in the applications section", func() {
		applicationsContents := []interface{}{map[string]string{
			"fred": "hello",
		},
			map[string]string{
				"george": "goodbye",
			}}
		input := map[string]interface{}{
			"applications": applicationsContents,
			"host":         "bob",
		}

		m := NewEmptyManifest()
		appMaps, err := m.getAppMaps(input)

		Context("the AppMaps function", func() {
			It("does not error", func() {
				Expect(err).To(BeNil())
			})

			It("should not alter what gets passed in", func() {

				Expect(input["applications"]).To(Equal(applicationsContents))
				// Make sure this doesn't change what's passed in
				Expect(input["applications"]).To(Equal(applicationsContents))

			})

			It("should return two entry", func() {
				Expect(len(appMaps)).To(Equal(2))
			})

			It("should merge global properties with application-level properties", func() {

				Expect(appMaps[0]["host"]).To(Equal("bob"))
				Expect(appMaps[0]["fred"]).To(Equal("hello"))
				Expect(appMaps[0]["george"]).To(BeNil())

				Expect(appMaps[1]["host"]).To(Equal("bob"))
				Expect(appMaps[1]["george"]).To(Equal("goodbye"))
				Expect(appMaps[1]["fred"]).To(BeNil())

			})
		})
	})

})

var _ = Describe("CloneWithExclude", func() {

	Context("When the map contains some values and excludeKey exists", func() {

		input := map[string]interface{}{
			"one":   1,
			"two":   2138,
			"three": 1908,
		}

		excludeKey := "two"

		actual := cloneWithExclude(input, excludeKey)

		It("should return a new map without the excludeKey", func() {

			expected := map[string]interface{}{
				"one":   1,
				"three": 1908,
			}

			Expect(actual).To(Equal(expected))
		})

		It("should not alter the original map", func() {
			Expect(input["two"]).To(Equal(2138))
		})
	})

	Context("When the map contains some values and excludeKey does not exist", func() {
		It("should return a new map with the same contents as the original", func() {
			input := map[string]interface{}{
				"one":   1,
				"two":   2138,
				"three": 1908,
			}

			excludeKey := "four"

			actual := cloneWithExclude(input, excludeKey)

			Expect(actual).To(Equal(input))
		})
	})

	Context("When the map contains a key that includes the excludeKey", func() {
		It("should return a new map with the same contents as the original", func() {
			input := map[string]interface{}{
				"one":       1,
				"two":       2138,
				"threefour": 1908,
			}

			excludeKey := "four"

			actual := cloneWithExclude(input, excludeKey)

			Expect(actual).To(Equal(input))
		})
	})

	Context("When the map is empty", func() {
		It("should return a new empty map", func() {
			input := map[string]interface{}{}

			excludeKey := "one"

			actual := cloneWithExclude(input, excludeKey)

			Expect(actual).To(Equal(input))
		})
	})
})
