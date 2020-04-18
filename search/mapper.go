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
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"go.uber.org/zap"
)

type eosBatchActionUpdater = func(trxID string, idx int, data map[string]interface{}) error

type EOSBlockMapper struct {
	hooksActionName  string
	restrictions     []*restriction
	filterOnProgram  cel.Program
	filterOutProgram cel.Program
}

func NewEOSBlockMapper(hooksActionName string, filterOn, filterOut string) (*EOSBlockMapper, error) {
	fonProgram, err := buildCELProgram("true", filterOn)
	if err != nil {
		return nil, err
	}

	foutProgram, err := buildCELProgram("false", filterOut)
	if err != nil {
		return nil, err
	}

	return &EOSBlockMapper{
		hooksActionName:  hooksActionName,
		filterOnProgram:  fonProgram,
		filterOutProgram: foutProgram,
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
		if !m.shouldIndexAction(data) {
			return nil
		}

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

func buildCELProgram(noopProgram string, programString string) (cel.Program, error) {
	stripped := strings.TrimSpace(programString)
	if stripped == "" || stripped == noopProgram {
		return nil, nil
	}

	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewIdent("db", decls.NewMapType(decls.String, decls.String), nil), // "table", "key" => string
			decls.NewIdent("data", decls.NewMapType(decls.String, decls.Any), nil),
			decls.NewIdent("ram", decls.NewMapType(decls.String, decls.String), nil),
			decls.NewIdent("receiver", decls.String, nil),
			decls.NewIdent("account", decls.String, nil),
			decls.NewIdent("action", decls.String, nil),
			decls.NewIdent("auth", decls.NewListType(decls.String), nil),
			decls.NewIdent("input", decls.Bool, nil),
			decls.NewIdent("notif", decls.Bool, nil),
			decls.NewIdent("scheduled", decls.Bool, nil),
		),
	)
	if err != nil {
		return nil, err
	}

	exprAst, issues := env.Compile(programString)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("filter expression parse/check error: %w", issues.Err())
	}

	if exprAst.ResultType() != decls.Bool {
		return nil, fmt.Errorf("filter expression should return a boolean, returned %s", exprAst.ResultType())
	}

	prg, err := env.Program(exprAst)
	if err != nil {
		return nil, fmt.Errorf("cel program construction error: %w", err)
	}

	return prg, nil
}

func (m *EOSBlockMapper) shouldIndexAction(doc map[string]interface{}) bool {
	filterOnResult := m.filterMatches(m.filterOnProgram, true, doc)
	filterOutResult := m.filterMatches(m.filterOutProgram, false, doc)
	return filterOnResult && !filterOutResult
}

func (m *EOSBlockMapper) filterMatches(program cel.Program, defaultVal bool, doc map[string]interface{}) bool {
	if program == nil {
		return defaultVal
	}

	res, _, err := program.Eval(doc)
	if err != nil {
		//fmt.Printf("filter program: %s\n", err.Error())
		return false
	}
	retval, valid := res.(types.Bool)
	if !valid {
		// TODO: use logger, we've checked the return value should be a Bool previously, so
		// it's even safe to panic here
		panic("return value of our cel program isn't of type bool")
	}
	return bool(retval)
}

func (m *EOSBlockMapper) prepareBatchDocuments(blk *pbcodec.Block, batchUpdater eosBatchActionUpdater) error {
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
			data["trx_idx"] = trxIndex

			receiver := string(actTrace.Receipt.Receiver)
			account := string(actTrace.Action.Account)
			data["notif"] = receiver != account
			data["input"] = actTrace.CreatorActionOrdinal == 0
			data["scheduled"] = scheduled
			if actTrace.SimpleName() == m.hooksActionName && actTrace.CreatorActionOrdinal != 0 {
				eventFields := tokenizeEvent(actTrace.GetData("key").String(), actTrace.GetData("data").String())
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
