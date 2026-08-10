package utils_test

import (
	"database/sql"
	"fyp/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NullStringPtr", func() {
	It("returns nil for an invalid NullString", func() {
		result := utils.NullStringPtr(sql.NullString{Valid: false})
		Expect(result).To(BeNil())
	})

	It("returns a pointer to the string for a valid NullString", func() {
		result := utils.NullStringPtr(sql.NullString{String: "hello", Valid: true})
		Expect(result).NotTo(BeNil())
		Expect(*result).To(Equal("hello"))
	})

	It("returns a pointer to an empty string when valid but empty", func() {
		result := utils.NullStringPtr(sql.NullString{String: "", Valid: true})
		Expect(result).NotTo(BeNil())
		Expect(*result).To(Equal(""))
	})
})

var _ = Describe("CapitalizeFirst", func() {
	It("returns an empty string unchanged", func() {
		Expect(utils.CapitalizeFirst("")).To(Equal(""))
	})

	It("capitalizes the first letter of a lowercase string", func() {
		Expect(utils.CapitalizeFirst("hello")).To(Equal("Hello"))
	})

	It("leaves an already-capitalized string unchanged", func() {
		Expect(utils.CapitalizeFirst("Hello")).To(Equal("Hello"))
	})

	It("capitalizes a single character", func() {
		Expect(utils.CapitalizeFirst("a")).To(Equal("A"))
	})

	It("only capitalizes the first character, not the rest", func() {
		Expect(utils.CapitalizeFirst("hELLO")).To(Equal("HELLO"))
	})
})
