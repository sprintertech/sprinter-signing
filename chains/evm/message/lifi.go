package message

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/protocol/lifi"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type OrderFetcher interface {
	GetOrder(orderID string) (*lifi.LifiOrder, error)
}

type AllocatorFetcher interface {
	Allocator(allocatorID *big.Int) (common.Address, error)
}

type LifiCompactMessageHandler struct {
	chainID uint64

	lifiAddress    common.Address
	liquidityPools map[uint64]common.Address
	tokenStore     config.TokenStore

	orderFetcher     OrderFetcher
	allocatorFetcher AllocatorFetcher

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
	sigChn      chan any
}

func NewLifiMessageHandler(
	chainID uint64,
	liquidityPools map[uint64]common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	tokenStore config.TokenStore,
	sigChn chan any,
) *LifiCompactMessageHandler {
	return &LifiCompactMessageHandler{
		chainID:        chainID,
		liquidityPools: liquidityPools,
		coordinator:    coordinator,
		host:           host,
		comm:           comm,
		fetcher:        fetcher,
		sigChn:         sigChn,
		tokenStore:     tokenStore,
	}
}

// HandleMessage verifies the lifi order signatures from the allocator and the sponsor and signs
// the order if it is valid
func (h *LifiCompactMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*LifiData)
	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	order, err := h.orderFetcher.GetOrder(data.OrderID)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.verifyOrder(order)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	data.ErrChn <- nil

	/*
			unlockHash, err := unlockHash(
				calldataBytes,
				data.BorrowAmount,
				destinationBorrowToken.Address,
				new(big.Int).SetUint64(destChainId),
				h.mayanPools[destChainId],
				msg.Deadline,
				data.Caller,
				data.LiquidityPool,
				data.Nonce,
			)
			if err != nil {
				return nil, err
			}

		sessionID := fmt.Sprintf("%d-%s", h.chainID, swap.OrderHash)
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
	*/
	return nil, nil
}

func (h *LifiCompactMessageHandler) verifyOrder(order *lifi.LifiOrder) error {
	digest, b, err := lifi.GenerateCompactDigest(
		new(big.Int).SetUint64(h.chainID),
		h.lifiAddress,
		*order,
	)
	if err != nil {
		return err
	}

	allocatorID, err := b.Commitments[0].AllocatorID()
	if err != nil {
		return err
	}
	for _, lock := range b.Commitments {
		id, err := lock.AllocatorID()
		if err != nil {
			return err
		}

		if allocatorID.String() != id.String() {
			return fmt.Errorf("order inputs have different allocators")
		}
	}

	valid, err := lifi.VerifyCompactSignature(
		digest,
		[]byte(order.SponsorSignature),
		common.HexToAddress(order.Order.User))
	if !valid || err != nil {
		return fmt.Errorf("sponsor signature invalid: %s", err)
	}

	allocator, err := h.allocatorFetcher.Allocator(allocatorID)
	if err != nil {
		return err
	}
	valid, err = lifi.VerifyCompactSignature(
		digest,
		[]byte(order.AllocatorSignature),
		allocator,
	)
	if !valid || err != nil {
		return fmt.Errorf("allocator signature invalid: %s", err)
	}

	return nil
}

func (h *LifiCompactMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(fmt.Sprintf("%d-%s", h.chainID, comm.LifiSessionID), comm.MayanMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				d := &LifiData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling Mayan message: %s", err)
					continue
				}

				d.ErrChn = make(chan error, 1)
				msg := NewLifiData(d.Source, d.Destination, d)
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

func (h *LifiCompactMessageHandler) notify(data *LifiData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.MayanMsg, fmt.Sprintf("%d-%s", h.chainID, comm.LifiSessionID))
}
