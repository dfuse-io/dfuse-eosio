package search

import (
	"encoding/json"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/linkedin/goavro/v2"
)

type EOSBigQueryBlockMapper struct {
	baseMapper *EOSBlockMapper
	codec *goavro.Codec
}

func NewEOSBigQueryBlockMapper(eventsActionName string, eventsUnrestricted bool, filterOn, filterOut string) (*EOSBigQueryBlockMapper, error) {
	base, err := NewEOSBlockMapper(eventsActionName, eventsUnrestricted, filterOn, filterOut)
	if err != nil {
		return nil, err
	}

	///////////////////////////////////////////////////
	// **************** WARNING *********************
	// If you need to add new field in the codec, you need to make it backward compatible.
	// This means that you must have a default value corresponding to that new field.
	// ex. { "name": "NEW_FIELD_NAME", "type": "string", "default": "" }
	///////////////////////////////////////////////////
	codec, err := goavro.NewCodec(`{
		"namespace": "io.dfuse",
		"name": "EOSAction",
		"type": "record",
		"fields": [
			{ "name": "block_num", "type": "long" },
			{ "name": "block_id", "type": "string" },
			{ "name": "block_time", "type" : {"type": "long", "logicalType" : "timestamp-millis"} },
			{ "name": "trx_id", "type": "string" },
			{ "name": "act_idx", "type": "long" },
			{ "name": "trx_idx", "type": "long" },

			{ "name": "receiver", "type": "string" },
			{ "name": "account", "type": "string" },
			{ "name": "action", "type": "string" },
			{ "name": "auth", "type": "string" },
			{ "name": "input", "type": "boolean" },
			{ "name": "notif", "type": "boolean" },
			{ "name": "scheduled", "type": "boolean" },

			{
				"name": "db",
				"type": {
                	"type": "array",  
                	"items":{
                    	"name": "DBOp",
                    	"type": "record",
                    	"fields": [
							{ "name": "key", "type": "string" },
							{ "name": "table", "type": "string" }
                    	]
                	}
            	}
        	},
			{
				"name": "ram",
				"type": {
                	"type": "array",  
                	"items":{
                    	"name": "DBOp",
                    	"type": "record",
                    	"fields": [
							{ "name": "consumed", "type": "string" },
							{ "name": "released", "type": "string" }
                    	]
                	}
            	}
        	},

			{ "name": "event", "type": "string" },
			{ "name": "data", "type": "string" }
		]
	}`)
	if err != nil {
		return nil, err
	}

	return &EOSBigQueryBlockMapper{
		baseMapper: base,
		codec: codec,
	}, nil
}

func (m *EOSBigQueryBlockMapper) Map(block *bstream.Block) ([]map[string]interface{}, error) {
	blk := block.ToNative().(*pbcodec.Block)

	var mappedActionsList []map[string]interface{}
	batchActionUpdater := func(trxID string, idx int, data map[string]interface{}) error {
		if !m.baseMapper.shouldIndexAction(data) {
			return nil
		}

		// Add extra metadata
		data["block_num"] = blk.Num()
		data["block_id"] = blk.ID()
		data["block_time"] = blk.MustTime()
		data["trx_id"] = trxID

		// Store nested stuff as JSON
		if data["event"] != nil {
			asJson, err := json.Marshal(data["event"])
			if err != nil {
				return err
			}
			data["event"] = asJson
		}
		if data["data"] != nil {
			asJson, err := json.Marshal(data["data"])
			if err != nil {
				return err
			}
			data["data"] = asJson
		}

		mappedActionsList = append(mappedActionsList, data)
		return nil
	}

	err := m.baseMapper.prepareBatchDocuments(blk, batchActionUpdater)
	if err != nil {
		return nil, err
	}

	return mappedActionsList, nil
}
