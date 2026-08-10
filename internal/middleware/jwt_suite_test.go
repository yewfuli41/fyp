package middleware

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJWTMiddleware(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JWT Middleware Suite")
}
