package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/interfaces/mocks"
	"fyp/internal/service"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("StaffService working hours", func() {
	var (
		ctx             context.Context
		db              *sql.DB
		dbMock          sqlmock.Sqlmock
		staffRepo       *mocks.MockIStaffRepo
		businessRepo    *mocks.MockIBusinessRepo
		authRepo        *mocks.MockIAuthRepo
		serviceSlotRepo *mocks.MockIServiceSlotRepo
		bookingRepo     *mocks.MockIBookingRepo
		emailService    *mocks.MockIEmailService
		staffSvc        interfaces.IStaffService
		businessID      int64
		staffID         int64
		businessHours   []param.WorkingHourParam
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		staffRepo = mocks.NewMockIStaffRepo(GinkgoT())
		businessRepo = mocks.NewMockIBusinessRepo(GinkgoT())
		authRepo = mocks.NewMockIAuthRepo(GinkgoT())
		serviceSlotRepo = mocks.NewMockIServiceSlotRepo(GinkgoT())
		bookingRepo = mocks.NewMockIBookingRepo(GinkgoT())
		emailService = mocks.NewMockIEmailService(GinkgoT())
		staffSvc = service.NewStaffService(db, staffRepo, businessRepo, authRepo, serviceSlotRepo, bookingRepo, emailService)

		businessID = 1
		staffID = 5
		businessHours = []param.WorkingHourParam{
			{Day: "monday", StartTime: time.Date(0, 1, 1, 8, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 20, 0, 0, 0, time.UTC)},
		}
	})

	AfterEach(func() {
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("GetStaffHoursConflicts", func() {
		It("returns only the booked slots that fall outside the proposed hours", func() {
			newHours := []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)},
			}
			bookedOutside := param.AssignedSlotParam{
				ServiceSlotID: 10, Date: "2026-08-03" /* Monday */, HasBooking: true,
				StartTime: time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 15, 0, 0, 0, time.UTC),
			}
			notBookedOutside := param.AssignedSlotParam{
				ServiceSlotID: 11, Date: "2026-08-03", HasBooking: false,
				StartTime: time.Date(0, 1, 1, 16, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
			}
			insideHours := param.AssignedSlotParam{
				ServiceSlotID: 12, Date: "2026-08-03", HasBooking: true,
				StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC),
			}
			serviceSlotRepo.EXPECT().GetFutureAssignedSlotWindows(ctx, staffID, mock.AnythingOfType("string")).
				Return([]param.AssignedSlotParam{bookedOutside, notBookedOutside, insideHours}, nil).Once()

			full := &param.ServiceSlotParam{ServiceSlotID: 10, Date: "2026-08-03"}
			serviceSlotRepo.EXPECT().GetServiceSlotByID(ctx, int64(10), businessID).Return(full, nil).Once()

			result, err := staffSvc.GetStaffHoursConflicts(ctx, businessID, staffID, newHours)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ServiceSlotID).To(Equal(int64(10)))
		})
	})

	Describe("UpdateStaffWorkingHours", func() {
		It("returns a validation error for an empty schedule without touching the repo", func() {
			result, err := staffSvc.UpdateStaffWorkingHours(ctx, businessID, staffID, nil, nil)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("rejects hours outside the business's own hours", func() {
			newHours := []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 6, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)},
			}
			businessRepo.EXPECT().GetBusinessByID(ctx, businessID).
				Return(&param.BusinessProfileParam{BusinessID: businessID, WorkingHours: businessHours}, nil).Once()

			result, err := staffSvc.UpdateStaffWorkingHours(ctx, businessID, staffID, newHours, nil)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("rolls back when a booked slot outside the new hours has no reassignment", func() {
			newHours := []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)},
			}
			businessRepo.EXPECT().GetBusinessByID(ctx, businessID).
				Return(&param.BusinessProfileParam{BusinessID: businessID, WorkingHours: businessHours}, nil).Once()

			booked := param.AssignedSlotParam{
				ServiceSlotID: 10, Date: "2026-08-03", HasBooking: true,
				StartTime: time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 15, 0, 0, 0, time.UTC),
			}
			serviceSlotRepo.EXPECT().GetFutureAssignedSlotWindows(ctx, staffID, mock.AnythingOfType("string")).
				Return([]param.AssignedSlotParam{booked}, nil).Once()

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).
				Return(&param.StaffParam{StaffID: staffID, BusinessID: businessID}, nil).Once()
			dbMock.ExpectRollback()

			result, err := staffSvc.UpdateStaffWorkingHours(ctx, businessID, staffID, newHours, nil)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("reassignments"))
		})

		It("reassigns the booked slot, unassigns the non-booked one, and saves the new hours", func() {
			newHours := []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)},
			}
			businessRepo.EXPECT().GetBusinessByID(ctx, businessID).
				Return(&param.BusinessProfileParam{BusinessID: businessID, WorkingHours: businessHours}, nil).Once()

			replacement := int64(8)
			bookedStart := time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)
			bookedEnd := time.Date(0, 1, 1, 15, 0, 0, 0, time.UTC)
			booked := param.AssignedSlotParam{ServiceSlotID: 10, Date: "2026-08-03", HasBooking: true, StartTime: bookedStart, EndTime: bookedEnd}
			notBooked := param.AssignedSlotParam{
				ServiceSlotID: 11, Date: "2026-08-03", HasBooking: false,
				StartTime: time.Date(0, 1, 1, 16, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
			}
			serviceSlotRepo.EXPECT().GetFutureAssignedSlotWindows(ctx, staffID, mock.AnythingOfType("string")).
				Return([]param.AssignedSlotParam{booked, notBooked}, nil).Once()

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).
				Return(&param.StaffParam{StaffID: staffID, BusinessID: businessID}, nil).Once()

			serviceSlotRepo.EXPECT().StaffBelongsToBusiness(ctx, replacement, businessID).Return(true, nil).Once()
			serviceSlotRepo.EXPECT().StaffCoversTime(ctx, replacement, mock.Anything, bookedStart, bookedEnd).Return(true, nil).Once()
			serviceSlotRepo.EXPECT().ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(10), businessID, &replacement).Return(nil).Once()
			bc := &param.BookingContextParam{BookingID: 400, CustomerEmail: "cust@example.com", CustomerName: "Cust", BusinessName: "Salon", WhenText: "Aug 3 14:00"}
			bookingRepo.EXPECT().GetBookingContextForSlot(ctx, int64(10)).Return(bc, nil).Once()
			serviceSlotRepo.EXPECT().ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(11), businessID, (*int64)(nil)).Return(nil).Once()

			staffRepo.EXPECT().DeleteStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), staffID).Return(nil).Once()
			staffRepo.EXPECT().
				InsertStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.StaffParam) bool {
					return p.StaffID == staffID && len(p.WorkingHours) == 1
				})).
				Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().SendStaffReassignedEmail("cust@example.com", "Cust", "Salon", "Aug 3 14:00").Return(nil).Once()

			result, err := staffSvc.UpdateStaffWorkingHours(ctx, businessID, staffID, newHours, []param.SlotReassignmentParam{
				{ServiceSlotID: 10, StaffID: &replacement},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.WorkingHours).To(Equal(newHours))
		})
	})
})

