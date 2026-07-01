package usecase_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"app/internal/domain"
	"app/internal/mocks"
	"app/internal/usecase"
)

var _ = Describe("ListStores", func() {
	var (
		ctrl *gomock.Controller
		repo *mocks.MockStockRepository
		uc   *usecase.ListStores
		ctx  context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockStockRepository(ctrl)
		uc = usecase.NewListStores(repo)
		ctx = context.Background()
	})

	AfterEach(func() { ctrl.Finish() })

	directory := []domain.Store{
		{StoreID: 20, Province: "Zaragoza", City: "Zaragoza", Address: "San Miguel, 4"},
		{StoreID: 38, Province: "Zaragoza", City: "Zaragoza", Address: "C. C. Gran Casa, av. María Zambrano, 35"},
		{StoreID: 1, Province: "Madrid", City: "Madrid", Address: "Gran Vía, 29"},
	}

	It("defaults the country cache and returns all stores when query is empty", func() {
		repo.EXPECT().Stores(ctx, 63).Return(directory, nil)

		stores, err := uc.Execute(ctx, "  ", 0)
		Expect(err).ToNot(HaveOccurred())
		Expect(stores).To(HaveLen(3))
	})

	It("matches space-insensitively so 'grancasa' resolves the Gran Casa store", func() {
		repo.EXPECT().Stores(ctx, 63).Return(directory, nil)

		stores, err := uc.Execute(ctx, "grancasa", 63)
		Expect(err).ToNot(HaveOccurred())
		Expect(stores).To(HaveLen(1))
		Expect(stores[0].StoreID).To(Equal(38))
	})

	It("matches by city too", func() {
		repo.EXPECT().Stores(ctx, 63).Return(directory, nil)

		stores, err := uc.Execute(ctx, "madrid", 63)
		Expect(err).ToNot(HaveOccurred())
		Expect(stores).To(HaveLen(1))
		Expect(stores[0].StoreID).To(Equal(1))
	})
})
