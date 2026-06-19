package param_test

import (
	"fyp/domain/errs"
	"fyp/domain/param"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceParam", func() {
	Describe("Validate", func() {
		It("returns nil for a valid service with no packages", func() {
			p := param.ServiceParam{ServiceName: "Massage"}
			Expect(p.Validate()).To(Succeed())
		})

		It("returns nil for a valid service with packages and items", func() {
			p := param.ServiceParam{
				ServiceName: "Massage",
				ServicePackages: []param.ServicePackageParam{
					{
						ServicePackageName: "Deep Tissue",
						PackageItems:       []param.PackageItemParam{{PackageItemName: "Oil"}},
					},
				},
			}
			Expect(p.Validate()).To(Succeed())
		})

		It("returns error for empty service name", func() {
			p := param.ServiceParam{ServiceName: ""}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "serviceName", Message: "Service name is required"},
			))
		})

		It("returns error for package name exceeding max length", func() {
			p := param.ServiceParam{
				ServiceName: "Massage",
				ServicePackages: []param.ServicePackageParam{
					{ServicePackageName: strings.Repeat("a", 256)},
				},
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)[0].Field).To(Equal("servicePackages[0].servicePackageName"))
		})

		It("returns error for package item name exceeding max length", func() {
			p := param.ServiceParam{
				ServiceName: "Massage",
				ServicePackages: []param.ServicePackageParam{
					{
						ServicePackageName: "Deep Tissue",
						PackageItems: []param.PackageItemParam{
							{PackageItemName: strings.Repeat("b", 256)},
						},
					},
				},
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)[0].Field).To(Equal("servicePackages[0].packageItems[0]"))
		})
	})
})
