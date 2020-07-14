package kv

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dfuse-io/kvdb/store"

	"go.uber.org/zap"
)

func parseAndCleanDSN(dsn string) (cleanDsn string, opt *dsnOptions, err error) {
	zlog.Debug("parsing DSN", zap.String("dsn", dsn))

	dsnOptions := &dsnOptions{
		reads:  []string{},
		writes: []string{},
	}
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
		dsnOptions.reads = append(dsnOptions.reads, strings.Split(readValues, ",")...)
	}

	if len(dsnOptions.reads) == 0 {
		dsnOptions.reads = append(dsnOptions.reads, "all")
	}

	for _, writeValues := range query["write"] {
		dsnOptions.writes = append(dsnOptions.writes, strings.Split(writeValues, ",")...)
	}

	if len(dsnOptions.writes) == 0 {
		dsnOptions.writes = append(dsnOptions.writes, "all")
	}

	cleanDsn, err = store.RemoveDSNOptions(dsn, "read", "write", "blk_marker")
	if err != nil {
		err = fmt.Errorf("Unable to clean dsn: %w", err)
	}

	return cleanDsn, dsnOptions, nil
}
