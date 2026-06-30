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
		_, err := uc.Execute(ctx, "  ", 63)
		Expect(err).To(MatchError(usecase.ErrEmptyProductID))
	})

	It("defaults the country cache to 63 and trims the product id", func() {
		repo.EXPECT().
			StockByStore(ctx, "16801604", 63).
			Return([]domain.Province{{Name: "Alicante"}}, nil)

		provinces, err := uc.Execute(ctx, " 16801604 ", 0)
		Expect(err).ToNot(HaveOccurred())
		Expect(provinces).To(HaveLen(1))
		Expect(provinces[0].Name).To(Equal("Alicante"))
	})

	It("propagates repository errors", func() {
		boom := errors.New("boom")
		repo.EXPECT().StockByStore(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, boom)

		_, err := uc.Execute(ctx, "1", 63)
		Expect(err).To(MatchError(boom))
	})
})
