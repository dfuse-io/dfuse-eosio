package kv

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dfuse-io/kvdb/store"

	"go.uber.org/zap"
)

func parseAndCleanDSN(dsn string) (cleanDsn string, read []string, write []string, err error) {
	zlog.Debug("parsing DSN", zap.String("dsn", dsn))
	dsnURL, err := url.Parse(dsn)
	if err != nil {
		err = fmt.Errorf("invalid dsn: %w", err)
		return
	}

	query, err := url.ParseQuery(dsnURL.RawQuery)
	if err != nil {
		err = fmt.Errorf("invalid query: %w", err)
		return
	}

	for _, readValues := range query["read"] {
		read = append(read, strings.Split(readValues, ",")...)
	}

	if len(read) == 0 {
		read = append(read, "all")
	}

	for _, writeValues := range query["write"] {
		write = append(write, strings.Split(writeValues, ",")...)
	}

	if len(write) == 0 {
		write = append(write, "all")
	}

	cleanDsn, err = store.RemoveDSNOptions(dsn, "read", "write")
	if err != nil {
		err = fmt.Errorf("Unable to clean dsn: %w", err)
	}

	return
}
