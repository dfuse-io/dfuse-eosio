package trxdb

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type SwitchDB struct {
	logger            *zap.Logger
	readingRoutingMap map[pbtrxdb.IndexableCategory]Driver
	writingDriver     Driver
}

func NewSwitchDB(dsns string, opts ...Option) (*SwitchDB, error) {
	zlog.Info("creating switch db", zap.String("dsns", dsns))
	db := &SwitchDB{
		logger:            zlog,
		readingRoutingMap: map[pbtrxdb.IndexableCategory]Driver{},
	}

	for _, dsn := range strings.Split(dsns, " ") {
		err := db.useDriver(strings.TrimSpace(dsn), opts...)
		if err != nil {
			return nil, err
		}
	}

	if len(db.readingRoutingMap) <= 0 && db.writingDriver == nil {
		return nil, fmt.Errorf("no switching info configured, this is invalid")
	}

	return db, nil
}

func (db *SwitchDB) useDriver(dsn string, opts ...Option) error {
	zlog.Info("using driver", zap.String("dsn", dsn))
	dsnURL, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("invalid dsn: %w", err)
	}

	query, err := url.ParseQuery(dsnURL.RawQuery)
	if err != nil {
		return fmt.Errorf("invalid query: %w", err)
	}

	zlog.Info("switch driver configuration", zap.Strings("read", query["read"]), zap.Strings("write", query["write"]))
	for _, readValues := range query["read"] {
		err := db.useReadDriver(dsnURL, readValues, opts)
		if err != nil {
			return fmt.Errorf("reading driver: %w", err)
		}
	}

	if len(query["write"]) > 0 {
		err := db.useWriteDriver(dsnURL, opts)
		if err != nil {
			return fmt.Errorf("writing driver: %w", err)
		}
	}

	return nil
}

func (db *SwitchDB) useReadDriver(dsnURL *url.URL, readValues string, opts []Option) error {
	zlog.Info("configuring a read driver", zap.Stringer("dsn", dsnURL))

	categories, err := NewIndexableCategories(readValues)
	if err != nil {
		return fmt.Errorf("invalid read values: %w", err)
	}

	// The read driver should not receive either `read` nor `write` option
	dsnURL = store.RemoveDSNOptionsFromURL(dsnURL, "read", "write")

	normalizedReadDSN := dsnURL.String()
	zlog.Info("creating read driver", zap.String("dsn", normalizedReadDSN), zap.Strings("categories", categories.AsHumanKeys()))
	driver, err := newFromDSN(normalizedReadDSN, append(opts, ReadOnly()))
	if err != nil {
		return fmt.Errorf("unable to create read driver: %w", err)
	}

	for _, category := range categories {
		if _, exists := db.readingRoutingMap[category]; exists {
			return fmt.Errorf("category %q is already mapped to a driver, configuration is invalid", category)
		}

		db.readingRoutingMap[category] = driver
	}

	return nil
}

func (db *SwitchDB) useWriteDriver(dsnURL *url.URL, opts []Option) error {
	zlog.Info("configuring a write driver", zap.Stringer("dsn", dsnURL))
	if db.writingDriver != nil {
		return fmt.Errorf("a writing driver has already been configured, only a single writing driver can be specified per instance, configuration is invalid")
	}

	// We keep `write` option since they will be used by the underlying kv store, but we remove `read` option since they are not supported
	normalizedWriteDSN := store.RemoveDSNOptionsFromURL(dsnURL, "read").String()

	zlog.Info("creating write driver", zap.String("dsn", normalizedWriteDSN))
	driver, err := newFromDSN(normalizedWriteDSN, opts)
	if err != nil {
		return fmt.Errorf("unable to create driver: %w", err)
	}

	db.writingDriver = driver
	return nil
}

func (db *SwitchDB) AcceptLoggerOption(o LoggerOption) error {
	db.logger = o.Logger
	return db.dispatchOption(o)
}

func (db *SwitchDB) AcceptWriteOnlyOption(o WriteOnlyOption) error {
	return db.dispatchOption(o)
}

func (db *SwitchDB) dispatchOption(option Option) (err error) {

	for category, readDriver := range db.readingRoutingMap {
		if traceEnabled {
			zlog.Debug("routing option to read driver", zap.Stringer("category", category), zap.Stringer("type", reflect.TypeOf(readDriver)))
		}

		err := option.setOption(readDriver)
		if err != nil {
			return err
		}
	}

	if db.writingDriver != nil {
		if traceEnabled {
			zlog.Debug("routing option to writing driver", zap.Stringer("type", reflect.TypeOf(db.writingDriver)))
		}

		err := option.setOption(db.writingDriver)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *SwitchDB) GetAccount(ctx context.Context, accountName string) (*pbcodec.AccountCreationRef, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_ACCOUNT)
	if err != nil {
		return nil, err
	}

	return driver.GetAccount(ctx, accountName)
}

func (db *SwitchDB) ListAccountNames(ctx context.Context, concurrentReadCount uint32) ([]string, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_ACCOUNT)
	if err != nil {
		return nil, err
	}

	return driver.ListAccountNames(ctx, concurrentReadCount)
}

// Transaction Category

