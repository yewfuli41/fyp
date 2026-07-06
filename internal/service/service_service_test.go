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
			// ensureDefaultServiceOptionItems in serviceService.go.
			serviceParam = param.ServiceParam{
				BusinessID:  1,
				ServiceName: "Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{
						ServiceOptionName: "Deep Tissue",
						ServiceOptionItems: []param.ServiceOptionItemParam{
							{ServiceOptionItemName: "Oil"},
						},
					},
				},
			}
			expectedServiceParam = param.ServiceParam{
				BusinessID:  1,
				ServiceName: "Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{
						ServiceOptionName: "Massage",
						ServiceOptionItems: []param.ServiceOptionItemParam{
							{ServiceOptionItemName: "Massage"},
						},
					},
					{
						ServiceOptionName: "Deep Tissue",
						ServiceOptionItems: []param.ServiceOptionItemParam{
							{ServiceOptionItemName: "Massage"},
							{ServiceOptionItemName: "Oil"},
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

			createdDefaultPkg := &param.ServiceOptionParam{ServiceOptionID: 20, ServiceID: 10, ServiceOptionName: "Massage"}
			serviceRepo.EXPECT().
				InsertServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionParam) bool {
					return p.ServiceID == 10 && p.ServiceOptionName == "Massage"
				})).
				Return(createdDefaultPkg, nil).
				Once()
			serviceRepo.EXPECT().
				InsertServiceOptionItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionItemParam) bool {
					return p.ServiceOptionID == 20 && p.ServiceOptionItemName == "Massage"
				})).
				Return(nil).
				Once()

			createdPkg := &param.ServiceOptionParam{ServiceOptionID: 21, ServiceID: 10, ServiceOptionName: "Deep Tissue"}
			serviceRepo.EXPECT().
				InsertServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionParam) bool {
					return p.ServiceID == 10 && p.ServiceOptionName == "Deep Tissue"
				})).
				Return(createdPkg, nil).
				Once()
			serviceRepo.EXPECT().
				InsertServiceOptionItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionItemParam) bool {
					return p.ServiceOptionID == 21 && p.ServiceOptionItemName == "Massage"
				})).
				Return(nil).
				Once()
			serviceRepo.EXPECT().
				InsertServiceOptionItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionItemParam) bool {
					return p.ServiceOptionID == 21 && p.ServiceOptionItemName == "Oil"
				})).
				Return(nil).
				Once()

			dbMock.ExpectCommit()

			result, err := serviceSvc.CreateService(ctx, serviceParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceID).To(Equal(int64(10)))
			Expect(result.ServiceOptions).To(HaveLen(2))
			Expect(result.ServiceOptions[0].ServiceOptionID).To(Equal(int64(20)))
			Expect(result.ServiceOptions[1].ServiceOptionID).To(Equal(int64(21)))
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

		It("rolls back when InsertServiceOption fails", func() {
			dbMock.ExpectBegin()
			serviceRepo.EXPECT().
				InsertService(ctx, mock.AnythingOfType("*sql.Tx"), expectedServiceParam).
				Return(createdSvc, nil).
				Once()
			serviceRepo.EXPECT().
				InsertServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), mock.Anything).
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
				ServiceOptions: []param.ServiceOptionParam{
					{
						ServiceOptionName: "Swedish",
						ServiceOptionItems: []param.ServiceOptionItemParam{
							{ServiceOptionItemName: "Lotion"},
						},
					},
				},
			}
			expectedServiceParam = param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{
						ServiceOptionName: "Updated Massage",
						ServiceOptionItems: []param.ServiceOptionItemParam{
							{ServiceOptionItemName: "Updated Massage"},
						},
					},
					{
						ServiceOptionName: "Swedish",
						ServiceOptionItems: []param.ServiceOptionItemParam{
							{ServiceOptionItemName: "Updated Massage"},
							{ServiceOptionItemName: "Lotion"},
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
				DeleteServiceOptionsByServiceID(ctx, mock.AnythingOfType("*sql.Tx"), int64(10)).
				Return(nil).
				Once()

			createdDefaultPkg := &param.ServiceOptionParam{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Updated Massage"}
			serviceRepo.EXPECT().
				InsertServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionParam) bool {
					return p.ServiceID == 10 && p.ServiceOptionName == "Updated Massage"
				})).
				Return(createdDefaultPkg, nil).
				Once()
			serviceRepo.EXPECT().
				InsertServiceOptionItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionItemParam) bool {
					return p.ServiceOptionID == 30 && p.ServiceOptionItemName == "Updated Massage"
				})).
				Return(nil).
				Once()

			createdPkg := &param.ServiceOptionParam{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"}
			serviceRepo.EXPECT().
				InsertServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionParam) bool {
					return p.ServiceID == 10 && p.ServiceOptionName == "Swedish"
				})).
				Return(createdPkg, nil).
				Once()
			serviceRepo.EXPECT().
				InsertServiceOptionItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionItemParam) bool {
					return p.ServiceOptionID == 31 && p.ServiceOptionItemName == "Updated Massage"
				})).
				Return(nil).
				Once()
			serviceRepo.EXPECT().
				InsertServiceOptionItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionItemParam) bool {
					return p.ServiceOptionID == 31 && p.ServiceOptionItemName == "Lotion"
				})).
				Return(nil).
				Once()

			dbMock.ExpectCommit()

			result, err := serviceSvc.UpdateService(ctx, serviceParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceName).To(Equal("Updated Massage"))
			Expect(result.ServiceOptions).To(HaveLen(2))
		})

		It("rolls back when DeleteServiceOptionsByServiceID fails", func() {
			dbMock.ExpectBegin()
			serviceRepo.EXPECT().
				UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), expectedServiceParam).
				Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().
				DeleteServiceOptionsByServiceID(ctx, mock.AnythingOfType("*sql.Tx"), int64(10)).
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
			packages := []param.ServiceOptionParam{
				{ServiceOptionID: 20, ServiceOptionName: "Deep Tissue"},
			}
			items := []param.ServiceOptionItemParam{
				{ServiceOptionItemName: "Oil"},
			}

			serviceRepo.EXPECT().GetServicesByBusinessID(ctx, int64(1)).Return(services, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return(packages, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(20)).Return(items, nil).Once()

			result, err := serviceSvc.GetServicesByBusinessID(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ServiceOptions).To(HaveLen(1))
			Expect(result[0].ServiceOptions[0].ServiceOptionItems).To(HaveLen(1))
		})

		It("returns error when GetServicesByBusinessID fails", func() {
			serviceRepo.EXPECT().GetServicesByBusinessID(ctx, int64(1)).Return(nil, fmt.Errorf("db error")).Once()

			result, err := serviceSvc.GetServicesByBusinessID(ctx, 1)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})

		It("returns error when GetServiceOptionsByServiceID fails", func() {
			services := []param.ServiceParam{{ServiceID: 10, ServiceName: "Massage"}}
			serviceRepo.EXPECT().GetServicesByBusinessID(ctx, int64(1)).Return(services, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return(nil, fmt.Errorf("pkg error")).Once()

			result, err := serviceSvc.GetServicesByBusinessID(ctx, 1)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("pkg error"))
		})
	})
})
