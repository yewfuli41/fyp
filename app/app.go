package app

import (
	"database/sql"
	"fyp/config"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"fyp/internal/service"
)

type App struct {
	AuthService        interfaces.IAuthService
	ProfileService     interfaces.IProfileService
	BusinessService    interfaces.IBusinessService
	ServiceService     interfaces.IServiceService
	StaffService       interfaces.IStaffService
	ServiceSlotService interfaces.IServiceSlotService
	EmailService       interfaces.IEmailService
	BookingService     interfaces.IBookingService
	LeaveService       interfaces.ILeaveService
	AnalyticsService   interfaces.IAnalyticsService
}

func NewApp(db *sql.DB, cfg *config.Config) *App {
	authRepo := repository.NewAuthRepo(db)
	authService := service.NewAuthService(authRepo, cfg.Auth)
	profileRepo := repository.NewProfileRepo(db)
	profileService := service.NewProfileService(profileRepo)
	businessRepo := repository.NewBusinessRepo(db)
	serviceSlotRepo := repository.NewServiceSlotRepo(db)
	businessService := service.NewBusinessService(db, businessRepo, serviceSlotRepo)
	serviceRepo := repository.NewServiceRepo(db)
	serviceService := service.NewServiceService(db, serviceRepo)
	emailService := service.NewResendEmailService(cfg.Email)
	bookingRepo := repository.NewBookingRepo(db)
	staffRepo := repository.NewStaffRepo(db)
	staffService := service.NewStaffService(db, staffRepo, businessRepo, authRepo, serviceSlotRepo, bookingRepo, emailService)
	leaveRepo := repository.NewLeaveRepo(db)
	serviceSlotService := service.NewServiceSlotService(db, serviceSlotRepo, serviceRepo, leaveRepo, cfg.ServiceSlot, emailService)
	bookingService := service.NewBookingService(db, bookingRepo, emailService, serviceRepo)
	leaveService := service.NewLeaveService(db, leaveRepo, serviceSlotRepo, bookingRepo, businessRepo, emailService)
	analyticsRepo := repository.NewAnalyticsRepo(db)
	analyticsService := service.NewAnalyticsService(analyticsRepo)
	return &App{
		AuthService:        authService,
		ProfileService:     profileService,
		BusinessService:    businessService,
		ServiceService:     serviceService,
		StaffService:       staffService,
		ServiceSlotService: serviceSlotService,
		EmailService:       emailService,
		BookingService:     bookingService,
		LeaveService:       leaveService,
		AnalyticsService:   analyticsService,
	}
}