func (db *SwitchDB) GetTransactionTraces(ctx context.Context, idPrefix string) ([]*pbcodec.TransactionEvent, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TRANSACTION)
	if err != nil {
		return nil, err
	}

	return driver.GetTransactionTraces(ctx, idPrefix)
}
func (db *SwitchDB) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) ([][]*pbcodec.TransactionEvent, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TRANSACTION)
	if err != nil {
		return nil, err
	}

	return driver.GetTransactionTracesBatch(ctx, idPrefixes)
}
func (db *SwitchDB) GetTransactionEvents(ctx context.Context, idPrefix string) ([]*pbcodec.TransactionEvent, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TRANSACTION)
	if err != nil {
		return nil, err
	}

	return driver.GetTransactionEvents(ctx, idPrefix)
}
func (db *SwitchDB) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) ([][]*pbcodec.TransactionEvent, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TRANSACTION)
	if err != nil {
		return nil, err
	}

	return driver.GetTransactionEventsBatch(ctx, idPrefixes)
}

// Timeline Category

func (db *SwitchDB) BlockIDAt(ctx context.Context, start time.Time) (id string, err error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TIMELINE)
	if err != nil {
		return "", err
	}

	return driver.BlockIDAt(ctx, start)
}
func (db *SwitchDB) BlockIDAfter(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TIMELINE)
	if err != nil {
		return "", time.Time{}, err
	}

	return driver.BlockIDAfter(ctx, start, inclusive)
}
func (db *SwitchDB) BlockIDBefore(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_TIMELINE)
	if err != nil {
		return "", time.Time{}, err
	}

	return driver.BlockIDBefore(ctx, start, inclusive)
}

// Block Category

func (db *SwitchDB) GetLastWrittenBlockID(ctx context.Context) (blockID string, err error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK)
	if err != nil {
		return "", err
	}

	return driver.GetLastWrittenBlockID(ctx)
}
func (db *SwitchDB) GetBlock(ctx context.Context, id string) (*pbcodec.BlockWithRefs, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK)
	if err != nil {
		return nil, err
	}

	return driver.GetBlock(ctx, id)
}
func (db *SwitchDB) GetBlockByNum(ctx context.Context, num uint32) ([]*pbcodec.BlockWithRefs, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK)
	if err != nil {
		return nil, err
	}

	return driver.GetBlockByNum(ctx, num)
}
func (db *SwitchDB) GetClosestIrreversibleIDAtBlockNum(ctx context.Context, num uint32) (ref bstream.BlockRef, err error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK)
	if err != nil {
		return nil, err
	}

	return driver.GetClosestIrreversibleIDAtBlockNum(ctx, num)
}
func (db *SwitchDB) GetIrreversibleIDAtBlockID(ctx context.Context, ID string) (ref bstream.BlockRef, err error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK)
	if err != nil {
		return nil, err
	}

	return driver.GetIrreversibleIDAtBlockID(ctx, ID)
}
func (db *SwitchDB) ListBlocks(ctx context.Context, highBlockNum uint32, limit int) ([]*pbcodec.BlockWithRefs, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK)
	if err != nil {
		return nil, err
	}

	return driver.ListBlocks(ctx, highBlockNum, limit)
}
func (db *SwitchDB) ListSiblingBlocks(ctx context.Context, blockNum uint32, spread uint32) ([]*pbcodec.BlockWithRefs, error) {
	driver, err := db.routeReadTo(pbtrxdb.IndexableCategory_INDEXABLE_CATEGORY_BLOCK)
	if err != nil {
		return nil, err
	}

	return driver.ListSiblingBlocks(ctx, blockNum, spread)
}

func (db *SwitchDB) routeReadTo(category pbtrxdb.IndexableCategory) (Driver, error) {
	driver, found := db.readingRoutingMap[category]
	if !found {
		return nil, fmt.Errorf("no valid backend able to read category %q, your deployment is not configured correctly", category)
	}

	if traceEnabled {
		zlog.Debug("routing category to read driver", zap.Stringer("category", category), zap.Stringer("type", reflect.TypeOf(driver)))
	}

	return driver, nil
}

// Write

func (db *SwitchDB) SetWriterChainID(chainID []byte) {
	driver, err := db.routeWriteTo()
	if err != nil {
		zlog.Error("invalid configuration, unable to get writing driver", zap.Error(err))
		return
	}

	driver.SetWriterChainID(chainID)
}

func (db *SwitchDB) GetLastWrittenIrreversibleBlockRef(ctx context.Context) (ref bstream.BlockRef, err error) {
	driver, err := db.routeWriteTo()
	if err != nil {
		return nil, err
	}

	return driver.GetLastWrittenIrreversibleBlockRef(ctx)
}

func (db *SwitchDB) PutBlock(ctx context.Context, blk *pbcodec.Block) error {
	driver, err := db.routeWriteTo()
	if err != nil {
		return err
	}

	return driver.PutBlock(ctx, blk)
}

func (db *SwitchDB) UpdateNowIrreversibleBlock(ctx context.Context, blk *pbcodec.Block) error {
	driver, err := db.routeWriteTo()
	if err != nil {
		return err
	}

	return driver.UpdateNowIrreversibleBlock(ctx, blk)
}

func (db *SwitchDB) Flush(ctx context.Context) error {
	driver, err := db.routeWriteTo()
	if err != nil {
		return err
	}

	return driver.Flush(ctx)
}

func (db *SwitchDB) routeWriteTo() (Driver, error) {
	if db.writingDriver == nil {
		return nil, fmt.Errorf("no valid writing backend, your deployment is not configured correctly")
	}

	if traceEnabled {
		zlog.Debug("routing to write driver", zap.Stringer("type", reflect.TypeOf(db.writingDriver)))
	}

	return db.writingDriver, nil
}

// Closer

func (db *SwitchDB) Close() error {
	var allErrors []error
	for _, driver := range db.readingRoutingMap {
		err := driver.Close()
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	if db.writingDriver != nil {
		err := db.writingDriver.Close()
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	// Combine all errors if present, it's a no-op if no error present (returns nil)
	return multierr.Combine(allErrors...)
}
