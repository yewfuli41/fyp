package param_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestParam(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Param Suite")
}
