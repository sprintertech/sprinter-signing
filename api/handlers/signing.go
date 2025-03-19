package handlers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	across "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type ProtocolType string

const (
	AcrossProtocol ProtocolType = "across"
)

type SigningBody struct {
	ChainId       uint64
	DepositId     *BigInt      `json:"depositId"`
	Nonce         *BigInt      `json:"nonce"`
	Protocol      ProtocolType `json:"protocol"`
	LiquidityPool string       `json:"liquidityPool"`
	Caller        string       `json:"caller"`
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
				DepositId:     b.DepositId.Int,
				Nonce:         b.Nonce.Int,
				LiquidityPool: common.HexToAddress(b.LiquidityPool),
				Caller:        common.HexToAddress(b.Caller),
				ErrChn:        errChn,
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

	if b.LiquidityPool == "" {
		return fmt.Errorf("missing field 'liquidityPool'")
	}

	if b.Caller == "" {
		return fmt.Errorf("missing field 'caller'")
	}

	if b.Nonce == nil {
		return fmt.Errorf("missing field 'nonce'")
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

type SignatureCacher interface {
	Subscribe(ctx context.Context, id string, sigChannel chan []byte)
}

type StatusHandler struct {
	cache  SignatureCacher
	chains map[uint64]struct{}
}

func NewStatusHandler(cache SignatureCacher, chains map[uint64]struct{}) *StatusHandler {
	return &StatusHandler{
		cache:  cache,
		chains: chains,
	}
}

// HandleRequest is an sse handler that waits until the signing signature is ready
// and returns it
func (h *StatusHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainId, ok := new(big.Int).SetString(vars["chainId"], 0)
	if !ok {
		JSONError(w, "chain id invalid", http.StatusBadRequest)
		return
	}
	_, ok = h.chains[chainId.Uint64()]
	if !ok {
		JSONError(w, fmt.Sprintf("chain %d not supported", chainId.Int64()), http.StatusNotFound)
		return
	}
	depositId, ok := vars["depositId"]
	if !ok {
		JSONError(w, "missing 'depositId", http.StatusBadRequest)
		return
	}

	h.setheaders(w)

	ctx := r.Context()
	sigChn := make(chan []byte, 1)
	h.cache.Subscribe(ctx, fmt.Sprintf("%d-%s", chainId, depositId), sigChn)
	for {
		select {
		case <-r.Context().Done():
			return
		case sig := <-sigChn:
			{
				fmt.Fprintf(w, "data: %s\n\n", hex.EncodeToString(sig))
				w.(http.Flusher).Flush()
				return
			}
		}
	}
}

func (h *StatusHandler) setheaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}
