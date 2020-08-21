package server

import (
	"context"
	"net/http"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	"github.com/gorilla/mux"
)

type WalletServer struct {
	addr string

	httpServer *http.Server
	mux        *mux.Router

	walletStore *wallet.Store
}

func New(addr string, walletStore *wallet.Store) *WalletServer {
	srv := &WalletServer{
		addr: addr,
		mux:  mux.NewRouter(),

		walletStore: walletStore,
	}

	srv.mux.Methods("GET").Path("/v0/wallet/{account}/transactions").HandlerFunc(srv.GetTransactionsHandler)

	return srv
}

func (srv *WalletServer) Serve() error {
	srv.httpServer = &http.Server{
		Addr:    srv.addr,
		Handler: srv.mux,
	}

	return srv.httpServer.ListenAndServe()
}

func (srv *WalletServer) Stop(ctx context.Context) error {
	return srv.httpServer.Shutdown(ctx)
}
