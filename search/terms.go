package search

import (
	"strings"

	"github.com/dfuse-io/search"
)

var cachedIndexedFieldsMap map[string]*search.IndexedField

// parsedIndexedFields initialize the list of indexed fields of the service
func parseIndexedFields(indexedTerms string) {
	var fields []*search.IndexedField

	terms := strings.Split(indexedTerms, ",")
	for _, term := range terms {
		term = strings.TrimSpace(term)
		fields = append(fields, &search.IndexedField{term, search.FreeFormType})
	}

	// Let's compute the fields map from the actual fields slice
	cachedIndexedFieldsMap = map[string]*search.IndexedField{}
	for _, field := range fields {
		cachedIndexedFieldsMap[field.Name] = field
	}
}
