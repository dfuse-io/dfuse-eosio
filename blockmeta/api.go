package blockmeta

import (
	"context"
	"fmt"
	"time"

	pbheadinfo "github.com/dfuse-io/pbgo/dfuse/headinfo/v1"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/streamingfast/blockmeta"
	"go.uber.org/zap"
)

var APIs []*eos.API
var ExtraAPIs []*eos.API

func init() {
	blockmeta.BlockNumToIDFromAPI = blockNumToIDFromAPI
	blockmeta.GetHeadInfoFromAPI = headInfoFromAPI
	blockmeta.GetIrrIDFromAPI = IrrIDFromAPI
}

func IrrIDFromAPI(ctx context.Context, _ uint64, libNum uint64) (string, error) {
	return blockNumToIDFromAPI(ctx, libNum)
}

func blockNumToIDFromAPI(ctx context.Context, blockNum uint64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	apis := append(APIs, ExtraAPIs...)
	if len(apis) == 0 {
		return "", nil
	}

	if blockNum < 2 {
		return "", fmt.Errorf("trying to get block ID below block 2 on EOS")
	}
	respChan := make(chan string)
	errChan := make(chan error)
	for _, a := range APIs {
		api := a
		go func() {
			blk, err := api.GetBlockByNum(ctx, uint32(blockNum))
			if err != nil || blk == nil {
				select {
				case errChan <- err:
				case <-ctx.Done():
				}
				return
			}

			id, err := blk.BlockID()
			if err != nil {
				select {
				case errChan <- err:
				case <-ctx.Done():
				}
				return

			}

			select {
			case respChan <- id.String():
			case <-ctx.Done():
			}

		}()
	}
	var errors []error
	for {
		if len(errors) == len(apis) {
			return "", fmt.Errorf("all EOS APIs failed with errors: %v", errors)
		}
		select {
		case resp := <-respChan:
			return resp, nil
		case err := <-errChan:
			errors = append(errors, err)
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func headInfoFromAPI(ctx context.Context) (*pbheadinfo.HeadInfoResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	respChan := make(chan *pbheadinfo.HeadInfoResponse)
	errChan := make(chan error)
	for _, a := range APIs {
		api := a
		go func() {
			info, err := api.GetInfo(ctx)
			if err != nil {
				select {
				case errChan <- err:
				case <-ctx.Done():
				}
				return
			}
			headTimestamp, err := ptypes.TimestampProto(info.HeadBlockTime.Time)
			if err != nil {
				zlog.Error("invalid timestamp conversion from head block", zap.Error(err))
				select {
				case errChan <- err:
				case <-ctx.Done():
				}
				return

			}

			resp := &pbheadinfo.HeadInfoResponse{
				LibNum:   uint64(info.LastIrreversibleBlockNum),
				LibID:    info.LastIrreversibleBlockID.String(),
				HeadNum:  uint64(info.HeadBlockNum),
				HeadID:   info.HeadBlockID.String(),
				HeadTime: headTimestamp,
			}

			select {
			case respChan <- resp:
			case <-ctx.Done():
			}

		}()
	}
	var errors []error
	for {
		if len(errors) == len(APIs) {
			return nil, fmt.Errorf("all APIs failed with errors: %v", errors)
		}
		select {
		case resp := <-respChan:
			return resp, nil
		case err := <-errChan:
			errors = append(errors, err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
