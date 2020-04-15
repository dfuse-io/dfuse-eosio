package search

import (
	"sort"
	"strings"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/search"
	searchArchive "github.com/dfuse-io/search/archive"
	"github.com/dfuse-io/search/querylang"
	"google.golang.org/grpc/codes"
)

func init() {
	// initializing bleve query factory
	search.GetBleveQueryFactory = func(rawQuery string) *search.BleveQuery {
		return &search.BleveQuery{
			Raw:              rawQuery,
			FieldTransformer: querylang.NoOpFieldTransformer,
			Validator:        &EOSBleveQueryValidator{},
		}
	}

	// initializing indexed fields cache
	InitEOSIndexedFields()
	search.GetIndexedFieldsMap = GetEOSIndexedFieldsMap

	// FIXME: Matt, this needs to become platform agnostic, what's the purpose of this exactly? Warm up?
	livenessQuery, _ := search.NewParsedQuery("receiver:999")
	searchArchive.LivenessQuery = livenessQuery

}

type EOSBleveQueryValidator struct{}

func (v *EOSBleveQueryValidator) Validate(q *search.BleveQuery) error {
	indexedFieldsMap := GetEOSIndexedFieldsMap()

	var unknownFields []string
	for _, fieldName := range q.FieldNames {
		if strings.HasPrefix(fieldName, "data.") {
			fieldName = strings.Join(strings.Split(fieldName, ".")[:2], ".")
		}

		if indexedFieldsMap[fieldName] != nil || strings.HasPrefix(fieldName, "event.") || strings.HasPrefix(fieldName, "parent.") /* we could list the optional fields for `parent.*` */ {
			continue
		}
		unknownFields = append(unknownFields, fieldName)
	}

	if len(unknownFields) <= 0 {
		return nil
	}

	sort.Strings(unknownFields)

	invalidArgString := "The following fields you are trying to search are not currently indexed: '%s'. Contact our support team for more."
	return derr.Statusf(codes.InvalidArgument, invalidArgString, strings.Join(unknownFields, "', '"))
}
