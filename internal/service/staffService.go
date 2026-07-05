package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
	"golang.org/x/crypto/bcrypt"
)

type staffService struct {
	staffRepo    interfaces.IStaffRepo
	businessRepo interfaces.IBusinessRepo
	authRepo     interfaces.IAuthRepo
	tx           *database.Transaction
	emailService interfaces.IEmailService
}

func NewStaffService(db *sql.DB, staffRepo interfaces.IStaffRepo, businessRepo interfaces.IBusinessRepo, authRepo interfaces.IAuthRepo, emailService interfaces.IEmailService) interfaces.IStaffService {
	return &staffService{
		staffRepo:    staffRepo,
		businessRepo: businessRepo,
		authRepo:     authRepo,
		tx:           database.NewTransaction(db),
		emailService: emailService,
	}
}

func timeOfDay(t time.Time) time.Time {
	return time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}

func validateStaffWithinBusinessHours(staffHours, businessHours []param.WorkingHourParam) errs.ValidationErrors {
	businessByDay := make(map[string][]param.WorkingHourParam)
	for _, wh := range businessHours {
		businessByDay[wh.Day] = append(businessByDay[wh.Day], wh)
	}

	var validationErrs errs.ValidationErrors
	for i, swh := range staffHours {
		sStart := timeOfDay(swh.StartTime)
		sEnd := timeOfDay(swh.EndTime)
		contained := false
		for _, bwh := range businessByDay[swh.Day] {
			bStart := timeOfDay(bwh.StartTime)
			bEnd := timeOfDay(bwh.EndTime)
			if !sStart.Before(bStart) && !sEnd.After(bEnd) {
				contained = true
				break
			}
		}
		if !contained {
			validationErrs = append(validationErrs, errs.ValidationError{
				Field:   fmt.Sprintf("workingHours[%d]", i),
				Message: "Staff working hours must be within business working hours on the same day",
			})
		}
	}
	if len(validationErrs) > 0 {
		return validationErrs
	}
	return nil
}

func (s *staffService) RegisterStaff(ctx context.Context, ownerParam *param.BusinessProfileParam, staffParam param.StaffParam, userParam param.AuthUserParam) error {
	if err := staffParam.ValidateRegisterStaff(); err != nil {
		return err
	}

	var validationErrs errs.ValidationErrors
	validationErrs = append(validationErrs, validateStaffWithinBusinessHours(staffParam.WorkingHours, ownerParam.WorkingHours)...)

	if staffParam.StaffEmail == userParam.Email {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   "staffEmail",
			Message: "Staff email cannot be the same as your email.",
		})
	}
	businessEmailExists, err := s.businessRepo.BusinessEmailExists(ctx, staffParam.StaffEmail)
	if err != nil {
		return err
	}
	if businessEmailExists {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   "staffEmail",
			Message: "This email address is already being used as the business email.",
		})
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}
	var tempPassword string
	var createdNewUser bool
	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		userProfile, err := s.authRepo.GetUser(ctx, staffParam.StaffEmail)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				tempPassword, err = generateTemporaryPassword()
				if err != nil {
					return err
				}
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
				if err != nil {
					return err
				}
				userProfile, err = s.authRepo.SignUpTx(ctx, tx, param.SignUpParam{
					Username:          staffParam.StaffName,
					Email:             staffParam.StaffEmail,
					ContactNumber:     staffParam.StaffContactNumber,
					Password:          string(hashedPassword),
					MustResetPassword: true,
				})
				if err != nil {
					return err
				}
				createdNewUser = true
			} else {
				return err
			}

		}
		staffParam.UserID = userProfile.UserID
		staffID, err := s.staffRepo.InsertStaff(ctx, tx, staffParam)
		if err != nil {
			if database.IsUniqueViolation(err, "staff_user_id_key") {
				return errs.ValidationErrors{
					{Field: "staffEmail", Message: "A staff profile already exists for this email."},
				}
			}
			return err
		}
		staffParam.StaffID = *staffID
		err = s.staffRepo.InsertStaffWorkingHours(ctx, tx, staffParam)
		return err
	})
	if err != nil {
		return err
	}
	if createdNewUser {
		err = s.emailService.SendStaffWelcomeEmail(staffParam.StaffEmail, tempPassword)
		if err != nil {
			log.Errorf("Failed to send welcome email to %s: %v", staffParam.StaffEmail, err)
			return fmt.Errorf("staff created but welcome email could not be sent. Staff tempprary password: %s", tempPassword)
		}
	}
	return nil
}

