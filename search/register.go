package search

import (
	"context"
	"fmt"

	"github.com/streamingfast/search"
	searchArchive "github.com/streamingfast/search/archive"
	"github.com/streamingfast/search/sqe"
)

func RegisterDefaultHandlers() {
	terms, err := NewIndexedTerms("*")
	if err != nil {
		panic(fmt.Errorf("failed setting up terms in init: %w", err))
	}

	RegisterHandlers(terms)
}

func RegisterHandlers(terms *IndexedTerms) {
	validator := &BleveQueryValidator{
		indexedTerms: terms,
	}

	search.GetMatchCollector = collector
	search.GetSearchMatchFactory = func() search.SearchMatch { return &SearchMatch{} }
	search.GetBleveQueryFactory = func(rawQuery string) *search.BleveQuery {
		return &search.BleveQuery{
			Raw:              rawQuery,
			FieldTransformer: sqe.NoOpFieldTransformer,
			Validator:        validator,
		}
	}
	livenessQuery, _ := search.NewParsedQuery(context.Background(), "receiver:999")
	searchArchive.LivenessQuery = livenessQuery
}
