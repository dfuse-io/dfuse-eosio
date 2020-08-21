package server

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (srv *Server) GetActionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pathVariables := mux.Vars(r)
	account, ok := pathVariables["account"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	actions, err := srv.service.GetActions(ctx, account)
	if err != nil {
		zlog.Error("could not get actions", zap.Error(err), zap.String("account", account))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(actions)
	if err != nil {
		zlog.Error("could not encode json response", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
