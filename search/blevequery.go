package search

import (
	"sort"
	"strings"

	"github.com/dfuse-io/derr"
	search "github.com/dfuse-io/search"
	"google.golang.org/grpc/codes"
)

type BleveQueryValidator struct {
	indexedTerms *IndexedTerms
}

func (v *BleveQueryValidator) Validate(q *search.BleveQuery) error {
	var unknownFields []string
	for _, fieldName := range q.FieldNames {
		if !v.indexedTerms.IsIndexed(fieldName) {
			unknownFields = append(unknownFields, fieldName)
		}
	}

	if len(unknownFields) <= 0 {
		return nil
	}

	sort.Strings(unknownFields)

	invalidArgString := "The following fields you are trying to search are not currently indexed: '%s'. Contact our support team for more."
	return derr.Statusf(codes.InvalidArgument, invalidArgString, strings.Join(unknownFields, "', '"))
}
