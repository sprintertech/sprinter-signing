package handlers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	evmMessage "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

const SIGNATURE_TIMEOUT = time.Second * 15

type UnlockResponse struct {
	Signature string `json:"signature"`
	ID        string `json:"id"`
}

type UnlockBody struct {
	ChainId  uint64
	Protocol ProtocolType `json:"protocol"`
	OrderID  string       `json:"orderId"`
	Settler  string       `json:"settler"`
}

type UnlockHandler struct {
	chains  map[uint64]struct{}
	msgChan chan []*message.Message
}

func NewUnlockHandler(msgChn chan []*message.Message, chains map[uint64]struct{}) *UnlockHandler {
	return &UnlockHandler{
		chains:  chains,
		msgChan: msgChn,
	}
}

// HandleSigning sends a message to the across message handler and returns status code 202
// if the deposit has been accepted for the signing process
func (h *UnlockHandler) HandleUnlock(w http.ResponseWriter, r *http.Request) {
	b := &UnlockBody{}
	d := json.NewDecoder(r.Body)
	err := d.Decode(b)
	if err != nil {
		JSONError(w, fmt.Errorf("invalid request body: %s", err), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	err = h.validate(b, vars)
	if err != nil {
		JSONError(w, fmt.Errorf("invalid request body: %s", err), http.StatusBadRequest)
		return
	}

	sigChn := make(chan interface{}, 1)
	var m *message.Message
	switch b.Protocol {
	case LifiProtocol:
		{
			m = evmMessage.NewLifiUnlockMessage(0, b.ChainId, &evmMessage.LifiUnlockData{
				Source:      0,
				Destination: b.ChainId,
				SigChn:      sigChn,
				OrderID:     b.OrderID,
				Settler:     common.HexToAddress(b.Settler),
			})
		}
	default:
		JSONError(w, fmt.Errorf("invalid protocol %s", b.Protocol), http.StatusBadRequest)
		return
	}
	h.msgChan <- []*message.Message{m}

	for {
		select {
		case <-time.After(SIGNATURE_TIMEOUT):
			JSONError(w, fmt.Errorf("timeout"), http.StatusInternalServerError)
			return
		case sig := <-sigChn:
			{
				sig, ok := sig.(signing.EcdsaSignature)
				if !ok {
					JSONError(w, fmt.Errorf("invalid signature"), http.StatusInternalServerError)
					return
				}

				data, _ := json.Marshal(UnlockResponse{
					Signature: hex.EncodeToString(sig.Signature),
					ID:        sig.ID,
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
				return
			}
		}
	}
}

func (h *UnlockHandler) validate(b *UnlockBody, vars map[string]string) error {
	chainId, ok := new(big.Int).SetString(vars["chainId"], 10)
	if !ok {
		return fmt.Errorf("field 'chainId' invalid")
	}
	b.ChainId = chainId.Uint64()

	if b.ChainId == 0 {
		return fmt.Errorf("missing field 'chainId'")
	}

	_, ok = h.chains[b.ChainId]
	if !ok {
		return fmt.Errorf("chain '%d' not supported", b.ChainId)
	}

	return nil
}
