package resolver

import (
	"context"
	"fmt"

	"fyp/app"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/contexts"
	"fyp/internal/interfaces/mocks"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The analytics dashboard is owner-only (UC-12), and every one of its queries
// gets the business it reads from here rather than from anything the caller
// sends — so this resolver is the whole of that rule. A caller can never name
// a business id, which is what stops one owner asking for another's figures;
// what remains to prove is that a non-owner gets nothing at all.
// UT-040 (Authorization Testing).
var _ = Describe("ownerBusinessID (analytics access)", func() {
	var (
		ctx             context.Context
		businessService *mocks.MockIBusinessService
		resolver        *queryResolver
		owner           *param.AuthUserParam
	)

	BeforeEach(func() {
		businessService = mocks.NewMockIBusinessService(GinkgoT())
		resolver = &queryResolver{&Resolver{App: &app.App{BusinessService: businessService}}}
		owner = &param.AuthUserParam{UserID: 42, Email: "owner@example.com"}
		ctx = contexts.WithUser(context.Background(), owner)
	})

	It("resolves an owner to their own business, never one they were asked for", func() {
		businessService.EXPECT().GetBusinessProfileByOwnerID(ctx, int64(42)).
			Return(&param.BusinessProfileParam{BusinessID: 7, OwnerUserID: 42}, nil).Once()

		businessID, err := resolver.ownerBusinessID(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(businessID).To(Equal(int64(7)))
	})

	It("refuses a caller who owns no business — a staff member or a customer", func() {
		// Both look identical here: whoever they are, they own no business,
		// so there is no business whose analytics they could be shown.
		businessService.EXPECT().GetBusinessProfileByOwnerID(ctx, int64(42)).
			Return(nil, errs.ErrBusinessProfileNotFound).Once()

		businessID, err := resolver.ownerBusinessID(ctx)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Business profile not found"))
		// Zero, not some other business's id — a caller who slipped past the
		// error would still read nothing.
		Expect(businessID).To(BeZero())
	})

	It("refuses a caller who is not signed in at all", func() {
		// No user in the context, so the business is never even looked up.
		businessID, err := resolver.ownerBusinessID(context.Background())

		Expect(err).To(HaveOccurred())
		Expect(businessID).To(BeZero())
	})

	It("surfaces a lookup failure instead of falling back to any business", func() {
		businessService.EXPECT().GetBusinessProfileByOwnerID(ctx, int64(42)).
			Return(nil, fmt.Errorf("db down")).Once()

		businessID, err := resolver.ownerBusinessID(ctx)

		Expect(err).To(HaveOccurred())
		Expect(businessID).To(BeZero())
	})
})
