package kv

import (
	kvdbstore "github.com/streamingfast/kvdb/store"
	"go.uber.org/zap"
)

func (db *DB) SetLogger(logger *zap.Logger) error {
	db.logger = logger
	db.logger.Debug("db is now using custom logger")
	return nil
}

func (db *DB) SetPurgeableStore(ttl, purgeInterval uint64) error {
	zlog.Info("applying purgeable option")

	if db.writeStore != nil {
		db.writeStore = kvdbstore.NewPurgeableStore([]byte{TblTTL}, db.writeStore, ttl)
	}

	db.purgeInterval = purgeInterval
	return nil
}
