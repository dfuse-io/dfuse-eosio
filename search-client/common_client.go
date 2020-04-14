package searchclient

import (
	"context"
	"io"

	"github.com/dfuse-io/dhammer"
	"github.com/dfuse-io/logging"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"google.golang.org/grpc"
)

type commonClient struct {
	client pbsearch.RouterClient
}

func newCommonClient(cc *grpc.ClientConn) *commonClient {
	return &commonClient{
		client: pbsearch.NewRouterClient(cc),
	}
}

func (c *commonClient) streamSearchToHammer(ctx context.Context, hammer *dhammer.Hammer, req *pbsearch.RouterRequest) {
	zlogger := logging.Logger(ctx, zlog)
	searchCtx, cancelSearch := context.WithCancel(ctx)
	defer func() {
		zlogger.Debug("search stream loop completed")
		cancelSearch()
		hammer.Close()
	}()

	zlogger.Debug("search stream performing actual search request")
	stream, err := c.client.StreamMatches(searchCtx, req)
	if err != nil {
		hammer.In <- &matchOrError{err: err}
		return
	}

	zlogger.Debug("search stream starting receive loop")
	for {
		match, err := stream.Recv()

		// Everything goes through dhammer, be it a match or an error, that way, the flow is linear
		// and dhammer takes care of sending back the processed match or error.
		batchItem := &matchOrError{
			match: match,
			err:   err,
		}

		// When we reach EOF, we close this receive loop and let dhammer drain itselfs.
		// It's dhammer who will forward the final `io.EOF` to the parent consumer.
		if err == io.EOF {
			zlogger.Debug("search stream reached EOF")
			return
		}

		if traceEnabled {
			zlogger.Debug("sending search item to hammer")
		}

		select {
		case <-ctx.Done():
			zlogger.Debug("search stream caller context done")
			return
		case hammer.In <- batchItem:
			// Avoiding logging (and any other things) here greatly improves batching
		}

		// We check and return upon error here **after** sending the batch item to
		// dhammer, so the error is seen by dhammer! Then we terminate the loop.
		// Otherwise, if done before the `select`, we would close the loop without properly
		// sending the error through dhammer, creating a hole here.
		if err != nil {
			return
		}
	}
}

type onHammerItem func(interface{})
type onHammerError func(error)

func (c *commonClient) hammerToConsumer(ctx context.Context, hammer *dhammer.Hammer, onItem onHammerItem, onError onHammerError) {
	zlogger := logging.Logger(ctx, zlog)

	zlogger.Debug("starting dhammer loop")
	defer func() {
		zlogger.Debug("dhammer loop completed")
	}()

	for {
		select {
		case <-ctx.Done():
			zlogger.Debug("dhammer caller context done")
			return
		case v, ok := <-hammer.Out:
			if !ok {
				zlogger.Debug("dhammer channel closed")
				if hammer.Err() != nil && hammer.Err() != context.Canceled {
					zlogger.Debug("sending error from dhammer to consumer")
					onError(hammer.Err())
				} else {
					zlogger.Debug("dhammer completely drained normally, sending EOF to consumer")
					onError(io.EOF)
				}

				return
			}

			select {
			case <-ctx.Done():
				zlogger.Debug("dhammer caller context done")
				return
			default:
				if traceEnabled {
					zlogger.Debug("sending hammer processed item to consumer")
				}

				onItem(v)
			}
		}
	}
}

func gatherTransactionPrefixesToFetch(items []interface{}, needsFetch func(*pbsearch.SearchMatch) bool) (prefixes []string, prefixToIndex map[string]int) {
	prefixToIndex = map[string]int{}
	for _, item := range items {
		m := item.(*matchOrError)

		// The hammer batch contains an error at some point, we ignored anything after it
		if m.err != nil {
			break
		}

		if needsFetch(m.match) {
			prefixes = append(prefixes, m.match.TrxIdPrefix)
			prefixToIndex[m.match.TrxIdPrefix] = len(prefixes) - 1
		}
	}

	return
}
