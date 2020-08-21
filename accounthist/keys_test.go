package accounthist

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_writeActionKey_lexicalOrdering(t *testing.T) {
	account := "omgomgomgomg"

	key1Bytes := make([]byte, actionKeyLen)
	encodeActionKey(key1Bytes, account, uint64(1))
	key1 := hex.EncodeToString(key1Bytes)

	key2Bytes := make([]byte, actionKeyLen)
	encodeActionKey(key2Bytes, account, uint64(2))
	key2 := hex.EncodeToString(key2Bytes)

	// newest key should be first in the ordering.
	assert.Greater(t, key1, key2)
}
