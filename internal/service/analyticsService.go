package service

import (
	"context"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

type analyticsService struct {
	analyticsRepo interfaces.IAnalyticsRepo
}

func NewAnalyticsService(analyticsRepo interfaces.IAnalyticsRepo) interfaces.IAnalyticsService {
	return &analyticsService{analyticsRepo: analyticsRepo}
}

func (s *analyticsService) GetBookingSummary(ctx context.Context, businessID int64, from, until string) (*param.BookingSummaryParam, error) {
	return s.analyticsRepo.GetBookingSummary(ctx, businessID, from, until)
}

func (s *analyticsService) GetBookingTrend(ctx context.Context, businessID int64, from, until, granularity string) ([]param.BookingTrendPointParam, error) {
	if granularity == "" {
		granularity = "day"
	}
	return s.analyticsRepo.GetBookingTrend(ctx, businessID, from, until, granularity)
}

func (s *analyticsService) GetServicePopularity(ctx context.Context, businessID int64, from, until string) ([]param.ServicePopularityParam, error) {
	return s.analyticsRepo.GetServicePopularity(ctx, businessID, from, until)
}

func (s *analyticsService) GetSlotUtilization(ctx context.Context, businessID int64, from, until, groupBy string) (*param.SlotUtilizationParam, error) {
	return s.analyticsRepo.GetSlotUtilization(ctx, businessID, from, until, groupBy)
}

func (s *analyticsService) GetStaffUtilization(ctx context.Context, businessID int64, from, until string) ([]param.StaffUtilizationParam, error) {
	return s.analyticsRepo.GetStaffUtilization(ctx, businessID, from, until)
}

func (s *analyticsService) GetCustomerRetention(ctx context.Context, businessID int64, from, until string) (*param.CustomerRetentionParam, error) {
	return s.analyticsRepo.GetCustomerRetention(ctx, businessID, from, until)
}

func (s *analyticsService) GetCancellationAnalysis(ctx context.Context, businessID int64, from, until string, staffIDs, serviceIDs []int64) (*param.CancellationAnalysisParam, error) {
	return s.analyticsRepo.GetCancellationAnalysis(ctx, businessID, from, until, staffIDs, serviceIDs)
}

func (s *analyticsService) GetServiceFilterOptions(ctx context.Context, businessID int64) ([]param.ServiceFilterOptionParam, error) {
	return s.analyticsRepo.GetServiceFilterOptions(ctx, businessID)
}

func (s *analyticsService) GetBusinessCreatedAt(ctx context.Context, businessID int64) (string, error) {
	return s.analyticsRepo.GetBusinessCreatedAt(ctx, businessID)
}
