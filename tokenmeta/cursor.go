package tokenmeta

import (
	"fmt"
	"strconv"
	"strings"
)

func parseCursor(cursor string) (blockNum uint64, headBlockID string, trxPrefix string, err error) {
	chunks := strings.Split(cursor, ":")

	if len(chunks) != 4 || chunks[0] != "1" {
		err = fmt.Errorf("invalid cursor")
		return
	}

	blockNum, err = strconv.ParseUint(chunks[1], 10, 64)
	if err != nil {
		return
	}
	headBlockID = chunks[2]
	trxPrefix = chunks[3]

	return
}
