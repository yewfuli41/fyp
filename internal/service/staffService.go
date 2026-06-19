package service

import (
	"context"
	"database/sql"
	"fyp/database"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

type staffService struct {
	staffRepo interfaces.IStaffRepo
	tx        *database.Transaction
}

func NewStaffService(db *sql.DB, staffRepo interfaces.IStaffRepo) interfaces.IStaffService {
	return &staffService{
		staffRepo: staffRepo,
		tx:        database.NewTransaction(db),
	}
}

func (s *staffService) RegisterStaff(ctx context.Context, ownerParam *param.BusinessProfileParam, staffParam param.StaffParam) error {
	return nil
}
