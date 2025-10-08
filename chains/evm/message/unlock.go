package message

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type LifiUnlockHandler struct {
	chainID uint64

	repayers map[uint64]common.Address

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
}

func NewLifiUnlockHandler(
	chainID uint64,
	repayers map[uint64]common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
) *LifiUnlockHandler {
	return &LifiUnlockHandler{
		chainID:     chainID,
		repayers:    repayers,
		coordinator: coordinator,
		host:        host,
		comm:        comm,
		fetcher:     fetcher,
	}
}

// HandleMessage signs the unlock request to the address of the repayer.
func (h *LifiUnlockHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*LifiUnlockData)
	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	unlockHash, err := h.lifiUnlockHash(data)
	if err != nil {
		return nil, err
	}

	sessionID := fmt.Sprintf("%d-%s", h.chainID, data.OrderID)
	signing, err := signing.NewSigning(
		new(big.Int).SetBytes(unlockHash),
		sessionID,
		sessionID,
		h.host,
		h.comm,
		h.fetcher)
	if err != nil {
		return nil, err
	}

	err = h.coordinator.Execute(context.Background(), []tss.TssProcess{signing}, data.SigChn, data.Coordinator)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *LifiUnlockHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(fmt.Sprintf("%d-%s", h.chainID, comm.LifiUnlockSessionID), comm.LifiUnlockMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				d := &LifiUnlockData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling across message: %s", err)
					continue
				}
				d.SigChn = make(chan interface{}, 1)

				msg := NewLifiUnlockMessage(d.Source, d.Destination, d)
				_, err = h.HandleMessage(msg)
				if err != nil {
					log.Err(err).Msgf("Failed handling across message %+v because of: %s", msg, err)
				}
			}
		case <-ctx.Done():
			{
				h.comm.UnSubscribe(subID)
				return
			}
		}
	}
}

func (h *LifiUnlockHandler) notify(data *LifiUnlockData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(
		h.host.Peerstore().Peers(),
		msgBytes, comm.LifiUnlockMsg,
		fmt.Sprintf("%d-%s", h.chainID, comm.LifiUnlockMsg))
}

func (h *LifiUnlockHandler) lifiUnlockHash(data *LifiUnlockData) ([]byte, error) {
	repaymentAddress, ok := h.repayers[h.chainID]
	if !ok {
		return nil, fmt.Errorf("invalid repayment chain %d", h.chainID)
	}

	msg := apitypes.TypedDataMessage{
		"orderId":     common.HexToHash(data.OrderID),
		"destination": common.HexToHash(repaymentAddress.Hex()),
		"call":        "0x",
	}
	chainId := math.HexOrDecimal256(*new(big.Int).SetUint64(h.chainID))
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"AllowOpen": []apitypes.Type{
				{Name: "orderId", Type: "bytes32"},
				{Name: "destination", Type: "bytes32"},
				{Name: "call", Type: "bytes"},
			},
		},
		PrimaryType: "AllowOpen",
		Domain: apitypes.TypedDataDomain{
			Name:              DOMAIN_NAME,
			ChainId:           &chainId,
			Version:           VERSION,
			VerifyingContract: data.Settler.Hex(),
		},
		Message: msg,
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return []byte{}, err
	}

	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return []byte{}, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(messageHash)))
	return crypto.Keccak256(rawData), nil
}
