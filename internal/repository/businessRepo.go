package repository

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/utils"
)

type businessRepo struct {
	DB *sql.DB
}

func NewBusinessRepo(db *sql.DB) interfaces.IBusinessRepo {
	return &businessRepo{
		DB: db,
	}
}

func (b *businessRepo) InsertBusinessProfile(ctx context.Context, tx *sql.Tx, param param.BusinessProfileParam) (*param.BusinessProfileParam, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO business_profiles (
			owner_user_id,
			business_name,
			description,
			address,
			image_url,
			business_contact_number,
			business_email
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING
			business_id,
			owner_user_id,
			business_name,
			description,
			address,
			image_url,
			business_contact_number,
			business_email
	`,
		param.OwnerUserID,
		param.BusinessName,
		param.Description,
		param.Address,
		param.ImageURL,
		param.BusinessContactNumber,
		param.BusinessEmail,
	)

	businessProfile, err := ScanBusinessProfile(row)
	if err != nil {
		return nil, err
	}

	return businessProfile, nil
}

func (b *businessRepo) InsertBusinessWorkingHours(ctx context.Context, tx *sql.Tx, param param.BusinessProfileParam) error {
	for _, wh := range param.WorkingHours {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO business_working_hours (
				business_id,
				day,
				start_time,
				end_time
			)
			VALUES ($1, $2, $3, $4)
		`,
			param.BusinessID,
			wh.Day,
			wh.StartTime.Format("15:04:05"),
			wh.EndTime.Format("15:04:05"),
		)
		if err != nil {
			return err
		}
	}
	return nil
}
func ScanBusinessProfile(row rowScanner) (*param.BusinessProfileParam, error) {
	var businessProfile param.BusinessProfileParam
	var description sql.NullString
	var address sql.NullString
	var imageURL sql.NullString
	var businessContactNumber sql.NullString
	var businessEmail sql.NullString

	if err := row.Scan(
		&businessProfile.BusinessID,
		&businessProfile.OwnerUserID,
		&businessProfile.BusinessName,
		&description,
		&address,
		&imageURL,
		&businessContactNumber,
		&businessEmail,
	); err != nil {
		return nil, err
	}

	businessProfile.Description = utils.NullStringPtr(description)
	businessProfile.Address = utils.NullStringPtr(address)
	businessProfile.ImageURL = utils.NullStringPtr(imageURL)
	businessProfile.BusinessContactNumber = utils.NullStringPtr(businessContactNumber)
	businessProfile.BusinessEmail = utils.NullStringPtr(businessEmail)

	return &businessProfile, nil
}
