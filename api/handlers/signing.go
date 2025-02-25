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

func NewSigningHandler(msgChan chan []*message.Message) *SigningHandler {
	return &SigningHandler{
		msgChan: msgChan,
	}
}

// HandleSigning sends a message to the across message handler and returns status code 202
// if the deposit has been accepted for the signing process
func (h *SigningHandler) HandleSigning(w http.ResponseWriter, r *http.Request) {
	i := &SigningInput{}
	d := json.NewDecoder(r.Body)
	err := d.Decode(i)
	if err != nil {
		JSONError(w, fmt.Sprintf("Failed reading request body: %s", err), http.StatusBadRequest)
		return
	}

	errChn := make(chan error, 1)
	am := across.NewAcrossMessage(0, i.ChainId, across.AcrossData{
		DepositId: i.DepositId,
		ErrChn:    errChn,
	})
	h.msgChan <- []*message.Message{am}

	err = <-errChn
	if err != nil {
		JSONError(w, fmt.Sprintf("Singing failed: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