var _ = Describe("StaffService staff management", func() {
	var (
		ctx             context.Context
		db              *sql.DB
		dbMock          sqlmock.Sqlmock
		staffRepo       *mocks.MockIStaffRepo
		businessRepo    *mocks.MockIBusinessRepo
		authRepo        *mocks.MockIAuthRepo
		serviceSlotRepo *mocks.MockIServiceSlotRepo
		bookingRepo     *mocks.MockIBookingRepo
		emailService    *mocks.MockIEmailService
		staffSvc        interfaces.IStaffService
		businessID      int64
		staffID         int64
		userID          int64
		ownerParam      *param.BusinessProfileParam
		validStaff      param.StaffParam
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		staffRepo = mocks.NewMockIStaffRepo(GinkgoT())
		businessRepo = mocks.NewMockIBusinessRepo(GinkgoT())
		authRepo = mocks.NewMockIAuthRepo(GinkgoT())
		serviceSlotRepo = mocks.NewMockIServiceSlotRepo(GinkgoT())
		bookingRepo = mocks.NewMockIBookingRepo(GinkgoT())
		emailService = mocks.NewMockIEmailService(GinkgoT())
		staffSvc = service.NewStaffService(db, staffRepo, businessRepo, authRepo, serviceSlotRepo, bookingRepo, emailService)

		businessID = 1
		staffID = 5
		userID = 42

		ownerParam = &param.BusinessProfileParam{
			BusinessID: businessID,
			WorkingHours: []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 8, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 20, 0, 0, 0, time.UTC)},
			},
		}

		validStaff = param.StaffParam{
			BusinessID:          businessID,
			StaffName:           "Jane Doe",
			StaffEmail:          "jane@example.com",
			StaffContactNumber:  "0123456789",
			Position:            "Stylist",
			Password:            "password123",
			WorkingHours: []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC)},
			},
		}
	})

	AfterEach(func() {
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("RegisterStaff", func() {
		It("returns a validation error without touching any repo when the staff param is invalid", func() {
			invalid := validStaff
			invalid.StaffName = ""

			err := staffSvc.RegisterStaff(ctx, ownerParam, invalid, param.AuthUserParam{Email: "owner@example.com"})
			Expect(err).To(HaveOccurred())
			_, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
		})

		It("rejects a staff email matching the owner's own email", func() {
			businessRepo.EXPECT().BusinessEmailExists(ctx, validStaff.StaffEmail).Return(false, nil).Once()

			err := staffSvc.RegisterStaff(ctx, ownerParam, validStaff, param.AuthUserParam{Email: validStaff.StaffEmail})
			Expect(err).To(HaveOccurred())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("staffEmail"))
		})

		It("rejects staff working hours outside business working hours", func() {
			outside := validStaff
			outside.WorkingHours = []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 6, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)},
			}
			businessRepo.EXPECT().BusinessEmailExists(ctx, outside.StaffEmail).Return(false, nil).Once()

			err := staffSvc.RegisterStaff(ctx, ownerParam, outside, param.AuthUserParam{Email: "owner@example.com"})
			Expect(err).To(HaveOccurred())
			_, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
		})

		It("rejects a staff email already used as a business email", func() {
			businessRepo.EXPECT().BusinessEmailExists(ctx, validStaff.StaffEmail).Return(true, nil).Once()

			err := staffSvc.RegisterStaff(ctx, ownerParam, validStaff, param.AuthUserParam{Email: "owner@example.com"})
			Expect(err).To(HaveOccurred())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("staffEmail"))
		})

		It("propagates an error from BusinessEmailExists", func() {
			businessRepo.EXPECT().BusinessEmailExists(ctx, validStaff.StaffEmail).Return(false, fmt.Errorf("db down")).Once()

			err := staffSvc.RegisterStaff(ctx, ownerParam, validStaff, param.AuthUserParam{Email: "owner@example.com"})
			Expect(err).To(MatchError("db down"))
		})

		It("creates a new user and staff profile, then sends a welcome email, when no existing user is found", func() {
			businessRepo.EXPECT().BusinessEmailExists(ctx, validStaff.StaffEmail).Return(false, nil).Once()

			dbMock.ExpectBegin()
			authRepo.EXPECT().GetUser(ctx, validStaff.StaffEmail).Return(nil, sql.ErrNoRows).Once()
			authRepo.EXPECT().SignUpTx(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.SignUpParam) bool {
				return p.Email == validStaff.StaffEmail && p.MustResetPassword
			})).Return(&param.AuthUserParam{UserID: userID}, nil).Once()

			newStaffID := int64(77)
			staffRepo.EXPECT().InsertStaff(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.StaffParam) bool {
				return p.UserID == userID
			})).Return(&newStaffID, nil).Once()
			staffRepo.EXPECT().DeleteStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), newStaffID).Return(nil).Once()
			staffRepo.EXPECT().InsertStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.StaffParam) bool {
				return p.StaffID == newStaffID
			})).Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().SendStaffWelcomeEmail(validStaff.StaffEmail).Return(nil).Once()

			err := staffSvc.RegisterStaff(ctx, ownerParam, validStaff, param.AuthUserParam{Email: "owner@example.com"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("reuses an existing user and does not send a welcome email", func() {
			businessRepo.EXPECT().BusinessEmailExists(ctx, validStaff.StaffEmail).Return(false, nil).Once()

			dbMock.ExpectBegin()
			authRepo.EXPECT().GetUser(ctx, validStaff.StaffEmail).Return(&param.AuthUserParam{UserID: userID}, nil).Once()

			newStaffID := int64(78)
			staffRepo.EXPECT().InsertStaff(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.StaffParam) bool {
				return p.UserID == userID
			})).Return(&newStaffID, nil).Once()
			staffRepo.EXPECT().DeleteStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), newStaffID).Return(nil).Once()
			staffRepo.EXPECT().InsertStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), mock.AnythingOfType("param.StaffParam")).Return(nil).Once()
			dbMock.ExpectCommit()

			err := staffSvc.RegisterStaff(ctx, ownerParam, validStaff, param.AuthUserParam{Email: "owner@example.com"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("rolls back and returns a validation error when the staff already has a profile", func() {
			businessRepo.EXPECT().BusinessEmailExists(ctx, validStaff.StaffEmail).Return(false, nil).Once()

			dbMock.ExpectBegin()
			authRepo.EXPECT().GetUser(ctx, validStaff.StaffEmail).Return(&param.AuthUserParam{UserID: userID}, nil).Once()
			staffRepo.EXPECT().InsertStaff(ctx, mock.AnythingOfType("*sql.Tx"), mock.AnythingOfType("param.StaffParam")).
				Return(nil, newUniqueViolation("staff_user_id_key")).Once()
			dbMock.ExpectRollback()

			err := staffSvc.RegisterStaff(ctx, ownerParam, validStaff, param.AuthUserParam{Email: "owner@example.com"})
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("staffEmail"))
		})

		It("rolls back the transaction when GetUser fails with an unexpected error", func() {
			businessRepo.EXPECT().BusinessEmailExists(ctx, validStaff.StaffEmail).Return(false, nil).Once()

			dbMock.ExpectBegin()
			authRepo.EXPECT().GetUser(ctx, validStaff.StaffEmail).Return(nil, fmt.Errorf("connection reset")).Once()
			dbMock.ExpectRollback()

			err := staffSvc.RegisterStaff(ctx, ownerParam, validStaff, param.AuthUserParam{Email: "owner@example.com"})
			Expect(err).To(MatchError("connection reset"))
		})

		It("returns a descriptive error when the welcome email fails to send", func() {
			businessRepo.EXPECT().BusinessEmailExists(ctx, validStaff.StaffEmail).Return(false, nil).Once()

			dbMock.ExpectBegin()
			authRepo.EXPECT().GetUser(ctx, validStaff.StaffEmail).Return(nil, sql.ErrNoRows).Once()
			authRepo.EXPECT().SignUpTx(ctx, mock.AnythingOfType("*sql.Tx"), mock.AnythingOfType("param.SignUpParam")).
				Return(&param.AuthUserParam{UserID: userID}, nil).Once()

			newStaffID := int64(79)
			staffRepo.EXPECT().InsertStaff(ctx, mock.AnythingOfType("*sql.Tx"), mock.AnythingOfType("param.StaffParam")).Return(&newStaffID, nil).Once()
			staffRepo.EXPECT().DeleteStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), newStaffID).Return(nil).Once()
			staffRepo.EXPECT().InsertStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), mock.AnythingOfType("param.StaffParam")).Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().SendStaffWelcomeEmail(validStaff.StaffEmail).Return(fmt.Errorf("smtp down")).Once()

			err := staffSvc.RegisterStaff(ctx, ownerParam, validStaff, param.AuthUserParam{Email: "owner@example.com"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("welcome email"))
		})
	})

	Describe("GetStaffProfileByUserID", func() {
		It("returns the staff profile with its working hours attached", func() {
			profile := &param.StaffParam{StaffID: staffID, UserID: userID, StaffName: "Jane Doe"}
			hours := []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC)},
			}
			staffRepo.EXPECT().GetStaffByUserID(ctx, userID).Return(profile, nil).Once()
			staffRepo.EXPECT().GetStaffWorkingHours(ctx, staffID).Return(hours, nil).Once()

			result, err := staffSvc.GetStaffProfileByUserID(ctx, userID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.WorkingHours).To(Equal(hours))
		})

		It("propagates an error when the staff profile is not found", func() {
			staffRepo.EXPECT().GetStaffByUserID(ctx, userID).Return(nil, sql.ErrNoRows).Once()

			result, err := staffSvc.GetStaffProfileByUserID(ctx, userID)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError(sql.ErrNoRows))
		})

		It("propagates an error from GetStaffWorkingHours", func() {
			profile := &param.StaffParam{StaffID: staffID, UserID: userID}
			staffRepo.EXPECT().GetStaffByUserID(ctx, userID).Return(profile, nil).Once()
			staffRepo.EXPECT().GetStaffWorkingHours(ctx, staffID).Return(nil, fmt.Errorf("db error")).Once()

			result, err := staffSvc.GetStaffProfileByUserID(ctx, userID)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetStaffByBusinessID", func() {
		It("returns every staff member with their working hours attached", func() {
			staffList := []param.StaffParam{
				{StaffID: 1, StaffName: "Alice"},
				{StaffID: 2, StaffName: "Bob"},
			}
			hours1 := []param.WorkingHourParam{{Day: "monday"}}
			hours2 := []param.WorkingHourParam{{Day: "tuesday"}}
			staffRepo.EXPECT().GetStaffByBusinessID(ctx, businessID).Return(staffList, nil).Once()
			staffRepo.EXPECT().GetStaffWorkingHours(ctx, int64(1)).Return(hours1, nil).Once()
			staffRepo.EXPECT().GetStaffWorkingHours(ctx, int64(2)).Return(hours2, nil).Once()

			result, err := staffSvc.GetStaffByBusinessID(ctx, businessID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].WorkingHours).To(Equal(hours1))
			Expect(result[1].WorkingHours).To(Equal(hours2))
		})

		It("propagates an error from GetStaffByBusinessID", func() {
			staffRepo.EXPECT().GetStaffByBusinessID(ctx, businessID).Return(nil, fmt.Errorf("db error")).Once()

			result, err := staffSvc.GetStaffByBusinessID(ctx, businessID)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})

		It("propagates an error when fetching working hours for one of the staff fails", func() {
			staffList := []param.StaffParam{{StaffID: 1, StaffName: "Alice"}}
			staffRepo.EXPECT().GetStaffByBusinessID(ctx, businessID).Return(staffList, nil).Once()
			staffRepo.EXPECT().GetStaffWorkingHours(ctx, int64(1)).Return(nil, fmt.Errorf("hours db error")).Once()

			result, err := staffSvc.GetStaffByBusinessID(ctx, businessID)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("hours db error"))
		})
	})

	Describe("UpdateStaff", func() {
		var updateParam param.StaffParam

		BeforeEach(func() {
			updateParam = param.StaffParam{
				StaffID:            staffID,
				BusinessID:         businessID,
				StaffName:          "Jane Updated",
				StaffEmail:         "jane@example.com",
				StaffContactNumber: "0123456789",
				Position:           "Senior Stylist",
			}
		})

		It("returns a validation error without starting a transaction when the param is invalid", func() {
			invalid := updateParam
			invalid.Position = ""

			result, err := staffSvc.UpdateStaff(ctx, invalid)
			Expect(result).To(BeNil())
			_, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
		})

		It("updates staff details without touching the email when the email is unchanged", func() {
			current := &param.StaffParam{StaffID: staffID, BusinessID: businessID, StaffEmail: "jane@example.com", MustResetPassword: true}
			updated := &param.StaffParam{StaffID: staffID, BusinessID: businessID, StaffName: "Jane Updated"}

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).Return(current, nil).Once()
			staffRepo.EXPECT().UpdateStaff(ctx, mock.AnythingOfType("*sql.Tx"), updateParam).Return(updated, nil).Once()
			dbMock.ExpectCommit()

			result, err := staffSvc.UpdateStaff(ctx, updateParam)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(updated))
		})

		It("updates the linked account email when the email changes and staff has not logged in yet", func() {
			changed := updateParam
			changed.StaffEmail = "new-jane@example.com"
			current := &param.StaffParam{StaffID: staffID, BusinessID: businessID, UserID: userID, StaffEmail: "jane@example.com", MustResetPassword: true}
			updated := &param.StaffParam{StaffID: staffID, BusinessID: businessID, StaffName: "Jane Updated"}

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).Return(current, nil).Once()
			businessRepo.EXPECT().BusinessEmailExists(ctx, "new-jane@example.com").Return(false, nil).Once()
			authRepo.EXPECT().UpdateUserEmailTx(ctx, mock.AnythingOfType("*sql.Tx"), userID, "new-jane@example.com").Return(nil).Once()
			staffRepo.EXPECT().UpdateStaff(ctx, mock.AnythingOfType("*sql.Tx"), changed).Return(updated, nil).Once()
			dbMock.ExpectCommit()

			result, err := staffSvc.UpdateStaff(ctx, changed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(updated))
		})

		It("rolls back when the email changed but the staff has already logged in", func() {
			changed := updateParam
			changed.StaffEmail = "new-jane@example.com"
			current := &param.StaffParam{StaffID: staffID, BusinessID: businessID, UserID: userID, StaffEmail: "jane@example.com", MustResetPassword: false}

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).Return(current, nil).Once()
			dbMock.ExpectRollback()

			result, err := staffSvc.UpdateStaff(ctx, changed)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("staffEmail"))
		})

		It("rolls back when the new email is already used as a business email", func() {
			changed := updateParam
			changed.StaffEmail = "new-jane@example.com"
			current := &param.StaffParam{StaffID: staffID, BusinessID: businessID, UserID: userID, StaffEmail: "jane@example.com", MustResetPassword: true}

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).Return(current, nil).Once()
			businessRepo.EXPECT().BusinessEmailExists(ctx, "new-jane@example.com").Return(true, nil).Once()
			dbMock.ExpectRollback()

			result, err := staffSvc.UpdateStaff(ctx, changed)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("staffEmail"))
		})

		It("rolls back on a unique-violation when the new email is already in use", func() {
			changed := updateParam
			changed.StaffEmail = "new-jane@example.com"
			current := &param.StaffParam{StaffID: staffID, BusinessID: businessID, UserID: userID, StaffEmail: "jane@example.com", MustResetPassword: true}

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).Return(current, nil).Once()
			businessRepo.EXPECT().BusinessEmailExists(ctx, "new-jane@example.com").Return(false, nil).Once()
			authRepo.EXPECT().UpdateUserEmailTx(ctx, mock.AnythingOfType("*sql.Tx"), userID, "new-jane@example.com").
				Return(newUniqueViolation("users_email_key")).Once()
			dbMock.ExpectRollback()

			result, err := staffSvc.UpdateStaff(ctx, changed)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("staffEmail"))
		})

		It("returns a not-found validation error when the staff profile does not exist for this business", func() {
			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).Return(nil, sql.ErrNoRows).Once()
			dbMock.ExpectRollback()

			result, err := staffSvc.UpdateStaff(ctx, updateParam)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("staffId"))
		})

		It("rolls back and propagates an unexpected error from UpdateStaff", func() {
			current := &param.StaffParam{StaffID: staffID, BusinessID: businessID, StaffEmail: "jane@example.com", MustResetPassword: true}

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).Return(current, nil).Once()
			staffRepo.EXPECT().UpdateStaff(ctx, mock.AnythingOfType("*sql.Tx"), updateParam).Return(nil, fmt.Errorf("db write error")).Once()
			dbMock.ExpectRollback()

			result, err := staffSvc.UpdateStaff(ctx, updateParam)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db write error"))
		})
	})

	Describe("DeleteStaff", func() {
		It("soft-deletes the staff member when they have no active bookings", func() {
			staffRepo.EXPECT().HasBookingForStaff(ctx, staffID).Return(false, nil).Once()

			dbMock.ExpectBegin()
			staffRepo.EXPECT().SoftDeleteStaff(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).Return(nil).Once()
			dbMock.ExpectCommit()

			err := staffSvc.DeleteStaff(ctx, staffID, businessID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("refuses to delete a staff member with an active booking", func() {
			staffRepo.EXPECT().HasBookingForStaff(ctx, staffID).Return(true, nil).Once()

			err := staffSvc.DeleteStaff(ctx, staffID, businessID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("booking exists"))
		})

		It("propagates an error from HasBookingForStaff", func() {
			staffRepo.EXPECT().HasBookingForStaff(ctx, staffID).Return(false, fmt.Errorf("db error")).Once()

			err := staffSvc.DeleteStaff(ctx, staffID, businessID)
			Expect(err).To(MatchError("db error"))
		})

		It("rolls back and propagates an error from SoftDeleteStaff", func() {
			staffRepo.EXPECT().HasBookingForStaff(ctx, staffID).Return(false, nil).Once()

			dbMock.ExpectBegin()
			staffRepo.EXPECT().SoftDeleteStaff(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).Return(fmt.Errorf("delete failed")).Once()
			dbMock.ExpectRollback()

			err := staffSvc.DeleteStaff(ctx, staffID, businessID)
			Expect(err).To(MatchError("delete failed"))
		})
	})
})
