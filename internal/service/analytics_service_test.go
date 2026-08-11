package service_test

import (
	"context"
	"fmt"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/interfaces/mocks"
	"fyp/internal/service"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AnalyticsService", func() {
	var (
		ctx           context.Context
		analyticsRepo *mocks.MockIAnalyticsRepo
		analyticsSvc  interfaces.IAnalyticsService
	)

	BeforeEach(func() {
		ctx = context.Background()
		analyticsRepo = mocks.NewMockIAnalyticsRepo(GinkgoT())
		analyticsSvc = service.NewAnalyticsService(analyticsRepo)
	})

	Describe("GetBookingSummary", func() {
		It("returns the repo's result unchanged", func() {
			expected := &param.BookingSummaryParam{
				TotalBookings:      10,
				ByStatus:           []param.StatusCountParam{{Status: "accepted", Count: 5}},
				ByType:             []param.TypeCountParam{{BookingType: "online", Count: 7}},
				TodayAcceptedCount: 2,
				PendingCount:       3,
			}

			analyticsRepo.EXPECT().
				GetBookingSummary(ctx, int64(1), "2026-01-01", "2026-01-31").
				Return(expected, nil).
				Once()

			result, err := analyticsSvc.GetBookingSummary(ctx, 1, "2026-01-01", "2026-01-31")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("propagates a repo error", func() {
			analyticsRepo.EXPECT().
				GetBookingSummary(ctx, int64(1), "2026-01-01", "2026-01-31").
				Return(nil, fmt.Errorf("db error")).
				Once()

			result, err := analyticsSvc.GetBookingSummary(ctx, 1, "2026-01-01", "2026-01-31")

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetBookingTrend", func() {
		It("defaults an empty granularity to \"day\" before calling the repo", func() {
			expected := []param.BookingTrendPointParam{{Date: "2026-01-01", Count: 4}}

			analyticsRepo.EXPECT().
				GetBookingTrend(ctx, int64(1), "2026-01-01", "2026-01-31", "day").
				Return(expected, nil).
				Once()

			result, err := analyticsSvc.GetBookingTrend(ctx, 1, "2026-01-01", "2026-01-31", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("passes a non-empty granularity through unchanged", func() {
			expected := []param.BookingTrendPointParam{{Date: "2026-01-01", Count: 9}}

			analyticsRepo.EXPECT().
				GetBookingTrend(ctx, int64(1), "2026-01-01", "2026-01-31", "week").
				Return(expected, nil).
				Once()

			result, err := analyticsSvc.GetBookingTrend(ctx, 1, "2026-01-01", "2026-01-31", "week")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("propagates a repo error", func() {
			analyticsRepo.EXPECT().
				GetBookingTrend(ctx, int64(1), "2026-01-01", "2026-01-31", "day").
				Return(nil, fmt.Errorf("db error")).
				Once()

			result, err := analyticsSvc.GetBookingTrend(ctx, 1, "2026-01-01", "2026-01-31", "")

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetServicePopularity", func() {
		It("returns the repo's result unchanged", func() {
			expected := []param.ServicePopularityParam{{ServiceID: 1, ServiceName: "Massage", BookingCount: 6}}

			analyticsRepo.EXPECT().
				GetServicePopularity(ctx, int64(1), "2026-01-01", "2026-01-31").
				Return(expected, nil).
				Once()

			result, err := analyticsSvc.GetServicePopularity(ctx, 1, "2026-01-01", "2026-01-31")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("propagates a repo error", func() {
			analyticsRepo.EXPECT().
				GetServicePopularity(ctx, int64(1), "2026-01-01", "2026-01-31").
				Return(nil, fmt.Errorf("db error")).
				Once()

			result, err := analyticsSvc.GetServicePopularity(ctx, 1, "2026-01-01", "2026-01-31")

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetSlotUtilization", func() {
		It("returns the repo's result unchanged", func() {
			expected := &param.SlotUtilizationParam{
				TotalSlots:  20,
				BookedSlots: 15,
				Sections: []param.UtilizationSectionParam{
					{Subtitle: "2026-01", Points: []param.UtilizationBreakdownParam{{Label: "01", TotalSlots: 5, BookedSlots: 3}}},
				},
			}

			analyticsRepo.EXPECT().
				GetSlotUtilization(ctx, int64(1), "2026-01-01", "2026-01-31", "date").
				Return(expected, nil).
				Once()

			result, err := analyticsSvc.GetSlotUtilization(ctx, 1, "2026-01-01", "2026-01-31", "date")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("propagates a repo error", func() {
			analyticsRepo.EXPECT().
				GetSlotUtilization(ctx, int64(1), "2026-01-01", "2026-01-31", "date").
				Return(nil, fmt.Errorf("db error")).
				Once()

			result, err := analyticsSvc.GetSlotUtilization(ctx, 1, "2026-01-01", "2026-01-31", "date")

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetStaffUtilization", func() {
		It("returns the repo's result unchanged", func() {
			staffID := int64(2)
			expected := []param.StaffUtilizationParam{
				{StaffID: &staffID, StaffName: "Alice", BookedSlots: 10, BookedHours: 20.5},
			}

			analyticsRepo.EXPECT().
				GetStaffUtilization(ctx, int64(1), "2026-01-01", "2026-01-31").
				Return(expected, nil).
				Once()

			result, err := analyticsSvc.GetStaffUtilization(ctx, 1, "2026-01-01", "2026-01-31")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("propagates a repo error", func() {
			analyticsRepo.EXPECT().
				GetStaffUtilization(ctx, int64(1), "2026-01-01", "2026-01-31").
				Return(nil, fmt.Errorf("db error")).
				Once()

			result, err := analyticsSvc.GetStaffUtilization(ctx, 1, "2026-01-01", "2026-01-31")

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetCustomerRetention", func() {
		It("returns the repo's result unchanged", func() {
			expected := &param.CustomerRetentionParam{NewCustomers: 4, ReturningCustomers: 6}

			analyticsRepo.EXPECT().
				GetCustomerRetention(ctx, int64(1), "2026-01-01", "2026-01-31").
				Return(expected, nil).
				Once()

			result, err := analyticsSvc.GetCustomerRetention(ctx, 1, "2026-01-01", "2026-01-31")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("propagates a repo error", func() {
			analyticsRepo.EXPECT().
				GetCustomerRetention(ctx, int64(1), "2026-01-01", "2026-01-31").
				Return(nil, fmt.Errorf("db error")).
				Once()

			result, err := analyticsSvc.GetCustomerRetention(ctx, 1, "2026-01-01", "2026-01-31")

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetCancellationAnalysis", func() {
		It("returns the repo's result unchanged", func() {
			staffID := int64(3)
			staffIDs := []int64{3}
			serviceIDs := []int64{5, 6}
			expected := &param.CancellationAnalysisParam{
				TotalBookings:  10,
				TotalCancelled: 2,
				TotalRejected:  1,
				ByService:      []param.ServiceCancellationParam{{ServiceID: 5, ServiceName: "Massage", Count: 1}},
				ByStaff:        []param.StaffCancellationParam{{StaffID: &staffID, StaffName: "Bob", CustomerCount: 1, StaffCount: 2}},
			}

			analyticsRepo.EXPECT().
				GetCancellationAnalysis(ctx, int64(1), "2026-01-01", "2026-01-31", staffIDs, serviceIDs).
				Return(expected, nil).
				Once()

			result, err := analyticsSvc.GetCancellationAnalysis(ctx, 1, "2026-01-01", "2026-01-31", staffIDs, serviceIDs)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("propagates a repo error", func() {
			staffIDs := []int64{3}
			serviceIDs := []int64{5, 6}

			analyticsRepo.EXPECT().
				GetCancellationAnalysis(ctx, int64(1), "2026-01-01", "2026-01-31", staffIDs, serviceIDs).
				Return(nil, fmt.Errorf("db error")).
				Once()

			result, err := analyticsSvc.GetCancellationAnalysis(ctx, 1, "2026-01-01", "2026-01-31", staffIDs, serviceIDs)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetServiceFilterOptions", func() {
		It("returns the repo's result unchanged", func() {
			expected := []param.ServiceFilterOptionParam{
				{ServiceIDs: []int64{1, 2}, ServiceName: "Massage", Deleted: false},
			}

			analyticsRepo.EXPECT().
				GetServiceFilterOptions(ctx, int64(1)).
				Return(expected, nil).
				Once()

			result, err := analyticsSvc.GetServiceFilterOptions(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("propagates a repo error", func() {
			analyticsRepo.EXPECT().
				GetServiceFilterOptions(ctx, int64(1)).
				Return(nil, fmt.Errorf("db error")).
				Once()

			result, err := analyticsSvc.GetServiceFilterOptions(ctx, 1)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetBusinessCreatedAt", func() {
		It("returns the repo's result unchanged", func() {
			analyticsRepo.EXPECT().
				GetBusinessCreatedAt(ctx, int64(1)).
				Return("2025-01-15", nil).
				Once()

			result, err := analyticsSvc.GetBusinessCreatedAt(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("2025-01-15"))
		})

		It("propagates a repo error", func() {
			analyticsRepo.EXPECT().
				GetBusinessCreatedAt(ctx, int64(1)).
				Return("", fmt.Errorf("db error")).
				Once()

			result, err := analyticsSvc.GetBusinessCreatedAt(ctx, 1)

			Expect(result).To(Equal(""))
			Expect(err).To(MatchError("db error"))
		})
	})
})
