package message

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm/signature"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type SprinterCreditMessageHandler struct {
	chainID uint64

	liquidators map[common.Address]common.Address

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
	sigChn      chan any
}

func NewSprinterCreditMessageHandler(
	chainID uint64,
	liquidators map[common.Address]common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	sigChn chan any,
) *SprinterCreditMessageHandler {
	return &SprinterCreditMessageHandler{
		chainID:     chainID,
		coordinator: coordinator,
		liquidators: liquidators,
		host:        host,
		comm:        comm,
		fetcher:     fetcher,
		sigChn:      sigChn,
	}
}

// HandleMessage signs the liquidation request if the transaction
// is going to the Liquidator contract.
func (h *SprinterCreditMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*SprinterCreditData)

	log.Info().Msgf("Handling sprinter remote collateral message %+v", data)

	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	calldata, err := hex.DecodeString(data.Calldata)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	token := common.HexToAddress(data.TokenOut)
	liquidator, ok := h.liquidators[token]
	if !ok {
		err := fmt.Errorf("no liquidator for token %s", data.TokenOut)
		data.ErrChn <- err
		return nil, err
	}

	unlockHash, err := signature.BorrowUnlockHash(
		calldata,
		data.BorrowAmount,
		token,
		new(big.Int).SetUint64(h.chainID),
		liquidator,
		data.Deadline,
		data.Caller,
		data.LiquidityPool,
		data.Nonce,
	)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	data.ErrChn <- nil

	sessionID := fmt.Sprintf("%d-%s", h.chainID, data.DepositID)
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

	err = h.coordinator.Execute(context.Background(), []tss.TssProcess{signing}, h.sigChn, data.Coordinator)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *SprinterCreditMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(
		fmt.Sprintf("%d-%s", h.chainID, comm.SprinterCreditSessionID),
		comm.SprinterCreditMsg,
		msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				go func(wMsg *comm.WrappedMessage) {
					d := &SprinterCreditData{}
					err := json.Unmarshal(wMsg.Payload, d)
					if err != nil {
						log.Warn().Msgf("Failed unmarshaling across message: %s", err)
						return
					}

					d.ErrChn = make(chan error, 1)
					msg := NewSprinterCreditMessage(d.Source, d.Destination, d)
					_, err = h.HandleMessage(msg)
					if err != nil {
						log.Err(err).Msgf("Failed handling across message %+v because of: %s", msg, err)
					}
				}(wMsg)
			}
		case <-ctx.Done():
			{
				h.comm.UnSubscribe(subID)
				return
			}
		}
	}
}

func (h *SprinterCreditMessageHandler) notify(data *SprinterCreditData) error {
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
		msgBytes,
		comm.SprinterCreditMsg,
		fmt.Sprintf("%d-%s", h.chainID, comm.SprinterCreditSessionID))
}
