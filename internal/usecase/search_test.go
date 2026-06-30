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

var _ = Describe("SearchBooks", func() {
	var (
		ctrl *gomock.Controller
		repo *mocks.MockCatalogRepository
		uc   *usecase.SearchBooks
		ctx  context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockCatalogRepository(ctrl)
		uc = usecase.NewSearchBooks(repo)
		ctx = context.Background()
	})

	AfterEach(func() { ctrl.Finish() })

	It("rejects an empty query without hitting the repository", func() {
		_, err := uc.Execute(ctx, domain.SearchQuery{Query: "   "})
		Expect(err).To(MatchError(usecase.ErrEmptyQuery))
	})

	It("applies defaults for store, lang, currency and rows", func() {
		repo.EXPECT().
			Search(ctx, domain.SearchQuery{
				Query:    "Harry Potter",
				Start:    0,
				Rows:     16,
				Store:    "ES",
				Lang:     "es",
				Currency: "EUR",
			}).
			Return(domain.SearchResult{Total: 1758}, nil)

		res, err := uc.Execute(ctx, domain.SearchQuery{Query: "  Harry Potter  "})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Total).To(Equal(1758))
	})

	It("clamps rows above the maximum", func() {
		repo.EXPECT().
			Search(ctx, gomock.AssignableToTypeOf(domain.SearchQuery{})).
			DoAndReturn(func(_ context.Context, q domain.SearchQuery) (domain.SearchResult, error) {
				Expect(q.Rows).To(Equal(100))
				return domain.SearchResult{}, nil
			})

		_, err := uc.Execute(ctx, domain.SearchQuery{Query: "x", Rows: 5000})
		Expect(err).ToNot(HaveOccurred())
	})

	It("propagates repository errors", func() {
		boom := errors.New("boom")
		repo.EXPECT().Search(gomock.Any(), gomock.Any()).Return(domain.SearchResult{}, boom)

		_, err := uc.Execute(ctx, domain.SearchQuery{Query: "x"})
		Expect(err).To(MatchError(boom))
	})
})
