package usecase_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"app/internal/domain"
	"app/internal/mocks"
	"app/internal/usecase"
)

var _ = Describe("GetStoreStock", func() {
	var (
		ctrl *gomock.Controller
		repo *mocks.MockStockRepository
		uc   *usecase.GetStoreStock
		ctx  context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockStockRepository(ctrl)
		uc = usecase.NewGetStoreStock(repo)
		ctx = context.Background()
	})

	AfterEach(func() { ctrl.Finish() })

	It("rejects an empty product id without hitting the repository", func() {
		_, err := uc.Execute(ctx, usecase.StockQuery{ProductID: "  ", CountryCache: 63})
		Expect(err).To(MatchError(usecase.ErrEmptyProductID))
	})

	It("defaults the country cache to 63 and trims the product id", func() {
		repo.EXPECT().
			StockByStore(ctx, "16801604", 63).
			Return([]domain.Province{{Name: "Alicante"}}, nil)

		provinces, err := uc.Execute(ctx, usecase.StockQuery{ProductID: " 16801604 "})
		Expect(err).ToNot(HaveOccurred())
		Expect(provinces).To(HaveLen(1))
		Expect(provinces[0].Name).To(Equal("Alicante"))
	})

	It("filters to a single store and drops zero-stock bookstores", func() {
		repo.EXPECT().
			StockByStore(ctx, "1", 63).
			Return([]domain.Province{{
				Name: "Zaragoza",
				Bookstores: []domain.Bookstore{
					{StoreID: 20, Stock: 0},
					{StoreID: 38, Stock: 2},
					{StoreID: 99, Stock: 5},
				},
			}}, nil)

		provinces, err := uc.Execute(ctx, usecase.StockQuery{ProductID: "1", StoreID: 38, InStockOnly: true})
		Expect(err).ToNot(HaveOccurred())
		Expect(provinces).To(HaveLen(1))
		Expect(provinces[0].Bookstores).To(HaveLen(1))
		Expect(provinces[0].Bookstores[0].StoreID).To(Equal(38))
	})

	It("propagates repository errors", func() {
		boom := errors.New("boom")
		repo.EXPECT().StockByStore(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, boom)

		_, err := uc.Execute(ctx, usecase.StockQuery{ProductID: "1", CountryCache: 63})
		Expect(err).To(MatchError(boom))
	})
})
