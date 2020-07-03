package search

import (
	"fmt"

	"github.com/dfuse-io/search"
	searchArchive "github.com/dfuse-io/search/archive"
	"github.com/dfuse-io/search/querylang"
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
			FieldTransformer: querylang.NoOpFieldTransformer,
			Validator:        validator,
		}
	}
	livenessQuery, _ := search.NewParsedQuery("receiver:999")
	searchArchive.LivenessQuery = livenessQuery
}
