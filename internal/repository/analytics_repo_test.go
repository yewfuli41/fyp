package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"regexp"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AnalyticsRepo", func() {
	var (
		db   *sql.DB
		mock sqlmock.Sqlmock
		repo interfaces.IAnalyticsRepo
		ctx  context.Context
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewAnalyticsRepo(db)
		ctx = context.Background()
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("GetBookingSummary", func() {
		It("returns totals, breakdowns, and the two live figures", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT b.status, COUNT(*)")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
					AddRow("pending", 2).
					AddRow("accepted", 3))

			mock.ExpectQuery(regexp.QuoteMeta("SELECT b.booking_type, COUNT(*)")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows([]string{"booking_type", "count"}).
					AddRow("online", 4).
					AddRow("walk_in", 1))

			mock.ExpectQuery(regexp.QuoteMeta("ss.date = CURRENT_DATE")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

			mock.ExpectQuery(regexp.QuoteMeta("b.status = 'pending'")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

			result, err := repo.GetBookingSummary(ctx, 1, "2026-08-01", "2026-08-31")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalBookings).To(Equal(int64(5)))
			Expect(result.ByStatus).To(HaveLen(2))
			Expect(result.ByStatus[0].Status).To(Equal("pending"))
			Expect(result.ByStatus[0].Count).To(Equal(int64(2)))
			Expect(result.ByType).To(HaveLen(2))
			Expect(result.ByType[1].BookingType).To(Equal("walk_in"))
			Expect(result.TodayAcceptedCount).To(Equal(int64(2)))
			Expect(result.PendingCount).To(Equal(int64(5)))
		})

		It("propagates a DB error from the status breakdown query", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT b.status, COUNT(*)")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnError(errors.New("boom"))

			result, err := repo.GetBookingSummary(ctx, 1, "2026-08-01", "2026-08-31")

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("GetBookingTrend", func() {
		It("returns bucketed points for the given granularity", func() {
			mock.ExpectQuery(regexp.QuoteMeta("date_trunc($4, ss.date::timestamp)")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31", "week").
				WillReturnRows(sqlmock.NewRows([]string{"date", "count"}).
					AddRow("2026-08-03", 5).
					AddRow("2026-08-10", 3))

			results, err := repo.GetBookingTrend(ctx, 1, "2026-08-01", "2026-08-31", "week")

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].Date).To(Equal("2026-08-03"))
			Expect(results[0].Count).To(Equal(int64(5)))
			Expect(results[1].Date).To(Equal("2026-08-10"))
			Expect(results[1].Count).To(Equal(int64(3)))
		})

		It("propagates a DB error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("date_trunc($4, ss.date::timestamp)")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31", "month").
				WillReturnError(errors.New("boom"))

			results, err := repo.GetBookingTrend(ctx, 1, "2026-08-01", "2026-08-31", "month")

			Expect(err).To(HaveOccurred())
			Expect(results).To(BeNil())
		})
	})

	Describe("GetServicePopularity", func() {
		It("returns services ordered by booking count", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT MIN(s.service_id), s.service_name, COUNT(*)")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows([]string{"service_id", "service_name", "count"}).
					AddRow(10, "Massage", 7).
					AddRow(11, "Facial", 3))

			results, err := repo.GetServicePopularity(ctx, 1, "2026-08-01", "2026-08-31")

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].ServiceID).To(Equal(int64(10)))
			Expect(results[0].ServiceName).To(Equal("Massage"))
			Expect(results[0].BookingCount).To(Equal(int64(7)))
			Expect(results[1].ServiceName).To(Equal("Facial"))
		})
	})

	Describe("GetSlotUtilization", func() {
		slotCols := []string{"staff_id", "staff_name", "date", "dow", "start_time", "end_time", "booked"}
		startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)

		It("buckets slots by weekday, using the requested range as-is", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows(slotCols).
					AddRow(nil, nil, "2026-08-10", 1, startTime, endTime, true).
					AddRow(5, "Jane", "2026-08-11", 2, startTime, endTime, false))

			result, err := repo.GetSlotUtilization(ctx, 1, "2026-08-01", "2026-08-31", "weekday")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalSlots).To(Equal(int64(2)))
			Expect(result.BookedSlots).To(Equal(int64(1)))
			Expect(result.Sections).To(HaveLen(1))
			Expect(result.Sections[0].Subtitle).To(Equal(""))
			Expect(result.Sections[0].Points).To(HaveLen(7))

			found := map[string]struct {
				total  int64
				booked int64
			}{}
			for _, p := range result.Sections[0].Points {
				found[p.Label] = struct {
					total  int64
					booked int64
				}{p.TotalSlots, p.BookedSlots}
			}
			Expect(found["monday"].total).To(Equal(int64(1)))
			Expect(found["monday"].booked).To(Equal(int64(1)))
			Expect(found["tuesday"].total).To(Equal(int64(1)))
			Expect(found["tuesday"].booked).To(Equal(int64(0)))
			Expect(found["sunday"].total).To(Equal(int64(0)))
		})

		It("looks up the business's creation date and spans full months when grouping by date", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_business_profiles")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow("2026-08-01"))

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows(slotCols).
					AddRow(nil, nil, "2026-08-15", 6, startTime, endTime, true))

			result, err := repo.GetSlotUtilization(ctx, 1, "2026-08-01", "2026-08-31", "date")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalSlots).To(Equal(int64(1)))
			Expect(result.BookedSlots).To(Equal(int64(1)))
			Expect(result.Sections).To(HaveLen(1))
			Expect(result.Sections[0].Subtitle).To(Equal("2026-08"))
			Expect(result.Sections[0].Points).To(HaveLen(31))

			for _, p := range result.Sections[0].Points {
				if p.Label == "15" {
					Expect(p.TotalSlots).To(Equal(int64(1)))
					Expect(p.BookedSlots).To(Equal(int64(1)))
				}
			}
		})

		It("looks up the business's creation date and spans full years, sectioned by year, when grouping by month", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_business_profiles")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow("2025-11-01"))

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2025-11-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows(slotCols).
					AddRow(nil, nil, "2025-12-05", 5, startTime, endTime, true).
					AddRow(nil, nil, "2026-08-15", 6, startTime, endTime, false))

			result, err := repo.GetSlotUtilization(ctx, 1, "2026-08-01", "2026-08-31", "month")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalSlots).To(Equal(int64(2)))
			Expect(result.BookedSlots).To(Equal(int64(1)))
			Expect(result.Sections).To(HaveLen(2))
			Expect(result.Sections[0].Subtitle).To(Equal("2025"))
			Expect(result.Sections[0].Points).To(HaveLen(12))
			Expect(result.Sections[1].Subtitle).To(Equal("2026"))
			Expect(result.Sections[1].Points).To(HaveLen(12))

			for _, p := range result.Sections[0].Points {
				if p.Label == "12" {
					Expect(p.TotalSlots).To(Equal(int64(1)))
					Expect(p.BookedSlots).To(Equal(int64(1)))
				}
			}
			for _, p := range result.Sections[1].Points {
				if p.Label == "08" {
					Expect(p.TotalSlots).To(Equal(int64(1)))
					Expect(p.BookedSlots).To(Equal(int64(0)))
				}
			}
		})

		It("looks up the business's creation date and spans full calendar years, unsectioned, when grouping by year", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_business_profiles")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow("2024-05-01"))

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2024-05-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows(slotCols).
					AddRow(nil, nil, "2025-06-10", 2, startTime, endTime, true).
					AddRow(nil, nil, "2026-08-01", 6, startTime, endTime, false))

			result, err := repo.GetSlotUtilization(ctx, 1, "2026-08-01", "2026-08-31", "year")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalSlots).To(Equal(int64(2)))
			Expect(result.BookedSlots).To(Equal(int64(1)))
			Expect(result.Sections).To(HaveLen(1))
			Expect(result.Sections[0].Subtitle).To(Equal(""))
			Expect(result.Sections[0].Points).To(HaveLen(3))

			found := map[string]int64{}
			for _, p := range result.Sections[0].Points {
				found[p.Label] = p.TotalSlots
			}
			Expect(found["2024"]).To(Equal(int64(0)))
			Expect(found["2025"]).To(Equal(int64(1)))
			Expect(found["2026"]).To(Equal(int64(1)))
		})

		It("clamps to a single year when the business's creation date is after the selected range's end", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_business_profiles")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow("2027-01-01"))

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2027-01-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows(slotCols))

			result, err := repo.GetSlotUtilization(ctx, 1, "2026-08-01", "2026-08-31", "year")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalSlots).To(Equal(int64(0)))
			Expect(result.Sections).To(HaveLen(1))
			Expect(result.Sections[0].Points).To(HaveLen(1))
			Expect(result.Sections[0].Points[0].Label).To(Equal("2027"))
		})
	})

	Describe("GetStaffUtilization", func() {
		It("aggregates booked slots and hours per staff member, including owner-managed", func() {
			startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime := time.Date(0, 1, 1, 11, 0, 0, 0, time.UTC)
			slotCols := []string{"staff_id", "staff_name", "date", "dow", "start_time", "end_time", "booked"}

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows(slotCols).
					AddRow(nil, nil, "2026-08-10", 1, startTime, endTime, true).
					AddRow(5, "Jane", "2026-08-11", 2, startTime, endTime, true))

			results, err := repo.GetStaffUtilization(ctx, 1, "2026-08-01", "2026-08-31")

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].StaffID).To(BeNil())
			Expect(results[0].StaffName).To(Equal("Owner-managed"))
			Expect(results[0].BookedSlots).To(Equal(int64(1)))
			Expect(results[0].BookedHours).To(Equal(2.0))
			Expect(*results[1].StaffID).To(Equal(int64(5)))
			Expect(results[1].StaffName).To(Equal("Jane"))
			Expect(results[1].BookedHours).To(Equal(2.0))
		})
	})

	Describe("GetCustomerRetention", func() {
		It("returns the new-vs-returning split", func() {
			mock.ExpectQuery(regexp.QuoteMeta("WITH customers AS")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows([]string{"new_customers", "returning_customers"}).AddRow(4, 9))

			result, err := repo.GetCustomerRetention(ctx, 1, "2026-08-01", "2026-08-31")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.NewCustomers).To(Equal(int64(4)))
			Expect(result.ReturningCustomers).To(Equal(int64(9)))
		})

		It("propagates a DB error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("WITH customers AS")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnError(errors.New("boom"))

			result, err := repo.GetCustomerRetention(ctx, 1, "2026-08-01", "2026-08-31")

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("GetCancellationAnalysis", func() {
		It("returns totals, by-service, and by-staff breakdowns with no filter", func() {
			mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FILTER (WHERE b.status = 'cancelled')")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows([]string{"total", "cancelled", "rejected"}).AddRow(10, 6, 4))

			mock.ExpectQuery(regexp.QuoteMeta("GROUP BY s.service_name ORDER BY COUNT(*) DESC")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows([]string{"service_id", "service_name", "count"}).
					AddRow(10, "Massage", 6))

			mock.ExpectQuery(regexp.QuoteMeta("GROUP BY ss.staff_id, st.staff_name")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows([]string{"staff_id", "staff_name", "customer_count", "staff_count"}).
					AddRow(nil, "Owner-managed", 2, 1).
					AddRow(5, "Jane", 1, 2))

			result, err := repo.GetCancellationAnalysis(ctx, 1, "2026-08-01", "2026-08-31", nil, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalBookings).To(Equal(int64(10)))
			Expect(result.TotalCancelled).To(Equal(int64(6)))
			Expect(result.TotalRejected).To(Equal(int64(4)))
			Expect(result.ByService).To(HaveLen(1))
			Expect(result.ByService[0].ServiceName).To(Equal("Massage"))
			Expect(result.ByStaff).To(HaveLen(2))
			Expect(result.ByStaff[0].StaffID).To(BeNil())
			Expect(result.ByStaff[0].StaffName).To(Equal("Owner-managed"))
			Expect(*result.ByStaff[1].StaffID).To(Equal(int64(5)))
		})

		It("narrows every query by staffIDs and serviceIDs when given", func() {
			staffIDs := []int64{5}
			serviceIDs := []int64{10}

			mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FILTER (WHERE b.status = 'cancelled')")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31", pq.Array(staffIDs), pq.Array(serviceIDs)).
				WillReturnRows(sqlmock.NewRows([]string{"total", "cancelled", "rejected"}).AddRow(3, 2, 1))

			mock.ExpectQuery(regexp.QuoteMeta("GROUP BY s.service_name ORDER BY COUNT(*) DESC")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31", pq.Array(staffIDs), pq.Array(serviceIDs)).
				WillReturnRows(sqlmock.NewRows([]string{"service_id", "service_name", "count"}).
					AddRow(10, "Massage", 3))

			mock.ExpectQuery(regexp.QuoteMeta("GROUP BY ss.staff_id, st.staff_name")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31", pq.Array(staffIDs), pq.Array(serviceIDs)).
				WillReturnRows(sqlmock.NewRows([]string{"staff_id", "staff_name", "customer_count", "staff_count"}).
					AddRow(5, "Jane", 1, 2))

			result, err := repo.GetCancellationAnalysis(ctx, 1, "2026-08-01", "2026-08-31", staffIDs, serviceIDs)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalBookings).To(Equal(int64(3)))
			Expect(result.ByService).To(HaveLen(1))
			Expect(result.ByStaff).To(HaveLen(1))
			Expect(*result.ByStaff[0].StaffID).To(Equal(int64(5)))
		})

		It("propagates a DB error from the totals query", func() {
			mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FILTER (WHERE b.status = 'cancelled')")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnError(errors.New("boom"))

			result, err := repo.GetCancellationAnalysis(ctx, 1, "2026-08-01", "2026-08-31", nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("GetServiceFilterOptions", func() {
		It("returns services grouped by name, including soft-deleted ones", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_services")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"service_name", "service_ids", "deleted"}).
					AddRow("Massage", "{10,12}", false).
					AddRow("Facial", "{11}", true))

			results, err := repo.GetServiceFilterOptions(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].ServiceName).To(Equal("Massage"))
			Expect(results[0].ServiceIDs).To(Equal([]int64{10, 12}))
			Expect(results[0].Deleted).To(BeFalse())
			Expect(results[1].ServiceName).To(Equal("Facial"))
			Expect(results[1].ServiceIDs).To(Equal([]int64{11}))
			Expect(results[1].Deleted).To(BeTrue())
		})
	})

	Describe("GetBusinessCreatedAt", func() {
		It("returns the business's creation date", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_business_profiles")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow("2026-01-15"))

			result, err := repo.GetBusinessCreatedAt(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("2026-01-15"))
		})

		It("propagates a DB error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_business_profiles")).
				WithArgs(int64(1)).
				WillReturnError(errors.New("boom"))

			_, err := repo.GetBusinessCreatedAt(ctx, 1)

			Expect(err).To(HaveOccurred())
		})
	})
})
