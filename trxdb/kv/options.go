package kv

import (
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"go.uber.org/zap"
)

func (db *DB) AcceptLoggerOption(o trxdb.LoggerOption) error {
	db.logger = o.Logger

	db.logger.Debug("db is now using custom logger")
	return nil
}

func (db *DB) AcceptWriteOnlyOption(o trxdb.WriteOnlyOption) error {
	db.indexableCategories = nil
	if o.Categories != nil {
		db.indexableCategories = o.Categories.ToMap()
		db.logger.Debug("db is now using write only option", zap.Strings("categories", o.Categories.AsHumanKeys()))
	}

	return nil
}
