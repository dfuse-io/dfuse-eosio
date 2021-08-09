package search

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/document"
	"github.com/blevesearch/bleve/mapping"
	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/streamingfast/search"
	"go.uber.org/zap"
)

type BatchActionUpdater = func(trxID string, idx int, data map[string]interface{}) error

// eventsConfig contains specific configuration for the correct indexation of
// dfuse Events, our special methodology to index the action from your smart contract
// the way the developer like it.
type eventsConfig struct {
	actionName   string
	unrestricted bool
}

type BlockMapper struct {
	*mapping.IndexMappingImpl

	eventsConfig eventsConfig
	indexed      *IndexedTerms
	tokenizer    tokenizer
}

func NewBlockMapper(eventsActionName string, eventsUnrestricted bool, indexedTermsSpecs string) (*BlockMapper, error) {
	indexed, err := NewIndexedTerms(indexedTermsSpecs)
	if err != nil {
		return nil, fmt.Errorf("indexed terms: %w", err)
	}

	return &BlockMapper{
		IndexMappingImpl: buildBleveIndexMapper(),

		eventsConfig: eventsConfig{
			actionName:   eventsActionName,
			unrestricted: eventsUnrestricted,
		},
		indexed:   indexed,
		tokenizer: tokenizer{indexedTerms: indexed},
	}, nil
}

func buildBleveIndexMapper() *mapping.IndexMappingImpl {
	// db ops
	dbDocMapping := bleve.NewDocumentMapping()
	dbDocMapping.AddFieldMappingsAt("key", search.TxtFieldMapping)
	dbDocMapping.AddFieldMappingsAt("table", search.TxtFieldMapping)

	// ram ops
	ramDocMapping := bleve.NewDocumentMapping()
	ramDocMapping.AddFieldMappingsAt("consumed", search.TxtFieldMapping)
	ramDocMapping.AddFieldMappingsAt("released", search.TxtFieldMapping)

	// Root doc
	rootDocMapping := bleve.NewDocumentStaticMapping()

	// Sortable internals
	rootDocMapping.AddFieldMappingsAt("act_idx", search.SortableNumericFieldMapping)
	rootDocMapping.AddFieldMappingsAt("block_num", search.SortableNumericFieldMapping)
	rootDocMapping.AddFieldMappingsAt("trx_idx", search.SortableNumericFieldMapping)

	rootDocMapping.AddFieldMappingsAt("receiver", search.TxtFieldMapping)
	rootDocMapping.AddFieldMappingsAt("account", search.TxtFieldMapping)
	rootDocMapping.AddFieldMappingsAt("action", search.TxtFieldMapping)
	rootDocMapping.AddFieldMappingsAt("auth", search.TxtFieldMapping)
	rootDocMapping.AddFieldMappingsAt("input", search.BoolFieldMapping)
	rootDocMapping.AddFieldMappingsAt("notif", search.BoolFieldMapping)
	rootDocMapping.AddFieldMappingsAt("scheduled", search.BoolFieldMapping)

	// add other sub-sections here
	rootDocMapping.AddSubDocumentMapping("data", search.DynamicNestedDocMapping)
	rootDocMapping.AddSubDocumentMapping("db", dbDocMapping)
	rootDocMapping.AddSubDocumentMapping("ram", ramDocMapping)
	rootDocMapping.AddSubDocumentMapping("event", search.DynamicNestedDocMapping)

	// this disables the _all field
	rootDocMapping.AddSubDocumentMapping("_all", search.DisabledMapping)

	mapper := bleve.NewIndexMapping()
	mapper.DefaultAnalyzer = keyword.Name
	mapper.StoreDynamic = false
	mapper.IndexDynamic = true
	mapper.DocValuesDynamic = false
	mapper.DefaultMapping = rootDocMapping

	return mapper
}

func (m *BlockMapper) IndexedTerms() *IndexedTerms { return m.indexed }

func (m *BlockMapper) Map(block *bstream.Block) ([]*document.Document, error) {
	blk := block.ToNative().(*pbcodec.Block)

	actionsCount := 0
	var docsList []*document.Document
	batchActionUpdater := func(trxID string, idx int, data map[string]interface{}) error {
		doc := document.NewDocument(newDocumentID(blk.Num(), trxID, idx))
		if traceEnabled {
			zlog.Debug("adding document to docs list", zap.String("id", doc.ID), zap.Int("size", doc.Size()), zap.Reflect("data", data))
		}

		err := m.MapDocument(doc, data)
		if err != nil {
			return err
		}

		actionsCount++
		docsList = append(docsList, doc)

		return nil
	}

	err := m.prepareBatchDocuments(blk, batchActionUpdater)
	if err != nil {
		return nil, err
	}

	metaDoc := document.NewDocument(fmt.Sprintf("meta:blknum:%d", blk.Num()))
	err = m.MapDocument(metaDoc, map[string]interface{}{
		"act_count": actionsCount,
	})

	if err != nil {
		return nil, err
	}

	docsList = append(docsList, metaDoc)

	return docsList, nil
}

