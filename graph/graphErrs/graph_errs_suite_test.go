package graphErrs_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGraphErrs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GraphErrs Suite")
}
