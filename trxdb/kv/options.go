package kv

import (
	kvdbstore "github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

func (db *DB) SetLogger(logger *zap.Logger) error {
	db.logger = logger
	db.logger.Debug("db is now using custom logger")
	return nil
}

func (db *DB) SetPurgeableStore(ttl, purgeInterval uint64) error {
	if db.blkWriteStore != nil {
		db.blkWriteStore = kvdbstore.NewPurgeableStore([]byte{TblTTL}, db.blkWriteStore, ttl)
	}
	if db.trxWriteStore != nil {
		db.trxWriteStore = kvdbstore.NewPurgeableStore([]byte{TblTTL}, db.trxWriteStore, ttl)
	}
	if db.irrBlockStore != nil {
		db.irrBlockStore = kvdbstore.NewPurgeableStore([]byte{TblTTL}, db.irrBlockStore, ttl)
	}

	db.purgeInterval = purgeInterval
	return nil
}
