package handlers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	across "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type SigningInput struct {
	DepositId *big.Int `json:"depositId"`
	ChainId   uint64   `json:"chainId"`
}

type SigningHandler struct {
	msgChan chan []*message.Message
}

func NewSigningHandler() *SigningHandler {
	return &SigningHandler{}
}

func (h *SigningHandler) HandleSigning(w http.ResponseWriter, r *http.Request) {
	i := &SigningInput{}
	d := json.NewDecoder(r.Body)
	err := d.Decode(i)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed reading request body: %s", err), http.StatusBadRequest)
		return
	}

	am := across.NewAcrossMessage(0, i.ChainId, across.AcrossData{
		DepositId: i.DepositId,
	})
	h.msgChan <- []*message.Message{am}
}
