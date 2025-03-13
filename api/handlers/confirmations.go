package handlers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/gorilla/mux"
)

type ConfirmationsHandler struct {
	confirmationsByChain map[uint64]map[uint64]uint64
}

func NewConfirmationsHandler(confirmationsByChain map[uint64]map[uint64]uint64) *ConfirmationsHandler {
	return &ConfirmationsHandler{
		confirmationsByChain: confirmationsByChain,
	}
}

// HandleRequest returns confirmations by value buckets for the requested chain
func (h *ConfirmationsHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainId, ok := new(big.Int).SetString(vars["chainId"], 10)
	if !ok {
		JSONError(w, fmt.Errorf("invalid chainId"), http.StatusBadRequest)
		return
	}

	confirmations, ok := h.confirmationsByChain[chainId.Uint64()]
	if !ok {
		JSONError(w, fmt.Errorf("no confirmations for chainID: %d", chainId.Uint64()), http.StatusNotFound)
		return
	}

	data, _ := json.Marshal(confirmations)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
