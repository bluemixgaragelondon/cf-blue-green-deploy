package manifest_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestManifest(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("manifest-junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Manifest Suite", []Reporter{junitReporter})

}
