package filtering

import (
	"encoding/json"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/google/cel-go/interpreter"
	"github.com/tidwall/gjson"
)

type ActionTraceActivation struct {
	trace *pbcodec.ActionTrace

	trxScheduled bool
}

func (a ActionTraceActivation) Parent() interpreter.Activation {
	return nil
}

func (a ActionTraceActivation) ResolveName(name string) (interface{}, bool) {
	switch name {
	case "receiver":
		if a.trace.Receipt != nil {
			return a.trace.Receipt.Receiver, true
		}
		return a.trace.Receiver, true
	case "account":
		return a.trace.Account(), true
	case "action":
		return a.trace.Name(), true
	case "auth":
		return tokenizeEOSAuthority(a.trace.Action.Authorization), true
	case "data":
		if len(a.trace.Action.JsonData) == 0 {
			return nil, false
		}
		var out map[string]interface{}
		err := json.Unmarshal([]byte(a.trace.Action.JsonData), &out)
		if err != nil {
			fmt.Println("Invalid jsondata:", a.trace.Action.JsonData)
			return nil, false
		}
		return out, true
		//return DataActivation{parent: a, Result: gjson.Parse(a.trace.Action.JsonData)}, true
	case "notif":
		receiver := a.trace.Receiver
		if a.trace.Receipt != nil {
			receiver = a.trace.Receipt.Receiver
		}
		return a.trace.Account() != receiver, true
	case "scheduled":
		return a.trxScheduled, true
	case "input":
		return a.trace.CreatorActionOrdinal == 0, true
	case "ram":
		panic("CEL filtering does not yet support ram.consumed nor ram.released")
	case "db":
		panic("CEL filtering does not yet support db.table, db.key, etc..")
	}
	return nil, false
}

type DataActivation struct {
	parent ActionTraceActivation
	gjson.Result
}

func (a DataActivation) Parent() interpreter.Activation {
	return a.parent
}

func (a DataActivation) ResolveName(name string) (interface{}, bool) {
	fmt.Println("Querying", name, "on data activation")
	res := a.Get(name)
	if len(res.Raw) == 0 {
		return nil, false
	}
	return res.Value(), true
}
