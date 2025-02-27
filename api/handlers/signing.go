package handlers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/gorilla/mux"
	across "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type ProtocolType string

const (
	AcrossProtocol ProtocolType = "across"
)

type SigningBody struct {
	ChainId   uint64
	DepositId *big.Int     `json:"depositId"`
	Protocol  ProtocolType `json:"protocol"`
}

type SigningHandler struct {
	msgChan chan []*message.Message
	chains  map[uint64]struct{}
}

func NewSigningHandler(msgChan chan []*message.Message, chains map[uint64]struct{}) *SigningHandler {
	return &SigningHandler{
		msgChan: msgChan,
		chains:  chains,
	}
}

// HandleSigning sends a message to the across message handler and returns status code 202
// if the deposit has been accepted for the signing process
func (h *SigningHandler) HandleSigning(w http.ResponseWriter, r *http.Request) {
	b := &SigningBody{}
	d := json.NewDecoder(r.Body)
	err := d.Decode(b)
	if err != nil {
		JSONError(w, fmt.Sprintf("invalid request body: %s", err), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	err = h.validate(b, vars)
	if err != nil {
		JSONError(w, fmt.Sprintf("invalid request body: %s", err), http.StatusBadRequest)
		return
	}
	errChn := make(chan error, 1)

	var m *message.Message
	switch b.Protocol {
	case AcrossProtocol:
		{
			m = across.NewAcrossMessage(0, b.ChainId, across.AcrossData{
				DepositId: b.DepositId,
				ErrChn:    errChn,
			})
		}
	default:
		JSONError(w, fmt.Sprintf("invalid protocol %s", b.Protocol), http.StatusBadRequest)
		return
	}
	h.msgChan <- []*message.Message{m}

	err = <-errChn
	if err != nil {
		JSONError(w, fmt.Sprintf("Singing failed: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *SigningHandler) validate(b *SigningBody, vars map[string]string) error {
	chainId, ok := new(big.Int).SetString(vars["chainId"], 10)
	if !ok {
		return fmt.Errorf("field 'chainId' invalid")
	}
	b.ChainId = chainId.Uint64()

	if b.DepositId == nil {
		return fmt.Errorf("missing field 'depositId'")
	}

	if b.ChainId == 0 {
		return fmt.Errorf("missing field 'chainId'")
	}

	_, ok = h.chains[b.ChainId]
	if !ok {
		return fmt.Errorf("chain '%d' not supported", b.ChainId)
	}

	return nil
}
