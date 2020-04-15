package search

import (
	"github.com/dfuse-io/search"
	searchArchive "github.com/dfuse-io/search/archive"
	"github.com/dfuse-io/search/querylang"
)

func init() {
	search.GetMatchCollector = Collect
	search.GetSearchMatchFactory = func() search.SearchMatch {
		return &EOSSearchMatch{}
	}
	search.GetBleveQueryFactory = func(rawQuery string) *search.BleveQuery {
		return &search.BleveQuery{
			Raw:              rawQuery,
			FieldTransformer: querylang.NoOpFieldTransformer,
			Validator:        &EOSBleveQueryValidator{},
		}
	}
	InitEOSIndexedFields()
	search.GetIndexedFieldsMap = GetEOSIndexedFieldsMap
	livenessQuery, _ := search.NewParsedQuery("receiver:999")
	searchArchive.LivenessQuery = livenessQuery

}
