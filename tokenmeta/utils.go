package tokenmeta

// check if the string is in the filter list. If the filter list is empty return true
func stringInFilter(str string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}

	for _, v := range filter {
		if v == str {
			return true
		}
	}
	return false
}
