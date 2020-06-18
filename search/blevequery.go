package search

import (
	"sort"
	"strings"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/search"
	"google.golang.org/grpc/codes"
)

type BleveQueryValidator struct {
	indexedTerms *filtering.IndexedTerms
}

func (v *BleveQueryValidator) Validate(q *search.BleveQuery) error {
	indexed := v.indexedTerms

	var unknownFields []string
	for _, fieldName := range q.FieldNames {
		if strings.HasPrefix(fieldName, "data.") {
			// transform `data.some.nested` into `data.some`
			fieldName = strings.Join(strings.Split(fieldName, ".")[:2], ".")
		}

		if !indexed.BaseFields[fieldName] && !strings.HasPrefix(fieldName, "event.") {
			unknownFields = append(unknownFields, fieldName)
		}
	}

	if len(unknownFields) <= 0 {
		return nil
	}

	sort.Strings(unknownFields)

	invalidArgString := "The following fields you are trying to search are not currently indexed: '%s'."
	return derr.Statusf(codes.InvalidArgument, invalidArgString, strings.Join(unknownFields, "', '"))
}
