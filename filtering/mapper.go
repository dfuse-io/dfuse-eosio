// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filtering

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/document"
	"github.com/blevesearch/bleve/mapping"
	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/search"
	"github.com/google/cel-go/cel"
	"go.uber.org/zap"
)

type batchActionUpdater = func(trxID string, idx int, data map[string]interface{}) error

// eventsConfig contains specific configuration for the correct indexation of
// dfuse Events, our special methodology to index the action from your smart contract
// the way the developer like it.
type eventsConfig struct {
	actionName   string
	unrestricted bool
}

type BlockMapper struct {
	*mapping.IndexMappingImpl

	eventsConfig     eventsConfig
	filterOnProgram  cel.Program
	filterOutProgram cel.Program
	indexed          *IndexedTerms
	isUnfiltered     bool
}

func NewBlockMapper(eventsActionName string, eventsUnrestricted bool, filterOn, filterOut, indexedTermsSpecs string) (*BlockMapper, error) {
	fonProgram, err := buildCELProgram("true", filterOn)
	if err != nil {
		return nil, err
	}

	foutProgram, err := buildCELProgram("false", filterOut)
	if err != nil {
		return nil, err
	}

	indexed, err := NewIndexedTerms(indexedTermsSpecs)
	if err != nil {
		return nil, err
	}

	return &BlockMapper{
		IndexMappingImpl: buildBleveIndexMapper(),
		eventsConfig: eventsConfig{
			actionName:   eventsActionName,
			unrestricted: eventsUnrestricted,
		},
		filterOnProgram:  fonProgram,
		filterOutProgram: foutProgram,
		indexed:          indexed,
		isUnfiltered:     fonProgram == nil && foutProgram == nil,
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

func (m *BlockMapper) IsUnfiltered() bool          { return m.isUnfiltered }
func (m *BlockMapper) IndexedTerms() *IndexedTerms { return m.indexed }

func (m *BlockMapper) MapForDB(blk *pbcodec.Block) (matchingTrxs map[string]bool, actions []*pbcodec.ActionTrace, err error) {
	matchingTrxs = map[string]bool{}
	for _, trxTrace := range blk.TransactionTraces {
		trxID := trxTrace.Id
		if !isTrxTraceIndexable(trxTrace) {
			continue
		}

		scheduled := trxTrace.Scheduled

		for _, actTrace := range trxTrace.ActionTraces {
			if !m.shouldIndexAction(ActionTraceActivation{
				trxScheduled: scheduled,
				trace:        actTrace,
			}) {
				continue
			}

			matchingTrxs[trxID] = true

			actions = append(actions, actTrace)
		}
	}
	return
}

func (m *BlockMapper) MapToBleve(block *bstream.Block) ([]*document.Document, error) {
	blk := block.ToNative().(*pbcodec.Block)

	actionsCount := 0
	var docsList []*document.Document
	batchActionUpdater := func(trxID string, idx int, data map[string]interface{}) error {
		if !m.shouldIndexAction(data) {
			return nil
		}

		if blk.Num() == uint64(125000550) {
			cnt, err := json.MarshalIndent(data, "", "  ")
			fmt.Println("DATAAAAAAAAAAAAAA:", string(cnt), err)
		}

		doc := document.NewDocument(EOSDocumentID(blk.Num(), trxID, idx))
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

	fmt.Println("SEARCH RESULTS", blk.Num(), len(docsList))

	return docsList, nil
}

func (m *BlockMapper) prepareBatchDocuments(blk *pbcodec.Block, batchUpdater batchActionUpdater) error {
	trxIndex := -1
	for _, trxTrace := range blk.TransactionTraces {
		trxIndex++

		trxID := trxTrace.Id
		if !isTrxTraceIndexable(trxTrace) {
			continue
		}

		scheduled := trxTrace.Scheduled

		type prepedDoc struct {
			trxID string
			idx   int
			data  map[string]interface{}
		}

		tokenizedActions := map[uint32]prepedDoc{}

		for idx, actTrace := range trxTrace.ActionTraces {
			data := tokenizeEOSExecutedAction(actTrace)
			// `block_num`, `trx_idx`: used for sorting
			data["block_num"] = blk.Num()
			data["trx_idx"] = trxIndex /// FIXME: trxTrace.Index

			receiver := string(actTrace.Receipt.Receiver)
			account := string(actTrace.Action.Account)
			if m.indexed.Notif {
				data["notif"] = receiver != account
			}
			if m.indexed.Input {
				// TODO: check the Ultra chain: what will that mean
				// with the additional predicate actions?
				data["input"] = actTrace.CreatorActionOrdinal == 0
			}
			if m.indexed.Scheduled {
				data["scheduled"] = scheduled
			}

			if actTrace.SimpleName() == m.eventsConfig.actionName && actTrace.CreatorActionOrdinal != 0 {
				eventFields := tokenizeEvent(m.eventsConfig, actTrace.GetData("key").String(), actTrace.GetData("data").String())
				if len(eventFields) > 0 {
					tokenizedActions[actTrace.CreatorActionOrdinal].data["event"] = eventFields
				}
			}

			// NOTE: we're still missing the RAM accounting for `hard_fail` and
			// `expired` transactions.  `expired` transactions do not have actions, so
			// we can't even have something to index here.

			ramOps := trxTrace.RAMOpsForAction(uint32(idx))

			ramData := m.processRAMOps(ramOps)
			if len(ramData) > 0 {
				data["ram"] = ramData
			}

			dbData := m.processDBOps(trxTrace.DBOpsForAction(uint32(idx)))
			if len(dbData) > 0 {
				data["db"] = dbData
			}

			tokenizedActions[actTrace.ActionOrdinal] = prepedDoc{
				trxID: trxID,
				idx:   idx,
				data:  data,
			}
		}

		// Loop and batch update all the actions
		for _, doc := range tokenizedActions {
			err := batchUpdater(doc.trxID, doc.idx, doc.data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func isTrxTraceIndexable(trxTrace *pbcodec.TransactionTrace) bool {
	if trxTrace.Receipt == nil {
		return false
	}

	status := trxTrace.Receipt.Status
	if status == pbcodec.TransactionStatus_TRANSACTIONSTATUS_SOFTFAIL {
		// We index `eosio:onerror` transaction that are in soft_fail state since it means a valid `onerror` handler execution
		return len(trxTrace.ActionTraces) >= 1 && trxTrace.ActionTraces[0].SimpleName() == "eosio:onerror"
	}

	return status == pbcodec.TransactionStatus_TRANSACTIONSTATUS_EXECUTED
}

func (m *BlockMapper) processRAMOps(ramOps []*pbcodec.RAMOp) map[string][]string {
	consumedRAM := make(map[string]bool)
	releasedRAM := make(map[string]bool)
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
	// ram.changed:eoscanadacom
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
	// db.key = []string{"accounts/eoscanadacom/.........eioh1"}
	// db.table = []string{"accounts/eoscanadacom", "accounts"}
	keys := make(map[string]bool)
	tables := make(map[string]bool)
	for _, op := range dbOps {
		keys[fmt.Sprintf("%s/%s/%s", op.TableName, op.Scope, op.PrimaryKey)] = true
		tables[fmt.Sprintf("%s/%s", op.TableName, op.Scope)] = true
		tables[string(op.TableName)] = true
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

func EOSDocumentID(blockNum uint64, transactionID string, actionIndex int) string {
	// 128 bits collision protection
	return fmt.Sprintf("%016x", blockNum) + ":" + transactionID[:32] + ":" + fmt.Sprintf("%04x", actionIndex)
}

func ExplodeEOSDocumentID(ref string) (blockNum uint64, trxID string, actionIdx uint16, skip bool) {
	var err error
	chunks := strings.Split(ref, ":")
	chunksCount := len(chunks)
	if chunksCount != 3 || chunks[0] == "meta" { // meta, flatten, etc.
		skip = true
		return
	}

	blockNum32, err := fromHexUint32(chunks[0])
	if err != nil {
		zlog.Panic("woah, block num invalid?", zap.Error(err))
	}

	blockNum = uint64(blockNum32)

	trxID = chunks[1]
	actionIdx, err = fromHexUint16(chunks[2])
	if err != nil {
		zlog.Panic("woah, action index invalid?", zap.Error(err))
	}

	return
}
