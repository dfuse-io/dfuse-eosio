package server

import (
	"context"
	"net/http"

	"github.com/dfuse-io/dfuse-eosio/accounthist"
	"github.com/gorilla/mux"
)

type Server struct {
	addr string

	httpServer *http.Server
	mux        *mux.Router

	service *accounthist.Service
}

func New(addr string, service *accounthist.Service) *Server {
	srv := &Server{
		addr: addr,
		mux:  mux.NewRouter(),

		service: service,
	}

	srv.mux.Methods("GET").Path("/v0/wallet/{account}/actions").HandlerFunc(srv.GetActionsHandler)

	return srv
}

func (srv *Server) Serve() error {
	srv.httpServer = &http.Server{
		Addr:    srv.addr,
		Handler: srv.mux,
	}

	return srv.httpServer.ListenAndServe()
}

func (srv *Server) Stop(ctx context.Context) error {
	return srv.httpServer.Shutdown(ctx)
}
