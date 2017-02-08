package manifest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloneWithExclude", func() {

	Context("When the map contains some values and excludeKey exists", func() {
		It("should return a new map without the excludeKey", func() {
			input := map[string]interface{}{
				"one":   1,
				"two":   2138,
				"three": 1908,
			}

			expected := map[string]interface{}{
				"one":   1,
				"three": 1908,
			}

			excludeKey := "two"

			actual := cloneWithExclude(input, excludeKey)

			Expect(actual).To(Equal(expected))
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
