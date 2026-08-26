package service_test

import (
	"context"
	"errors"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/interfaces/mocks"
	"fyp/internal/service"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProfileService", func() {
	var (
		ctx         context.Context
		profileRepo *mocks.MockIProfileRepo
		profileSvc  interfaces.IProfileService
	)

	BeforeEach(func() {
		ctx = context.Background()
		profileRepo = mocks.NewMockIProfileRepo(GinkgoT())
		profileSvc = service.NewProfileService(profileRepo)
	})

	Describe("UpdateProfile", func() {
		It("updates the profile successfully when parameters are valid", func() {
			profileParam := param.ProfileParam{
				UserId:        1,
				Username:      "newname",
				Email:         "new@example.com",
				ContactNumber: "0123456789",
			}

			profileRepo.EXPECT().
				UpdateUser(ctx, profileParam).
				Return(nil).
				Once()

			err := profileSvc.UpdateProfile(ctx, profileParam)

			Expect(err).NotTo(HaveOccurred())
		})

		It("returns validation errors if parameters are invalid", func() {
			invalidParam := param.ProfileParam{
				UserId: 1,
				Email:  "invalid-email",
			}

			err := profileSvc.UpdateProfile(ctx, invalidParam)

			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(errs.ValidationErrors{}))
		})

		It("maps unique violation for email to a validation error", func() {
			profileParam := param.ProfileParam{
				UserId:        1,
				Username:      "name",
				Email:         "duplicate@example.com",
				ContactNumber: "0123456789",
			}
			uniqueErr := newUniqueViolation(database.ConstraintUserEmail)

			profileRepo.EXPECT().
				UpdateUser(ctx, profileParam).
				Return(uniqueErr).
				Once()

			err := profileSvc.UpdateProfile(ctx, profileParam)

			Expect(err).To(Equal(errs.ValidationErrors{
				{Field: "email", Message: "This email is already registered"},
			}))
		})

		It("returns generic repository errors", func() {
			profileParam := param.ProfileParam{
				UserId:        1,
				Username:      "name",
				Email:         "test@example.com",
				ContactNumber: "0123456789",
			}
			dbErr := errors.New("connection failed")

			profileRepo.EXPECT().
				UpdateUser(ctx, profileParam).
				Return(dbErr).
				Once()

			err := profileSvc.UpdateProfile(ctx, profileParam)

			Expect(err).To(Equal(dbErr))
		})
	})
})
