package eosdbtest

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"github.com/dfuse-io/dfuse-eosio/codec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/jsonpb"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/golang/protobuf/ptypes"
)

func testBlock1() *pbcodec.Block {
	blockTime, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	blockTimestamp, _ := ptypes.TimestampProto(blockTime)

	trx := &eos.Transaction{
		TransactionHeader: eos.TransactionHeader{
			Expiration:     eos.JSONTime{blockTime},
			RefBlockNum:    123,
			RefBlockPrefix: 234,
		},
		Actions: []*eos.Action{
			{
				Account:    "some",
				Name:       "name",
				ActionData: eos.NewActionDataFromHexData([]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}),
			},
		},
	}
	signedTrx := eos.NewSignedTransaction(trx)
	signedTrx.Signatures = append(signedTrx.Signatures, ecc.MustNewSignature("SIG_K1_K7kTcvsznS2pSQ2unjW9nduqHieWnc5B6rFdbVif4RM1DCTVhQUpzwng3XTGewDhVZqNvqSAEwHgB8yBnfDYAHquRX4fBo"))
	packed, err := signedTrx.Pack(eos.CompressionNone)
	if err != nil {
		panic(err)
	}
	trxID, _ := hex.DecodeString("00112233aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	receipt := &eos.TransactionReceipt{
		TransactionReceiptHeader: eos.TransactionReceiptHeader{
			Status:               eos.TransactionStatusExecuted,
			CPUUsageMicroSeconds: 32,
			NetUsageWords:        eos.Varuint32(32),
		},
		Transaction: eos.TransactionWithID{
			ID:     eos.Checksum256([]byte(trxID)),
			Packed: packed,
		},
	}

	pbblock := &pbcodec.Block{
		Id:                       "00000002aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Number:                   2,
		DposIrreversibleBlocknum: 1,
		Header: &pbcodec.BlockHeader{
			Previous:  "00000001aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Producer:  "tester",
			Timestamp: blockTimestamp,
		},
		Transactions: []*pbcodec.TransactionReceipt{
			codec.TransactionReceiptToDEOS(receipt),
		},
		ImplicitTransactionOps: []*pbcodec.TrxOp{
			{
				Operation:     pbcodec.TrxOp_OPERATION_CREATE,
				Name:          "onblock",
				TransactionId: "abc999aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Transaction: &pbcodec.SignedTransaction{
					Transaction: &pbcodec.Transaction{},
				},
			},
		},
		TransactionTraces: []*pbcodec.TransactionTrace{
			{
				Id: "00112233aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				DtrxOps: []*pbcodec.DTrxOp{
					{
						Operation:     pbcodec.DTrxOp_OPERATION_CREATE,
						TransactionId: "aaa777aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						Transaction: &pbcodec.SignedTransaction{
							Transaction: &pbcodec.Transaction{},
						},
					},
					{
						Operation:     pbcodec.DTrxOp_OPERATION_CANCEL,
						TransactionId: "aaa888aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						Transaction: &pbcodec.SignedTransaction{
							Transaction: &pbcodec.Transaction{},
						},
					},
				},
				ActionTraces: []*pbcodec.ActionTrace{
					{
						Receiver: "eosio",
						Action: &pbcodec.Action{
							Account:  "eosio",
							Name:     "newaccount",
							JsonData: `{"creator": "frankenstein", "name": "createdacct"}`,
						},
					},
				},
			},
		},
	}

	if os.Getenv("DEBUG") != "" {
		marshaler := &jsonpb.Marshaler{}
		out, err := marshaler.MarshalToString(pbblock)
		if err != nil {
			panic(err)
		}

		// We re-normalize to a plain map[string]interface{} so it's printed as JSON and not a proto default String implementation
		normalizedOut := map[string]interface{}{}
		err = json.Unmarshal([]byte(out), &normalizedOut)
		if err != nil {
			panic(err)
		}

		//zlog.Debug("created test block", zap.Any("block", normalizedOut))
	}

	return pbblock
}
