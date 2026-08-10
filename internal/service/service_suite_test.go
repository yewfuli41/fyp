package service_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/lib/pq"
)

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Suite")
}

func newUniqueViolation(constraint string) error {
	return &pq.Error{
		Code:       "23505",
		Constraint: constraint,
	}
}
