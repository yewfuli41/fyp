package interfaces

import (
	"context"
	"fyp/domain/param"
)

type IAnalyticsRepo interface {
	GetBookingSummary(ctx context.Context, businessID int64, from, until string) (*param.BookingSummaryParam, error)
	// GetBookingTrend buckets bookings by appointment date at the given
	// granularity ("day", "week", or "month").
	GetBookingTrend(ctx context.Context, businessID int64, from, until, granularity string) ([]param.BookingTrendPointParam, error)
	GetServicePopularity(ctx context.Context, businessID int64, from, until string) ([]param.ServicePopularityParam, error)
	// GetSlotUtilization buckets by groupBy ("weekday", "date", "month", or "year").
	GetSlotUtilization(ctx context.Context, businessID int64, from, until, groupBy string) (*param.SlotUtilizationParam, error)
	GetStaffUtilization(ctx context.Context, businessID int64, from, until string) ([]param.StaffUtilizationParam, error)
	GetCustomerRetention(ctx context.Context, businessID int64, from, until string) (*param.CustomerRetentionParam, error)
	// GetCancellationAnalysis narrows to just staffIDs' or serviceIDs'
	// bookings when given (either may be nil/empty for "any").
	GetCancellationAnalysis(ctx context.Context, businessID int64, from, until string, staffIDs, serviceIDs []int64) (*param.CancellationAnalysisParam, error)
	// GetServiceFilterOptions lists both active and soft-deleted services,
	// for the cancellation analysis service filter.
	GetServiceFilterOptions(ctx context.Context, businessID int64) ([]param.ServiceFilterOptionParam, error)
	// GetBusinessCreatedAt returns businessID's creation date ("YYYY-MM-DD").
	GetBusinessCreatedAt(ctx context.Context, businessID int64) (string, error)
}

type IAnalyticsService interface {
	GetBookingSummary(ctx context.Context, businessID int64, from, until string) (*param.BookingSummaryParam, error)
	GetBookingTrend(ctx context.Context, businessID int64, from, until, granularity string) ([]param.BookingTrendPointParam, error)
	GetServicePopularity(ctx context.Context, businessID int64, from, until string) ([]param.ServicePopularityParam, error)
	// GetSlotUtilization buckets by groupBy ("weekday", "date", "month", or "year").
	GetSlotUtilization(ctx context.Context, businessID int64, from, until, groupBy string) (*param.SlotUtilizationParam, error)
	GetStaffUtilization(ctx context.Context, businessID int64, from, until string) ([]param.StaffUtilizationParam, error)
	GetCustomerRetention(ctx context.Context, businessID int64, from, until string) (*param.CustomerRetentionParam, error)
	// GetCancellationAnalysis narrows to just staffIDs' or serviceIDs'
	// bookings when given (either may be nil/empty for "any").
	GetCancellationAnalysis(ctx context.Context, businessID int64, from, until string, staffIDs, serviceIDs []int64) (*param.CancellationAnalysisParam, error)
	// GetServiceFilterOptions lists both active and soft-deleted services,
	// for the cancellation analysis service filter.
	GetServiceFilterOptions(ctx context.Context, businessID int64) ([]param.ServiceFilterOptionParam, error)
	// GetBusinessCreatedAt returns businessID's creation date ("YYYY-MM-DD").
	GetBusinessCreatedAt(ctx context.Context, businessID int64) (string, error)
}
