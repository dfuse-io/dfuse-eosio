package server

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
)

func (srv *WalletServer) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pathVariables := mux.Vars(r)
	account, ok := pathVariables["account"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	transactions, err := srv.walletStore.GetTransactions(ctx, account)
	if err != nil {
		zlog.Error("could not get transactions", zap.Error(err), zap.String("account", account))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(transactions)
	if err != nil {
		zlog.Error("could not encode json response", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
