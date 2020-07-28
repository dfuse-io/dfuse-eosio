package rest

import (
	"net/http"
	"net/url"

	"github.com/dfuse-io/validator"
)

func validateBlocksRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, validator.Rules{
		"skip":  []string{"numeric"},
		"limit": []string{"required", "numeric_between:1,100"},
	})
}

func validateListRequest(r *http.Request) url.Values {
	return validator.ValidateQueryParams(r, validator.Rules{
		"limit":  []string{"required", "numeric_between:1,100"},
		"cursor": []string{"eosws.cursor"},
	})
}
