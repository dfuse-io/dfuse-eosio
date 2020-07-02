package search

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
	"go.uber.org/zap"
)

type eosBatchActionUpdater = func(trxID string, idx int, data map[string]interface{}) error

// eventsConfig contains specific configuration for the correct indexation of
// dfuse Events, our special methodology to index the action from your smart contract
// the way the developer like it.
type eventsConfig struct {
	actionName   string
	unrestricted bool
}

type EOSBlockMapper struct {
	eventsConfig eventsConfig
	restrictions []*restriction
}

func NewEOSBlockMapper(eventsActionName string, eventsUnrestricted bool) (*EOSBlockMapper, error) {
	return &EOSBlockMapper{
		eventsConfig: eventsConfig{
			actionName:   eventsActionName,
			unrestricted: eventsUnrestricted,
		},
	}, nil
}

func (m *EOSBlockMapper) IndexMapping() *mapping.IndexMappingImpl {
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

func (m *EOSBlockMapper) Map(mapper *mapping.IndexMappingImpl, block *bstream.Block) ([]*document.Document, error) {
	blk := block.ToNative().(*pbcodec.Block)

	actionsCount := 0
	var docsList []*document.Document
	batchActionUpdater := func(trxID string, idx int, data map[string]interface{}) error {
		doc := document.NewDocument(EOSDocumentID(blk.Num(), trxID, idx))
		err := mapper.MapDocument(doc, data)
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
	err = mapper.MapDocument(metaDoc, map[string]interface{}{
		"act_count": actionsCount,
	})

	if err != nil {
		return nil, err
	}

	docsList = append(docsList, metaDoc)

	return docsList, nil
}

func parseRestrictionsJSON(JSONStr string) ([]*restriction, error) {
	var restrictions []*restriction
	if JSONStr == "" {
		return nil, nil
	}
	err := json.Unmarshal([]byte(JSONStr), &restrictions)
	return restrictions, err

}

// example: `{"account":"eosio.token","data.to":"someaccount"}` will not Pass()
// true for an action that matches EXACTLY those two conditions
type restriction map[string]string

func (r restriction) Pass(actionWrapper map[string]interface{}) bool {
	actionData, _ := actionWrapper["data"].(map[string]interface{})

	for k, v := range r {
		if strings.HasPrefix(k, "data.") {
			if val, ok := actionData[k[5:]]; !ok || val != v {
				return true
			}
			continue
		}
		if val, ok := actionWrapper[k]; !ok || val != v {
			return true
		}
	}
	return false
}

func (m *EOSBlockMapper) prepareBatchDocuments(blk *pbcodec.Block, batchUpdater eosBatchActionUpdater) error {
	for _, trxTrace := range blk.TransactionTraces() {
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

		for idx, actTrace := range trxTrace.ActionTraces {
			data := tokenizeEOSExecutedAction(actTrace)
			// `block_num`, `trx_idx`: used for sorting
			data["block_num"] = blk.Num()
			data["trx_idx"] = trxTrace.Index

			receiver := string(actTrace.Receipt.Receiver)
			account := string(actTrace.Action.Account)
			data["notif"] = receiver != account
			data["input"] = actTrace.CreatorActionOrdinal == 0
			data["scheduled"] = scheduled
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

func (m *EOSBlockMapper) processRAMOps(ramOps []*pbcodec.RAMOp) map[string][]string {
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

func (m *EOSBlockMapper) processDBOps(dbOps []*pbcodec.DBOp) map[string][]string {
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
