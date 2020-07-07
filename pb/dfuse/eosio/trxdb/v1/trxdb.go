package pbtrxdb

import (
	"sort"
	"strings"
)

// HumanKey returns the human friendly key that can be used to represent this category
func (i IndexableCategory) HumanKey() string {
	return strings.ToLower(
		strings.Replace(IndexableCategory_name[int32(i)], "INDEXABLE_CATEGORY_", "", 1),
	)
}

// IndexableCategoryFromHumanKey transforms the human key representing the category
// and turns it into a proper `IndexableCategory` category object, returning an extra
// bool to determine if the conversion was successful.
func IndexableCategoryFromHumanKey(key string) (IndexableCategory, bool) {
	value, found := IndexableCategory_value["INDEXABLE_CATEGORY_"+strings.ToUpper(strings.TrimSpace(key))]
	return IndexableCategory(value), found
}

// IndexableCategoryHumanKeys returns the valid set of human keys that can be used as
// valid indexable categories.
func IndexableCategoryHumanKeys() (out []string) {
	for category := range IndexableCategory_name {
		out = append(out, IndexableCategory(category).HumanKey())
	}
	sort.Sort(sort.StringSlice(out))

	return
}