func (s *staffService) GetStaffProfileByUserID(ctx context.Context, userID int64) (*param.StaffParam, error) {
	staffProfile, err := s.staffRepo.GetStaffByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	hours, err := s.staffRepo.GetStaffWorkingHours(ctx, staffProfile.StaffID)
	if err != nil {
		return nil, err
	}
	staffProfile.WorkingHours = hours
	return staffProfile, nil
}

func (s *staffService) GetStaffByBusinessID(ctx context.Context, businessID int64) ([]param.StaffParam, error) {
	staffList, err := s.staffRepo.GetStaffByBusinessID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	for i := range staffList {
		hours, err := s.staffRepo.GetStaffWorkingHours(ctx, staffList[i].StaffID)
		if err != nil {
			return nil, err
		}
		staffList[i].WorkingHours = hours
	}
	return staffList, nil
}

func (s *staffService) UpdateStaff(ctx context.Context, p param.StaffParam) (*param.StaffParam, error) {
	if err := p.ValidateUpdateStaff(); err != nil {
		return nil, err
	}

	var result *param.StaffParam
	err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		current, err := s.staffRepo.GetStaffByIDTx(ctx, tx, p.StaffID, p.BusinessID)
		if err != nil {
			return err
		}

		newEmail := strings.TrimSpace(p.StaffEmail)
		if !strings.EqualFold(newEmail, current.StaffEmail) {
			if err := s.updateStaffEmail(ctx, tx, current, newEmail); err != nil {
				return err
			}
		}

		updated, err := s.staffRepo.UpdateStaff(ctx, tx, p)
		if err != nil {
			return err
		}
		result = updated
		return nil
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ValidationErrors{{Field: "staffId", Message: "Staff profile not found."}}
		}
		return nil, err
	}
	return result, nil
}

// updateStaffEmail changes the linked account's email. It is only permitted
// before the staff has logged in for the first time (must_reset_password = true),
// since after that the email identifies an active account.
func (s *staffService) updateStaffEmail(ctx context.Context, tx *sql.Tx, current *param.StaffParam, newEmail string) error {
	if !current.MustResetPassword {
		return errs.ValidationErrors{{
			Field:   "staffEmail",
			Message: "Email can only be changed before the staff logs in for the first time.",
		}}
	}

	businessEmailExists, err := s.businessRepo.BusinessEmailExists(ctx, newEmail)
	if err != nil {
		return err
	}
	if businessEmailExists {
		return errs.ValidationErrors{{Field: "staffEmail", Message: "This email address is already being used as a business email."}}
	}

	if err := s.authRepo.UpdateUserEmailTx(ctx, tx, current.UserID, newEmail); err != nil {
		if database.IsUniqueViolation(err, "users_email_key") {
			return errs.ValidationErrors{{Field: "staffEmail", Message: "This email address is already in use."}}
		}
		return err
	}
	return nil
}

func (s *staffService) DeleteStaff(ctx context.Context, staffID int64, businessID int64) error {
	hasBooking, err := s.staffRepo.HasBookingForStaff(ctx, staffID)
	if err != nil {
		return err
	}
	if hasBooking {
		return fmt.Errorf("Deletion disabled - booking exists.")
	}

	return s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		return s.staffRepo.SoftDeleteStaff(ctx, tx, staffID, businessID)
	})
}

func generateTemporaryPassword() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
