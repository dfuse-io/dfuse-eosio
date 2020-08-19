package search

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
)

func Test_validateQueryFields(t *testing.T) {
	tests := []struct {
		in            string
		expectedError error
	}{
		{
			"account:eoscanadacom",
			nil,
		},
		{
			"unknow:eoscanadacom",
			derr.Status(codes.InvalidArgument, "The following fields you are trying to search are not currently indexed: 'unknow'. Contact our support team for more."),
		},
		{
			"unknow:eoscanadacom second:test",
			derr.Status(codes.InvalidArgument, "The following fields you are trying to search are not currently indexed: 'second', 'unknow'. Contact our support team for more."),
		},
		{
			"unknow:eoscanadacom account:value second:test",
			derr.Status(codes.InvalidArgument, "The following fields you are trying to search are not currently indexed: 'second', 'unknow'. Contact our support team for more."),
		},
		{
			"data.from:eoscanadacom data.nested:value account:test",
			derr.Status(codes.InvalidArgument, "The following fields you are trying to search are not currently indexed: 'data.nested'. Contact our support team for more."),
		},
		{
			"data.from:eoscanadacom data.nested.deep:value account:test",
			derr.Status(codes.InvalidArgument, "The following fields you are trying to search are not currently indexed: 'data.nested.deep'. Contact our support team for more."),
		},
		{
			"data.from.something:value data.auth.keys.key:value",
			nil,
		},
		{
			"event.field1:value event.field2.nested:value",
			nil,
		},
		{
			"data.from:eoscanadacom data.:value account:test",
			derr.Status(codes.InvalidArgument, "The following fields you are trying to search are not currently indexed: 'data.'. Contact our support team for more."),
		},
	}

	for idx, test := range tests {
		t.Run(fmt.Sprintf("index %d", idx+1), func(t *testing.T) {
			_, err := search.NewParsedQuery(test.in)
			if test.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.JSONEq(t, toJSONString(t, test.expectedError), toJSONString(t, err))
			}
		})
	}
}

func toJSONString(t *testing.T, v interface{}) string {
	t.Helper()

	out, err := json.Marshal(v)
	require.NoError(t, err)

	return string(out)
}

func fixedTraceID(hexInput string) (out trace.TraceID) {
	rawTraceID, _ := hex.DecodeString(hexInput)
	copy(out[:], rawTraceID)

	return
}