func (m *BlockMapper) prepareBatchDocuments(blk *pbcodec.Block, batchUpdater BatchActionUpdater) error {
	for _, trxTrace := range blk.TransactionTraces() {
		// We only index transaction trace that were correctly recorded in the blockchain
		if trxTrace.HasBeenReverted() {
			continue
		}

		trxID := trxTrace.Id
		scheduled := trxTrace.Scheduled

		type prepedDoc struct {
			trxID string
			idx   int
			data  map[string]interface{}
		}

		tokenizedActions := map[uint32]prepedDoc{}
		actionMatcher := blk.FilteringActionMatcher(trxTrace, isRequiredSystemAction)

		for idx, actTrace := range trxTrace.ActionTraces {
			if !actionMatcher.Matched(actTrace.ExecutionIndex) {
				continue
			}

			data := m.tokenizer.tokenize(actTrace)
			// `block_num`, `trx_idx`: used for sorting
			data["block_num"] = blk.Num()
			data["trx_idx"] = trxTrace.Index

			if m.indexed.Notif {
				data["notif"] = actTrace.Receipt.Receiver != actTrace.Action.Account
			}
			if m.indexed.Input {
				data["input"] = actTrace.IsInput()
			}
			if m.indexed.Scheduled {
				data["scheduled"] = scheduled
			}

			if m.indexed.Event {
				if actTrace.SimpleName() == m.eventsConfig.actionName && !actTrace.IsInput() {
					eventFields := m.tokenizer.tokenizeEvent(m.eventsConfig, actTrace.GetData("key").String(), actTrace.GetData("data").String())
					if len(eventFields) > 0 {
						tokenizedActions[actTrace.CreatorActionOrdinal].data["event"] = eventFields
					}
				}
			}

			// NOTE: we're still missing the RAM accounting for `hard_fail` and
			// `expired` transactions.  `expired` transactions do not have actions, so
			// we can't even have something to index here.

			if m.indexed.RAMConsumed || m.indexed.RAMReleased {
				ramData := m.processRAMOps(trxTrace.RAMOpsForAction(uint32(idx)))
				if len(ramData) > 0 {
					data["ram"] = ramData
				}
			}

			if m.indexed.DBKey || m.indexed.DBTable {
				dbData := m.processDBOps(trxTrace.DBOpsForAction(uint32(idx)))
				if len(dbData) > 0 {
					data["db"] = dbData
				}
			}

			tokenizedActions[actTrace.ActionOrdinal] = prepedDoc{
				trxID: trxID,
				idx:   idx,
				data:  data,
			}
		}

		// Loop and batch update all the actions
		if traceEnabled {
			zlog.Debug("mapped block to documents", zap.Int("action_count", len(tokenizedActions)), zap.Stringer("block", blk.AsRef()))
		}

		for _, doc := range tokenizedActions {
			err := batchUpdater(doc.trxID, doc.idx, doc.data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func isRequiredSystemAction(actTrace *pbcodec.ActionTrace) bool {
	return actTrace.Receiver == "eosio" && actTrace.Action.Account == "eosio" && actTrace.Action.Name == "setabi"
}

func (m *BlockMapper) processRAMOps(ramOps []*pbcodec.RAMOp) map[string][]string {
	if len(ramOps) <= 0 {
		return nil
	}

	var consumedRAM map[string]bool
	if m.indexed.RAMConsumed {
		consumedRAM = make(map[string]bool)
	}

	var releasedRAM map[string]bool
	if m.indexed.RAMReleased {
		releasedRAM = make(map[string]bool)
	}

	for _, ramop := range ramOps {
		if ramop.Delta == 0 {
			continue
		}

		if ramop.Delta > 0 {
			consumedRAM[ramop.Payer] = true
		} else {
			releasedRAM[ramop.Payer] = true
		}
	}

	ramData := make(map[string][]string)
	if len(consumedRAM) != 0 {
		ramData["consumed"] = toList(consumedRAM)
	}
	if len(releasedRAM) != 0 {
		ramData["released"] = toList(releasedRAM)
	}

	return ramData
}

func (m *BlockMapper) processDBOps(dbOps []*pbcodec.DBOp) map[string][]string {
	dbOpCount := len(dbOps)
	if dbOpCount <= 0 {
		return nil
	}

	var keys map[string]bool
	if m.indexed.DBKey {
		keys = make(map[string]bool, dbOpCount)
	}

	var tables map[string]bool
	if m.indexed.DBTable {
		tables = make(map[string]bool, dbOpCount)
	}

	for _, op := range dbOps {
		if m.indexed.DBKey {
			keys[fmt.Sprintf("%s/%s/%s", op.TableName, op.Scope, op.PrimaryKey)] = true
		}

		if m.indexed.DBTable {
			tables[fmt.Sprintf("%s/%s", op.TableName, op.Scope)] = true
			tables[string(op.TableName)] = true
		}
	}

	opData := make(map[string][]string)
	if len(keys) != 0 {
		opData["key"] = toList(keys)
	}

	if len(tables) != 0 {
		opData["table"] = toList(tables)
	}

	return opData
}

func newDocumentID(blockNum uint64, transactionID string, actionIndex int) string {
	// 128 bits collision protection
	return fmt.Sprintf("%016x", blockNum) + ":" + transactionID[:32] + ":" + fmt.Sprintf("%04x", actionIndex)
}

func explodeDocumentID(ref string) (blockNum uint64, trxID string, actionIdx uint16, skip bool) {
	var err error
	chunks := strings.Split(ref, ":")
	chunksCount := len(chunks)
	if chunksCount != 3 || chunks[0] == "meta" { // meta, flatten, etc.
		skip = true
		return
	}

	blockNum32, err := fromHexUint32(chunks[0])
	if err != nil {
		zlog.Panic("block num invalid", zap.Error(err))
	}

	blockNum = uint64(blockNum32)

	trxID = chunks[1]
	actionIdx, err = fromHexUint16(chunks[2])
	if err != nil {
		zlog.Panic("action index invalid", zap.Error(err))
	}

	return
}
