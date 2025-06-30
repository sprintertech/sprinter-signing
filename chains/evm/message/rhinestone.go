package message

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/protocol/rhinestone"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type BundleFetcher interface {
	GetBundle(bundleID string) (*rhinestone.Bundle, error)
}

type RhinestoneMessageHandler struct {
	chainID uint64

	liquidityPools map[uint64]common.Address
	bundleFetcher  BundleFetcher
	tokenStore     config.TokenStore

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan any
}

// HandleMessage
func (h *RhinestoneMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*RhinestoneData)
	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	bundle, err := h.bundleFetcher.GetBundle(data.BundleID)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	calldata, err := hex.DecodeString(bundle.BundleEvent.FillPayload.Data)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	deadline, err := strconv.ParseUint(bundle.BundleData.Expires, 10, 64)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	unlockHash, err := unlockHash(
		calldata,
		data.BorrowAmount,
		common.Address{},
		new(big.Int).SetUint64(bundle.TargetChainId),
		common.HexToAddress(bundle.BundleEvent.FillPayload.To),
		deadline,
		data.Caller,
		data.LiquidityPool,
		data.Nonce,
	)
	if err != nil {
		return nil, err
	}

	sessionID := fmt.Sprintf("%d-%s", h.chainID, bundle.BundleEvent.BundleId)
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

func (h *RhinestoneMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(fmt.Sprintf("%d-%s", h.chainID, comm.RhinestoneSessionID), comm.RhinestoneMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				d := &RhinestoneData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling Mayan message: %s", err)
					continue
				}

				d.ErrChn = make(chan error, 1)
				msg := NewRhinestoneMessage(d.Source, d.Destination, d)
				_, err = h.HandleMessage(msg)
				if err != nil {
					log.Err(err).Msgf("Failed handling Mayan message %+v because of: %s", msg, err)
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

func (h *RhinestoneMessageHandler) notify(data *RhinestoneData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.MayanMsg, fmt.Sprintf("%d-%s", h.chainID, comm.MayanSessionID))
}
