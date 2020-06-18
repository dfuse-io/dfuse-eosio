package search

import (
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/search"
	searchArchive "github.com/dfuse-io/search/archive"
	"github.com/dfuse-io/search/querylang"
)

func RegisterDefaultHandlers() {
	terms, err := filtering.NewIndexedTerms("*")
	if err != nil {
		panic(fmt.Sprintf("failed setting up terms in init: %s", err))
	}
	RegisterHandlers(terms)
}

func RegisterHandlers(terms *filtering.IndexedTerms) {
	// GET RID OF THOSE GLOBALS!

	search.GetMatchCollector = collector
	search.GetSearchMatchFactory = func() search.SearchMatch { return &EOSSearchMatch{} }
	search.GetBleveQueryFactory = func(rawQuery string) *search.BleveQuery {
		return &search.BleveQuery{
			Raw:              rawQuery,
			FieldTransformer: querylang.NoOpFieldTransformer,
			// TODO: BleveQueryValidator, where does that belong?
			Validator: &BleveQueryValidator{
				indexedTerms: terms,
			},
		}
	}
	livenessQuery, _ := search.NewParsedQuery("receiver:999")
	searchArchive.LivenessQuery = livenessQuery

}
