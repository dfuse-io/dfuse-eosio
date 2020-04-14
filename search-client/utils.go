package searchclient

import (
	"encoding/hex"
	"strings"
)

func decodeHex(input string) ([]byte, error) {
	out, err := hex.DecodeString(strings.ToLower(input))
	if err != nil {
		return nil, err
	}

	return out, nil
}
