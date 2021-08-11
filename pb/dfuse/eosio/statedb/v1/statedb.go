package pbstatedb

import (
	context "context"
	"errors"
	fmt "fmt"
	"io"
	"strconv"

	"github.com/streamingfast/bstream"
	grpc "google.golang.org/grpc"
)

const MetdataUpToBlockID = "statedb-up-to-block-id"
const MetdataUpToBlockNum = "statedb-up-to-block-num"
const MetdataLastIrrBlockID = "statedb-last-irr-block-id"
const MetdataLastIrrBlockNum = "statedb-last-irr-block-num"

var ErrStreamReferenceNotFound = errors.New("not found")
var SkipTable = errors.New("skip table")

func ForEachMultiScopesTableRows(ctx context.Context, client StateClient, request *StreamMultiScopesTableRowsRequest, onEach func(scope string, row *TableRowResponse) error) (*StreamReference, error) {
	stream, err := client.StreamMultiScopesTableRows(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("new stream: %w", err)
	}

	ref, err := ExtractStreamReference(stream)
	if err != nil {
		return nil, fmt.Errorf("stream reference: %w", err)
	}

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return ref, nil
			}

			return nil, fmt.Errorf("stream: %w", err)
		}

		for _, row := range response.Rows {
			err = onEach(response.Scope, row)
			if err != nil {
				if err == SkipTable {
					break
				}

				return nil, err
			}
		}
	}
}

func ForEachTableRows(ctx context.Context, client StateClient, request *StreamTableRowsRequest, onEach func(response *TableRowResponse) error) (*StreamReference, error) {
	stream, err := client.StreamTableRows(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("new stream: %w", err)
	}

	ref, err := ExtractStreamReference(stream)
	if err != nil {
		return nil, fmt.Errorf("stream reference: %w", err)
	}

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return ref, nil
			}

			return nil, fmt.Errorf("stream: %w", err)
		}

		err = onEach(response)
		if err != nil {
			return nil, err
		}
	}
}

func FetchTableScopes(ctx context.Context, client StateClient, blockNum uint64, contract, table string) (out []string, err error) {
	err = ForEachTableScopes(ctx, client, blockNum, contract, table, func(response *TableScopeResponse) error {
		out = append(out, response.Scope)
		return nil
	})

	return out, err
}

func ForEachTableScopes(ctx context.Context, client StateClient, blockNum uint64, contract, table string, onEach func(response *TableScopeResponse) error) error {
	stream, err := client.StreamTableScopes(ctx, &StreamTableScopesRequest{
		BlockNum: blockNum,
		Contract: contract,
		Table:    table,
	})
	if err != nil {
		return fmt.Errorf("new stream: %w", err)
	}

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("stream: %w", err)
		}

		err = onEach(response)
		if err != nil {
			return err
		}
	}
}

// StreamReference determines at which block and irreversible block the server
// was currently at when served the streaming request.
//
// **Important** The last irreversible block will always be set but the
//               up to block value can be `nil`, for example if irreversible
//               only was set.
type StreamReference struct {
	UpToBlock             bstream.BlockRef
	LastIrreversibleBlock bstream.BlockRef
}

func ExtractStreamReference(stream grpc.ClientStream) (*StreamReference, error) {
	header, err := stream.Header()
	if err != nil {
		return nil, err
	}

	upToBlockIDs := header.Get(MetdataUpToBlockID)
	upToBlockNums := header.Get(MetdataUpToBlockNum)
	lastIrrBlockIDs := header.Get(MetdataLastIrrBlockID)
	lastIrrBlockNums := header.Get(MetdataLastIrrBlockNum)

	// All empty, we assume not found
	if len(upToBlockIDs) <= 0 && len(upToBlockNums) <= 0 && len(lastIrrBlockIDs) <= 0 && len(lastIrrBlockNums) <= 0 {
		return nil, ErrStreamReferenceNotFound
	}

	if len(lastIrrBlockIDs) <= 0 || len(lastIrrBlockNums) <= 0 {
		return nil, errors.New("last irreversible block ref should be defined")
	}

	libNum, err := strconv.ParseUint(lastIrrBlockNums[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid last irrversible block num: %w", err)
	}

	ref := &StreamReference{
		LastIrreversibleBlock: bstream.NewBlockRef(lastIrrBlockIDs[0], libNum),
	}

	if len(upToBlockIDs) > 0 && len(upToBlockNums) > 0 {
		blockNum, err := strconv.ParseUint(upToBlockNums[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid up to block num: %w", err)
		}

		ref.UpToBlock = bstream.NewBlockRef(upToBlockIDs[0], blockNum)
	}

	return ref, nil
}
