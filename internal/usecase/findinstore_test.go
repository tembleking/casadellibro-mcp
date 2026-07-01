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

var _ = Describe("FindBooksInStore", func() {
	var (
		ctrl    *gomock.Controller
		catalog *mocks.MockCatalogRepository
		stock   *mocks.MockStockRepository
		uc      *usecase.FindBooksInStore
		ctx     context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		catalog = mocks.NewMockCatalogRepository(ctrl)
		stock = mocks.NewMockStockRepository(ctrl)
		uc = usecase.NewFindBooksInStore(usecase.NewSearchBooks(catalog), usecase.NewGetStoreStock(stock))
		ctx = context.Background()
	})

	AfterEach(func() { ctrl.Finish() })

	It("requires a positive store id", func() {
		_, err := uc.Execute(ctx, usecase.FindInStoreQuery{Query: "x", StoreID: 0})
		Expect(err).To(MatchError(usecase.ErrNoStoreID))
	})

	It("requires a non-empty query", func() {
		_, err := uc.Execute(ctx, usecase.FindInStoreQuery{Query: "  ", StoreID: 38})
		Expect(err).To(MatchError(usecase.ErrEmptyQuery))
	})

	It("keeps only books in stock at the store, annotated with store stock", func() {
		catalog.EXPECT().
			Search(gomock.Any(), gomock.Any()).
			Return(domain.SearchResult{
				Total: 3,
				Books: []domain.Book{
					{ProductID: "1"}, {ProductID: "2"}, {ProductID: "3"},
				},
			}, nil)

		// 1 -> stock 2, 2 -> zero (filtered out by in-stock-only), 3 -> stock 1
		stock.EXPECT().StockByStore(gomock.Any(), "1", 63).
			Return([]domain.Province{{Name: "Z", Bookstores: []domain.Bookstore{{StoreID: 38, Stock: 2, Availability: "hoy"}}}}, nil)
		stock.EXPECT().StockByStore(gomock.Any(), "2", 63).
			Return([]domain.Province{{Name: "Z", Bookstores: []domain.Bookstore{{StoreID: 38, Stock: 0}}}}, nil)
		stock.EXPECT().StockByStore(gomock.Any(), "3", 63).
			Return([]domain.Province{{Name: "Z", Bookstores: []domain.Bookstore{{StoreID: 38, Stock: 1}}}}, nil)

		res, err := uc.Execute(ctx, usecase.FindInStoreQuery{Query: "x", StoreID: 38})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Scanned).To(Equal(3))
		Expect(res.Books).To(HaveLen(2))
		// catalog order preserved despite concurrent stock checks.
		Expect(res.Books[0].ProductID).To(Equal("1"))
		Expect(res.Books[0].StoreStock).To(Equal(2))
		Expect(res.Books[1].ProductID).To(Equal("3"))
	})

	It("scans one page from start and reports the next page", func() {
		catalog.EXPECT().
			Search(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, q domain.SearchQuery) (domain.SearchResult, error) {
				Expect(q.Start).To(Equal(10))
				Expect(q.Rows).To(Equal(1)) // limit 1 -> fetch only 1 row
				return domain.SearchResult{Total: 500, Books: []domain.Book{{ProductID: "1"}}}, nil
			})
		stock.EXPECT().StockByStore(gomock.Any(), "1", 63).
			Return([]domain.Province{{Name: "Z", Bookstores: []domain.Bookstore{{StoreID: 38, Stock: 0}}}}, nil)

		res, err := uc.Execute(ctx, usecase.FindInStoreQuery{Query: "x", StoreID: 38, Start: 10, Limit: 1})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Start).To(Equal(10))
		Expect(res.Scanned).To(Equal(1))
		Expect(res.NextStart).To(Equal(11))
		Expect(res.Total).To(Equal(500))
		Expect(res.HasMore).To(BeTrue())
		Expect(res.Books).To(BeEmpty())
	})

	It("de-duplicates repeated product ids within a page", func() {
		catalog.EXPECT().
			Search(gomock.Any(), gomock.Any()).
			Return(domain.SearchResult{
				Total: 2,
				Books: []domain.Book{{ProductID: "1"}, {ProductID: "1"}},
			}, nil)
		// dedup -> product 1 checked once
		stock.EXPECT().StockByStore(gomock.Any(), "1", 63).
			Return([]domain.Province{{Name: "Z", Bookstores: []domain.Bookstore{{StoreID: 38, Stock: 1}}}}, nil)

		res, err := uc.Execute(ctx, usecase.FindInStoreQuery{Query: "x", StoreID: 38})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Scanned).To(Equal(2)) // two catalog positions consumed
		Expect(res.Books).To(HaveLen(1)) // but only one unique book
		Expect(res.HasMore).To(BeFalse())
	})
})
