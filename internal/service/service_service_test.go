package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/errs"
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
			// A service is saved with its options as provided. ensureDefaultOption
			// only supplies a fallback name for the first option when it is empty
			// (and adds a single option when none are given) — see
			// ensureDefaultOption in serviceService.go. No extra default option or
			// per-option default item is injected.
			serviceParam = param.ServiceParam{
				BusinessID:  1,
				ServiceName: "Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{
						ServiceOptionName: "Deep Tissue",
						EffectiveFrom:     "2030-01-01", // now required — no more defaulting to today
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
						ServiceOptionName: "Deep Tissue",
						EffectiveFrom:     "2030-01-01",
						ServiceOptionItems: []param.ServiceOptionItemParam{
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

			createdPkg := &param.ServiceOptionParam{ServiceOptionID: 21, ServiceID: 10, ServiceOptionName: "Deep Tissue"}
			serviceRepo.EXPECT().
				InsertServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.ServiceOptionParam) bool {
					return p.ServiceID == 10 && p.ServiceOptionName == "Deep Tissue"
				})).
				Return(createdPkg, nil).
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
			Expect(result.ServiceOptions).To(HaveLen(1))
			Expect(result.ServiceOptions[0].ServiceOptionID).To(Equal(int64(21)))
		})

		It("returns validation error", func() {
			invalidParam := param.ServiceParam{ServiceName: ""}

			result, err := serviceSvc.CreateService(ctx, invalidParam)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})

		// UT-006 (Service & Option Constraints).
		It("rejects an option with no effective-from date — it's no longer defaulted to today", func() {
			p := param.ServiceParam{
				BusinessID:  1,
				ServiceName: "Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionName: "Deep Tissue"}, // EffectiveFrom left blank
				},
			}

			// The check only runs once inside the transaction, after the service
			// row itself is inserted — so that part of the flow still happens.
			dbMock.ExpectBegin()
			serviceRepo.EXPECT().
				InsertService(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceParam) bool {
					return sp.ServiceName == "Massage"
				})).
				Return(createdSvc, nil).Once()
			dbMock.ExpectRollback()

			result, err := serviceSvc.CreateService(ctx, p)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "Effective from is required",
			}))
		})

		// UT-007 (Service & Option Constraints).
		It("rejects a brand new option with an effective-from date in the past", func() {
			past := "2000-01-01"
			p := param.ServiceParam{
				BusinessID:  1,
				ServiceName: "Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionName: "Deep Tissue", EffectiveFrom: past},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().
				InsertService(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceParam) bool {
					return sp.ServiceName == "Massage"
				})).
				Return(createdSvc, nil).Once()
			dbMock.ExpectRollback()

			result, err := serviceSvc.CreateService(ctx, p)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "Effective from cannot be in the past",
			}))
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
		var updatedSvc *param.ServiceParam

		// A window safely in the future so validateEditWindow (from >= today) passes.
		const editFrom = "2030-01-01"
		const editUntil = "2030-01-31"

		BeforeEach(func() {
			updatedSvc = &param.ServiceParam{ServiceID: 10, ServiceName: "Updated Massage"}
		})

		anyServiceParam := mock.MatchedBy(func(sp param.ServiceParam) bool { return sp.ServiceID == 10 })

		It("leaves an unchanged option untouched", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 31, ServiceOptionName: "Swedish",
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			serviceRepo.EXPECT().SetServiceDefaultOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(10), int64(31)).Return(nil).Once()
			dbMock.ExpectCommit()

			result, err := serviceSvc.UpdateService(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceOptions).To(HaveLen(1))
			Expect(result.ServiceOptions[0].ServiceOptionID).To(Equal(int64(31)))
		})

		It("rejects moving an existing option's effective-from date, even to a future date", func() {
			newFrom := editFrom // "2030-01-01"
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 30, ServiceOptionName: "Default", // default option, unchanged
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}},
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", // unchanged name/items
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}},
						EffectiveFrom:      newFrom},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default"},
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish", EffectiveFrom: "2020-01-01"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[1]", Message: "An option's effective-from date can't be changed after it's created. Delete this option and add a replacement with the dates you want.",
			}))
		})

		It("rejects setting an effective-until date on an existing option", func() {
			until := editUntil // "2030-01-31"
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 30, ServiceOptionName: "Default",
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}},
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", // unchanged name/items
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}},
						EffectiveUntil:     &until},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default"},
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish", EffectiveFrom: "2020-01-01"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[1]", Message: "An option's effective-until date can't be changed after it's created. Delete this option and add a replacement with the dates you want.",
			}))
		})

		It("rejects clearing an existing option's saved effective-until date", func() {
			until := editUntil
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 30, ServiceOptionName: "Default",
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}},
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", ClearEffectiveUntil: true,
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default"},
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish", EffectiveFrom: "2020-01-01", EffectiveUntil: &until},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[1]", Message: "An option's effective-until date can't be removed after it's created. Delete this option and add a replacement with the dates you want.",
			}))
		})

		// UT-008 (Service & Option Constraints).
		It("rejects an effective-until in the past for an existing option — the window is fixed either way", func() {
			past := "2000-01-01"
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 30, ServiceOptionName: "Default",
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}},
					{ServiceOptionID: 31, ServiceOptionName: "Swedish",
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}},
						EffectiveUntil:     &past},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default"},
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[1]", Message: "An option's effective-until date can't be changed after it's created. Delete this option and add a replacement with the dates you want.",
			}))
		})

		// UT-009 (Service & Option Constraints).
		It("rejects renaming an existing option directly", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 31, ServiceOptionName: "Swedish Deluxe", // changed name
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "To change this option's name or items, delete it and add a new option instead.",
			}))
		})

		// UT-009 (Service & Option Constraints).
		It("rejects changing an existing option's items directly", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", // same name, changed items
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Oil"}}},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "To change this option's name or items, delete it and add a new option instead.",
			}))
		})

		It("allows a brand new option to be scheduled with its own effective window", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", // unchanged
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
					{ServiceOptionName: "Hot Stone", // brand new, no ID
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Stones"}},
						EffectiveFrom:      editFrom},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()

			created := &param.ServiceOptionParam{ServiceOptionID: 41, ServiceID: 10, ServiceOptionName: "Hot Stone"}
			serviceRepo.EXPECT().InsertServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(o param.ServiceOptionParam) bool {
				return o.ServiceOptionName == "Hot Stone" && o.EffectiveFrom == editFrom && o.EffectiveUntil == nil
			})).Return(created, nil).Once()
			serviceRepo.EXPECT().InsertServiceOptionItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(it param.ServiceOptionItemParam) bool {
				return it.ServiceOptionID == 41 && it.ServiceOptionItemName == "Stones"
			})).Return(nil).Once()
			serviceRepo.EXPECT().SetServiceDefaultOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(10), int64(31)).Return(nil).Once()
			dbMock.ExpectCommit()

			result, err := serviceSvc.UpdateService(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceOptions).To(HaveLen(2))
			Expect(result.ServiceOptions[1].ServiceOptionID).To(Equal(int64(41)))
		})

		// UT-006 (Service & Option Constraints).
		It("rejects a brand new option with no effective-from date", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", // unchanged
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
					{ServiceOptionName: "Hot Stone", // brand new, EffectiveFrom left blank
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Stones"}}},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[1]", Message: "Effective from is required",
			}))
		})

		// UT-011 (Service & Option Constraints).
		It("rejects an effective-until date on the default option", func() {
			until := editUntil
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 30, ServiceOptionName: "Default",
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}},
						EffectiveUntil:     &until},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "The default option can't have an effective-until date. Set another option as default first.",
			}))
		})

		// UT-011 (Service & Option Constraints).
		It("rejects moving the default option's effective-from date into the future", func() {
			future := editFrom // "2030-01-01"
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 30, ServiceOptionName: "Default", // unchanged content
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}},
						EffectiveFrom:      future},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default", EffectiveFrom: "2021-06-01"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "An option's effective-from date can't be changed after it's created. Delete this option and add a replacement with the dates you want.",
			}))
		})

		// UT-011 (Service & Option Constraints).
		It("rejects a past effective-from date for the default option — the window is fixed either way", func() {
			past := "2000-01-01"
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 30, ServiceOptionName: "Default",
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}},
						EffectiveFrom:      past},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default", EffectiveFrom: "2021-06-01"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "An option's effective-from date can't be changed after it's created. Delete this option and add a replacement with the dates you want.",
			}))
		})

		It("promotes a different existing option to default when it's submitted at position 0", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", // was NOT default, now at position 0
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
					{ServiceOptionID: 30, ServiceOptionName: "Default", // was default, now demoted
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default"},
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			// The new position-0 option (31) becomes default; 30 is no longer.
			serviceRepo.EXPECT().SetServiceDefaultOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(10), int64(31)).Return(nil).Once()
			dbMock.ExpectCommit()

			result, err := serviceSvc.UpdateService(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceOptions).To(HaveLen(2))
			Expect(result.ServiceOptions[0].ServiceOptionID).To(Equal(int64(31)))
		})

		It("rejects promoting an option with an already-saved end date to default", func() {
			until := editUntil
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", // now at position 0
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
					{ServiceOptionID: 30, ServiceOptionName: "Default",
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default"},
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish", EffectiveUntil: &until},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "This option was created with an effective-until date, which can't be changed. Delete it and add a replacement without an end date to use it as the default.",
			}))
		})

		It("rejects promoting an option to default even when the same save asks to clear its end date", func() {
			until := editUntil
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", ClearEffectiveUntil: true, // now at position 0
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
					{ServiceOptionID: 30, ServiceOptionName: "Default",
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 30, ServiceID: 10, ServiceOptionName: "Default"},
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish", EffectiveFrom: "2020-01-01", EffectiveUntil: &until},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(30)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Base"}}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "This option was created with an effective-until date, which can't be changed. Delete it and add a replacement without an end date to use it as the default.",
			}))
		})

		It("rejects an option ID that doesn't belong to this service", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 999, ServiceOptionName: "Swedish"},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).Return(nil, nil).Once()
			dbMock.ExpectRollback()

			result, err := serviceSvc.UpdateService(ctx, p)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions[0]", Message: "This option no longer exists",
			}))
		})

		It("inserts a brand new option and retires one that's no longer present", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionName: "Swedish", EffectiveFrom: "2030-01-01", // brand new (no id)
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 99, ServiceID: 10, ServiceOptionName: "Deep Tissue"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(99)).Return(nil, nil).Once()

			created := &param.ServiceOptionParam{ServiceOptionID: 32, ServiceID: 10, ServiceOptionName: "Swedish"}
			serviceRepo.EXPECT().InsertServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(o param.ServiceOptionParam) bool {
				return o.ServiceOptionName == "Swedish"
			})).Return(created, nil).Once()
			serviceRepo.EXPECT().InsertServiceOptionItem(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(it param.ServiceOptionItemParam) bool {
				return it.ServiceOptionID == 32 && it.ServiceOptionItemName == "Lotion"
			})).Return(nil).Once()

			// Deep Tissue (99) was dropped → delete it directly, once confirmed
			// nothing booked references it.
			serviceRepo.EXPECT().HasBookingForOption(ctx, int64(99)).Return(false, nil).Once()
			serviceRepo.EXPECT().SoftDeleteServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(99)).Return(nil).Once()
			serviceRepo.EXPECT().SetServiceDefaultOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(10), int64(32)).Return(nil).Once()

			dbMock.ExpectCommit()

			result, err := serviceSvc.UpdateService(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceOptions).To(HaveLen(1))
			Expect(result.ServiceOptions[0].ServiceOptionID).To(Equal(int64(32)))
		})

		// UT-010 (Service & Option Constraints).
		It("rejects deleting an option that has an active booking", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionID: 31, ServiceOptionName: "Swedish", // stays the default, unchanged
						ServiceOptionItems: []param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}},
					// Deep Tissue (99) omitted → would be deleted, but it has a booking.
				},
			}

			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return([]param.ServiceOptionParam{
				{ServiceOptionID: 31, ServiceID: 10, ServiceOptionName: "Swedish"},
				{ServiceOptionID: 99, ServiceID: 10, ServiceOptionName: "Deep Tissue"},
			}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(31)).
				Return([]param.ServiceOptionItemParam{{ServiceOptionItemName: "Lotion"}}, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionID(ctx, int64(99)).Return(nil, nil).Once()
			serviceRepo.EXPECT().HasBookingForOption(ctx, int64(99)).Return(true, nil).Once()
			dbMock.ExpectRollback()

			_, err := serviceSvc.UpdateService(ctx, p)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "serviceOptions", Message: `Can't delete "Deep Tissue" — it has a booking.`,
			}))
		})

		It("rolls back when GetServiceOptionsByServiceID fails", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				ServiceName: "Updated Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionName: "Swedish"},
				},
			}
			dbMock.ExpectBegin()
			serviceRepo.EXPECT().UpdateService(ctx, mock.AnythingOfType("*sql.Tx"), anyServiceParam).Return(updatedSvc, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return(nil, fmt.Errorf("fetch existing options error")).Once()
			dbMock.ExpectRollback()

			result, err := serviceSvc.UpdateService(ctx, p)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("fetch existing options error"))
		})
	})

	Describe("DeleteService", func() {
		It("successfully deletes a service, cascading to its options and any now-orphaned slots", func() {
			serviceRepo.EXPECT().HasBookingForService(ctx, int64(10)).Return(false, nil).Once()
			dbMock.ExpectBegin()
			serviceRepo.EXPECT().
				GetServiceOptionsByServiceID(ctx, int64(10)).
				Return([]param.ServiceOptionParam{
					{ServiceOptionID: 20, ServiceID: 10, ServiceOptionName: "Deep Tissue"},
					{ServiceOptionID: 21, ServiceID: 10, ServiceOptionName: "Swedish"},
				}, nil).
				Once()
			serviceRepo.EXPECT().SoftDeleteServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(20)).Return(nil).Once()
			serviceRepo.EXPECT().SoftDeleteServiceOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(21)).Return(nil).Once()
			serviceRepo.EXPECT().CascadeDeleteServiceSlots(ctx, mock.AnythingOfType("*sql.Tx"), int64(10)).Return(nil).Once()
			serviceRepo.EXPECT().SoftDeleteService(ctx, mock.AnythingOfType("*sql.Tx"), int64(10), int64(1)).Return(nil).Once()
			dbMock.ExpectCommit()

			err := serviceSvc.DeleteService(ctx, 10, 1)
			Expect(err).NotTo(HaveOccurred())
		})

		// UT-012 (Service & Option Constraints).
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

		It("loads removed option items from deleted rows for management display", func() {
			services := []param.ServiceParam{
				{ServiceID: 10, ServiceName: "Massage"},
			}
			packages := []param.ServiceOptionParam{
				{ServiceOptionID: 20, ServiceOptionName: "Deep Tissue", IsRemoved: true},
			}
			items := []param.ServiceOptionItemParam{
				{ServiceOptionItemName: "Oil"},
			}

			serviceRepo.EXPECT().GetServicesByBusinessID(ctx, int64(1)).Return(services, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionsByServiceID(ctx, int64(10)).Return(packages, nil).Once()
			serviceRepo.EXPECT().GetServiceOptionItemsByOptionIDIncludeDeleted(ctx, int64(20)).Return(items, nil).Once()

			result, err := serviceSvc.GetServicesByBusinessID(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ServiceOptions).To(HaveLen(1))
			Expect(result[0].ServiceOptions[0].IsRemoved).To(BeTrue())
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
