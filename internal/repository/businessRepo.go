package repository

import (
	"context"
	"database/sql"
	"fmt"
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

func (b *businessRepo) UpdateBusinessProfile(ctx context.Context, tx *sql.Tx, param param.BusinessProfileParam) (*param.BusinessProfileParam, error) {
	row := tx.QueryRowContext(ctx, `
		UPDATE business_profiles
		SET
			business_name = $1,
			description = $2,
			address = $3,
			image_url = $4,
			business_contact_number = $5,
			business_email = $6
		WHERE owner_user_id = $7
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
		param.BusinessName,
		param.Description,
		param.Address,
		param.ImageURL,
		param.BusinessContactNumber,
		param.BusinessEmail,
		param.OwnerUserID,
	)

	businessProfile, err := ScanBusinessProfile(row)
	if err != nil {
		return nil, err
	}

	return businessProfile, nil
}

func (b *businessRepo) DeleteBusinessWorkingHours(ctx context.Context, tx *sql.Tx, businessID int64) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM business_working_hours
		WHERE business_id = $1
	`, businessID)
	return err
}

func (b *businessRepo) GetBusinessProfileByOwnerID(ctx context.Context, ownerID int64) (*param.BusinessProfileParam, error) {
	row := b.DB.QueryRowContext(ctx, `
		SELECT
			business_id,
			owner_user_id,
			business_name,
			description,
			address,
			image_url,
			business_contact_number,
			business_email
		FROM business_profiles
		WHERE owner_user_id = $1
	`, ownerID)

	businessProfile, err := ScanBusinessProfile(row)
	if err != nil {
		return nil, err
	}

	return businessProfile, nil
}

func (b *businessRepo) BusinessEmailExists(ctx context.Context, businessEmail string) (bool, error) {
	var exists bool
	err := b.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM business_profiles WHERE business_email = $1)`, businessEmail).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (b *businessRepo) GetBusinessWorkingHours(ctx context.Context, businessID int64) ([]param.WorkingHourParam, error) {
	rows, err := b.DB.QueryContext(ctx, `
		SELECT
			day,
			start_time,
			end_time
		FROM business_working_hours
		WHERE business_id = $1
	`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workingHours []param.WorkingHourParam
	for rows.Next() {
		var wh param.WorkingHourParam
		if err := rows.Scan(&wh.Day, &wh.StartTime, &wh.EndTime); err != nil {
			return nil, err
		}
		workingHours = append(workingHours, wh)
	}

	return workingHours, nil
}

func (b *businessRepo) GetBusinesses(ctx context.Context, search string) ([]param.BusinessProfileParam, error) {
	query := `
		SELECT business_id, owner_user_id, business_name, description, address,
		       image_url, business_contact_number, business_email
		FROM business_profiles
		WHERE 1=1`
	args := []any{}
	if search != "" {
		args = append(args, "%"+search+"%")
		query += fmt.Sprintf(" AND LOWER(business_name) LIKE LOWER($%d)", len(args))
	}
	query += " ORDER BY business_name"

	rows, err := b.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []param.BusinessProfileParam
	for rows.Next() {
		bp, err := ScanBusinessProfile(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *bp)
	}
	return results, rows.Err()
}

func (b *businessRepo) GetBusinessByID(ctx context.Context, businessID int64) (*param.BusinessProfileParam, error) {
	row := b.DB.QueryRowContext(ctx, `
		SELECT business_id, owner_user_id, business_name, description, address,
		       image_url, business_contact_number, business_email
		FROM business_profiles
		WHERE business_id = $1
	`, businessID)
	return ScanBusinessProfile(row)
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
	businessProfile.Address = address.String
	businessProfile.ImageURL = utils.NullStringPtr(imageURL)
	businessProfile.BusinessContactNumber = businessContactNumber.String
	businessProfile.BusinessEmail = businessEmail.String

	return &businessProfile, nil
}
