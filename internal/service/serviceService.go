package service

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/database"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

type serviceService struct {
	serviceRepo interfaces.IServiceRepo
	tx          *database.Transaction
}

func NewServiceService(db *sql.DB, serviceRepo interfaces.IServiceRepo) interfaces.IServiceService {
	return &serviceService{
		serviceRepo: serviceRepo,
		tx:          database.NewTransaction(db),
	}
}

func (s *serviceService) CreateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error) {
	var result *param.ServiceParam
	err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := p.Validate(); err != nil {
			return err
		}

		if len(p.ServicePackages) == 0 {
			p.ServicePackages = []param.ServicePackageParam{{ServicePackageName: p.ServiceName}}
		}

		created, err := s.serviceRepo.InsertService(ctx, tx, p)
		if err != nil {
			return err
		}

		for _, pkg := range p.ServicePackages {
			pkg.ServiceID = created.ServiceID
			createdPkg, err := s.serviceRepo.InsertServicePackage(ctx, tx, pkg)
			if err != nil {
				return err
			}
			for _, item := range pkg.PackageItems {
				item.ServicePackageID = createdPkg.ServicePackageID
				if err := s.serviceRepo.InsertPackageItem(ctx, tx, item); err != nil {
					return err
				}
			}
			createdPkg.PackageItems = pkg.PackageItems
			created.ServicePackages = append(created.ServicePackages, *createdPkg)
		}
		result = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *serviceService) UpdateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error) {
	var result *param.ServiceParam
	err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := p.Validate(); err != nil {
			return err
		}

		if len(p.ServicePackages) == 0 {
			p.ServicePackages = []param.ServicePackageParam{{ServicePackageName: p.ServiceName}}
		}

		updated, err := s.serviceRepo.UpdateService(ctx, tx, p)
		if err != nil {
			return err
		}

		if err := s.serviceRepo.DeleteServicePackagesByServiceID(ctx, tx, updated.ServiceID); err != nil {
			return err
		}

		for _, pkg := range p.ServicePackages {
			pkg.ServiceID = updated.ServiceID
			createdPkg, err := s.serviceRepo.InsertServicePackage(ctx, tx, pkg)
			if err != nil {
				return err
			}
			for _, item := range pkg.PackageItems {
				item.ServicePackageID = createdPkg.ServicePackageID
				if err := s.serviceRepo.InsertPackageItem(ctx, tx, item); err != nil {
					return err
				}
			}
			createdPkg.PackageItems = pkg.PackageItems
			updated.ServicePackages = append(updated.ServicePackages, *createdPkg)
		}
		result = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *serviceService) DeleteService(ctx context.Context, serviceID int64, businessID int64) error {
	hasBooking, err := s.serviceRepo.HasBookingForService(ctx, serviceID)
	if err != nil {
		return err
	}
	if hasBooking {
		return fmt.Errorf("Deletion disabled - booking exists.")
	}

	return s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		return s.serviceRepo.SoftDeleteService(ctx, tx, serviceID, businessID)
	})
}

func (s *serviceService) GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error) {
	services, err := s.serviceRepo.GetServicesByBusinessID(ctx, businessID)
	if err != nil {
		return nil, err
	}

	for i := range services {
		packages, err := s.serviceRepo.GetServicePackagesByServiceID(ctx, services[i].ServiceID)
		if err != nil {
			return nil, err
		}
		for j := range packages {
			items, err := s.serviceRepo.GetPackageItemsByPackageID(ctx, packages[j].ServicePackageID)
			if err != nil {
				return nil, err
			}
			packages[j].PackageItems = items
		}
		services[i].ServicePackages = packages
	}
	return services, nil
}
