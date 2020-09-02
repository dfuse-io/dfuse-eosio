package grpc

import (
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetActions(req *pbaccounthist.GetActionsRequest, stream pbaccounthist.AccountHistory_GetActionsServer) error {
	if req.Limit < 0 {
		return status.Error(codes.InvalidArgument, "negative limit is not valid")
	}

	// TODO: triple check that `account` is an EOS Name (encode / decode and check for ==, otherwise, BadRequest), perhaps at the DGraphQL level plz
	account := req.Account
	limit := uint64(req.Limit)

	err := s.service.StreamActions(stream.Context(), account, limit, req.Cursor, func(cursor *pbaccounthist.Cursor, actionTrace *pbcodec.ActionTrace) error {
		if err := stream.Send(&pbaccounthist.ActionResponse{Cursor: cursor, ActionTrace: actionTrace}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return status.Errorf(codes.Unknown, "unable to stream actions: %s", err)
	}

	return nil
}
