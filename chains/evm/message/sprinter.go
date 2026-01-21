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

type SprinterRemoteCollateralMessageHandler struct {
	chainID uint64

	liquidator common.Address
	token      common.Address

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
	sigChn      chan any
}

func NewSprinterRemoteCollateralMessageHandler(
	chainID uint64,
	liquidator common.Address,
	token common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	sigChn chan any,
) *SprinterRemoteCollateralMessageHandler {
	return &SprinterRemoteCollateralMessageHandler{
		chainID:     chainID,
		coordinator: coordinator,
		token:       token,
		host:        host,
		comm:        comm,
		fetcher:     fetcher,
		sigChn:      sigChn,
	}
}

// HandleMessage signs the liquidation request if the transaction
// is going to the Liquidator contract.
func (h *SprinterRemoteCollateralMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*SprinterRemoteCollateralData)

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

	unlockHash, err := signature.BorrowUnlockHash(
		calldata,
		data.BorrowAmount,
		h.token,
		new(big.Int).SetUint64(h.chainID),
		h.liquidator,
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

	sessionID := fmt.Sprintf("%d-%s", h.chainID, data.Nonce)
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

func (h *SprinterRemoteCollateralMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(
		fmt.Sprintf("%d-%s-%s", h.chainID, h.token.Hex(), comm.SprinterRemoteCollateralSessionID),
		comm.SprinterRemoteCollateralMsg,
		msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				d := &SprinterRemoteCollateralData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling across message: %s", err)
					continue
				}

				d.ErrChn = make(chan error, 1)
				msg := NewSprinterRemoteCollateralMessage(d.Source, d.Destination, d)
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

func (h *SprinterRemoteCollateralMessageHandler) notify(data *SprinterRemoteCollateralData) error {
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
		comm.SprinterRemoteCollateralMsg,
		fmt.Sprintf("%d-%s-%s", h.chainID, h.token.Hex(), comm.SprinterRemoteCollateralSessionID))
}
