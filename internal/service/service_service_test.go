package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/interfaces/mocks"
	"fyp/internal/service"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("ServiceService", func() {
	var (
		ctx         context.Context
		db          *sql.DB
		dbMock      sqlmock.Sqlmock
		serviceRepo *mocks.MockIServiceRepo
		serviceSvc  interfaces.IServiceService
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		serviceRepo = mocks.NewMockIServiceRepo(GinkgoT())
		serviceSvc = service.NewServiceService(db, serviceRepo)
	})

	AfterEach(func() {
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("CreateService", func() {
		var (
			serviceParam         param.ServiceParam
			expectedServiceParam param.ServiceParam
			createdSvc           *param.ServiceParam
		)

		BeforeEach(func() {
			// A service is always saved with an extra default package (named
			// after the service) and every package always gets a default item
			// (also named after the service) — see ensureDefaultPackage /
			// ensureDefaultPackageItems in serviceService.go.
			serviceParam = param.ServiceParam{
				BusinessID:  1,
				ServiceName: "Massage",
				ServicePackages: []param.ServicePackageParam{
					{
						ServicePackageName: "Deep Tissue",
						PackageItems: []param.PackageItemParam{
							{PackageItemName: "Oil"},
						},
					},
				},
			}
			expectedServiceParam = param.ServiceParam{
				BusinessID:  1,
				ServiceName: "Massage",
				ServicePackages: []param.ServicePackageParam{
					{
						ServicePackageName: "Massage",
						PackageItems: []param.PackageItemParam{
							{PackageItemName: "Massage"},
						},
					},
					{
						ServicePackageName: "Deep Tissue",
						PackageItems: []param.PackageItemParam{
							{PackageItemName: "Massage"},
							{PackageItemName: "Oil"},
						},
					},
				},
			}
			createdSvc = &param.ServiceParam{
				ServiceID:   10,
				BusinessID:  1,
				ServiceName: "Massage",
			}
		})

		It("successfully creates a service with packages and items", func() {
			dbMock.ExpectBegin()

			serviceRepo.EXPECT().
				InsertService(ctx, mock.AnythingOfType("*sql.Tx"), expectedServiceParam).
				Return(createdSvc, nil).
				Once()

			createdDefaultPkg := &param.ServicePackageParam{ServicePackageID: 20, ServiceID: 10, ServicePackageName: "Massage"}
			serviceRepo.EXPECT().
				InsertServicePackage(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServicePackageParam) bool {
					return p.ServiceID == 10 && p.ServicePackageName == "Massage"
				})).
				Return(createdDefaultPkg, nil).
				Once()
			serviceRepo.EXPECT().
				InsertPackageItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.PackageItemParam) bool {
					return p.ServicePackageID == 20 && p.PackageItemName == "Massage"
				})).
				Return(nil).
				Once()

			createdPkg := &param.ServicePackageParam{ServicePackageID: 21, ServiceID: 10, ServicePackageName: "Deep Tissue"}
			serviceRepo.EXPECT().
				InsertServicePackage(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServicePackageParam) bool {
					return p.ServiceID == 10 && p.ServicePackageName == "Deep Tissue"
				})).
				Return(createdPkg, nil).
				Once()
			serviceRepo.EXPECT().
				InsertPackageItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.PackageItemParam) bool {
					return p.ServicePackageID == 21 && p.PackageItemName == "Massage"
				})).
				Return(nil).
				Once()
			serviceRepo.EXPECT().
				InsertPackageItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.PackageItemParam) bool {
					return p.ServicePackageID == 21 && p.PackageItemName == "Oil"
				})).
				Return(nil).
				Once()

			dbMock.ExpectCommit()

			result, err := serviceSvc.CreateService(ctx, serviceParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceID).To(Equal(int64(10)))
			Expect(result.ServicePackages).To(HaveLen(2))
			Expect(result.ServicePackages[0].ServicePackageID).To(Equal(int64(20)))
			Expect(result.ServicePackages[1].ServicePackageID).To(Equal(int64(21)))
		})

		It("returns validation error", func() {
			invalidParam := param.ServiceParam{ServiceName: ""}

			result, err := serviceSvc.CreateService(ctx, invalidParam)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("rolls back when InsertService fails", func() {
			dbMock.ExpectBegin()
			serviceRepo.EXPECT().
				InsertService(ctx, mock.AnythingOfType("*sql.Tx"), expectedServiceParam).
				Return(nil, fmt.Errorf("insert error")).
				Once()
			dbMock.ExpectRollback()

			result, err := serviceSvc.CreateService(ctx, serviceParam)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("insert error"))
		})

		It("rolls back when InsertServicePackage fails", func() {
			dbMock.ExpectBegin()
			serviceRepo.EXPECT().
				InsertService(ctx, mock.AnythingOfType("*sql.Tx"), expectedServiceParam).
				Return(createdSvc, nil).
				Once()
			serviceRepo.EXPECT().
				InsertServicePackage(ctx, mock.AnythingOfType("*sql.Tx"), mock.Anything).
				Return(nil, fmt.Errorf("pkg error")).
				Once()
			dbMock.ExpectRollback()

			result, err := serviceSvc.CreateService(ctx, serviceParam)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("pkg error"))
		})
	})

	Describe("UpdateService", func() {

		var (
			serviceParam         param.ServiceParam
			expectedServiceParam param.ServiceParam
			updatedSvc           *param.ServiceParam
		)

		BeforeEach(func() {
			serviceParam = param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServicePackages: []param.ServicePackageParam{
					{
						ServicePackageName: "Swedish",
						PackageItems: []param.PackageItemParam{
							{PackageItemName: "Lotion"},
						},
					},
				},
			}
			expectedServiceParam = param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServicePackages: []param.ServicePackageParam{
					{
						ServicePackageName: "Updated Massage",
						PackageItems: []param.PackageItemParam{
							{PackageItemName: "Updated Massage"},
						},
					},
					{
						ServicePackageName: "Swedish",
						PackageItems: []param.PackageItemParam{
							{PackageItemName: "Updated Massage"},
							{PackageItemName: "Lotion"},
						},
					},
				},
			}
			updatedSvc = &param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
			}
		})

		It("successfully updates a service and its packages", func() {
			dbMock.ExpectBegin()

			serviceRepo.EXPECT().
				UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), expectedServiceParam).
				Return(updatedSvc, nil).
				Once()

			serviceRepo.EXPECT().
				DeleteServicePackagesByServiceID(ctx, mock.AnythingOfType("*sql.Tx"), int64(10)).
				Return(nil).
				Once()

			createdDefaultPkg := &param.ServicePackageParam{ServicePackageID: 30, ServiceID: 10, ServicePackageName: "Updated Massage"}
			serviceRepo.EXPECT().
				InsertServicePackage(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServicePackageParam) bool {
					return p.ServiceID == 10 && p.ServicePackageName == "Updated Massage"
				})).
				Return(createdDefaultPkg, nil).
				Once()
			serviceRepo.EXPECT().
				InsertPackageItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.PackageItemParam) bool {
					return p.ServicePackageID == 30 && p.PackageItemName == "Updated Massage"
				})).
				Return(nil).
				Once()

			createdPkg := &param.ServicePackageParam{ServicePackageID: 31, ServiceID: 10, ServicePackageName: "Swedish"}
			serviceRepo.EXPECT().
				InsertServicePackage(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServicePackageParam) bool {
					return p.ServiceID == 10 && p.ServicePackageName == "Swedish"
				})).
				Return(createdPkg, nil).
				Once()
			serviceRepo.EXPECT().
				InsertPackageItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.PackageItemParam) bool {
					return p.ServicePackageID == 31 && p.PackageItemName == "Updated Massage"
				})).
				Return(nil).
				Once()
			serviceRepo.EXPECT().
				InsertPackageItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.PackageItemParam) bool {
					return p.ServicePackageID == 31 && p.PackageItemName == "Lotion"
				})).
				Return(nil).
				Once()

			dbMock.ExpectCommit()

			result, err := serviceSvc.UpdateService(ctx, serviceParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceName).To(Equal("Updated Massage"))
			Expect(result.ServicePackages).To(HaveLen(2))
		})

		It("rolls back when DeleteServicePackagesByServiceID fails", func() {
			dbMock.ExpectBegin()
			serviceRepo.EXPECT().
				UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), expectedServiceParam).
				Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().
				DeleteServicePackagesByServiceID(ctx, mock.AnythingOfType("*sql.Tx"), int64(10)).
				Return(fmt.Errorf("delete pkg error")).Once()
			dbMock.ExpectRollback()

			result, err := serviceSvc.UpdateService(ctx, serviceParam)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("delete pkg error"))
		})
	})

	Describe("DeleteService", func() {
		It("successfully deletes a service when no bookings exist", func() {
			serviceRepo.EXPECT().HasBookingForService(ctx, int64(10)).Return(false, nil).Once()
			dbMock.ExpectBegin()
			serviceRepo.EXPECT().SoftDeleteService(ctx, mock.AnythingOfType("*sql.Tx"), int64(10), int64(1)).Return(nil).Once()
			dbMock.ExpectCommit()

			err := serviceSvc.DeleteService(ctx, 10, 1)
			Expect(err).NotTo(HaveOccurred())
		})

		It("fails to delete when bookings exist", func() {
			serviceRepo.EXPECT().HasBookingForService(ctx, int64(10)).Return(true, nil).Once()

			err := serviceSvc.DeleteService(ctx, 10, 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("booking exists"))
		})

		It("returns error when HasBookingForService fails", func() {
			serviceRepo.EXPECT().HasBookingForService(ctx, int64(10)).Return(false, fmt.Errorf("db error")).Once()

			err := serviceSvc.DeleteService(ctx, 10, 1)
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetServicesByBusinessID", func() {
		It("successfully fetches services with packages and items", func() {
			services := []param.ServiceParam{
				{ServiceID: 10, ServiceName: "Massage"},
			}
			packages := []param.ServicePackageParam{
				{ServicePackageID: 20, ServicePackageName: "Deep Tissue"},
			}
			items := []param.PackageItemParam{
				{PackageItemName: "Oil"},
			}

			serviceRepo.EXPECT().GetServicesByBusinessID(ctx, int64(1)).Return(services, nil).Once()
			serviceRepo.EXPECT().GetServicePackagesByServiceID(ctx, int64(10)).Return(packages, nil).Once()
			serviceRepo.EXPECT().GetPackageItemsByPackageID(ctx, int64(20)).Return(items, nil).Once()

			result, err := serviceSvc.GetServicesByBusinessID(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ServicePackages).To(HaveLen(1))
			Expect(result[0].ServicePackages[0].PackageItems).To(HaveLen(1))
		})

		It("returns error when GetServicesByBusinessID fails", func() {
			serviceRepo.EXPECT().GetServicesByBusinessID(ctx, int64(1)).Return(nil, fmt.Errorf("db error")).Once()

			result, err := serviceSvc.GetServicesByBusinessID(ctx, 1)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})

		It("returns error when GetServicePackagesByServiceID fails", func() {
			services := []param.ServiceParam{{ServiceID: 10, ServiceName: "Massage"}}
			serviceRepo.EXPECT().GetServicesByBusinessID(ctx, int64(1)).Return(services, nil).Once()
			serviceRepo.EXPECT().GetServicePackagesByServiceID(ctx, int64(10)).Return(nil, fmt.Errorf("pkg error")).Once()

			result, err := serviceSvc.GetServicesByBusinessID(ctx, 1)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("pkg error"))
		})
	})
})
