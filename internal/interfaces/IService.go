package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IServiceRepo interface {
	InsertService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error)
	InsertServiceOption(ctx context.Context, tx *sql.Tx, p param.ServiceOptionParam) (*param.ServiceOptionParam, error)
	InsertServiceOptionItem(ctx context.Context, tx *sql.Tx, p param.ServiceOptionItemParam) error
	UpdateService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error)
	SoftDeleteServiceOption(ctx context.Context, tx *sql.Tx, serviceOptionID int64) error
	SoftDeleteService(ctx context.Context, tx *sql.Tx, serviceID int64, businessID int64) error
	GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error)
	// GetServiceOptionsByServiceID returns every live option for a service.
	GetServiceOptionsByServiceID(ctx context.Context, serviceID int64) ([]param.ServiceOptionParam, error)
	GetServiceOptionItemsByOptionID(ctx context.Context, serviceOptionID int64) ([]param.ServiceOptionItemParam, error)
	GetServiceOptionItemsByOptionIDIncludeDeleted(ctx context.Context, serviceOptionID int64) ([]param.ServiceOptionItemParam, error)
	HasBookingForService(ctx context.Context, serviceID int64) (bool, error)
	// HasBookingForOption reports whether this option has an active booking,
	// so deleting it can't orphan one.
	HasBookingForOption(ctx context.Context, serviceOptionID int64) (bool, error)

	// IsOptionEffectiveOn reports whether serviceOptionID's own validity
	// window covers date.
	IsOptionEffectiveOn(ctx context.Context, serviceOptionID int64, date string) (bool, error)
	// SetOptionWindow resizes an option's own [effective_from, effective_until].
	SetOptionWindow(ctx context.Context, tx *sql.Tx, serviceOptionID int64, from string, until *string) error
	// SetServiceDefaultOption marks serviceOptionID as the service's default
	// option and every other option under it as not-default.
	SetServiceDefaultOption(ctx context.Context, tx *sql.Tx, serviceID int64, serviceOptionID int64) error
	// DropSlotOptionIfExpired soft-deletes a slot's offering of an option if
	// that option's own window no longer covers the slot's date — called once
	// a booking that had held this offering is released via reject/cancel.
	DropSlotOptionIfExpired(ctx context.Context, tx *sql.Tx, slotOptionID int64) error
	// CascadeDeleteServiceSlots soft-deletes every service_slot_options row
	// offering one of this service's options, then soft-deletes any slot left
	// with no remaining active option (from any other service) — called when
	// deleting a whole service, after HasBookingForService has already
	// confirmed nothing under it is actively booked.
	CascadeDeleteServiceSlots(ctx context.Context, tx *sql.Tx, serviceID int64) error
}

type IServiceService interface {
	CreateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error)
	UpdateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error)
	DeleteService(ctx context.Context, serviceID int64, businessID int64) error
	GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error)
}
