package message

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/contracts"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/protocol/mayan"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	tssMessage "github.com/sprintertech/sprinter-signing/tss/message"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type MayanContract interface {
	DecodeFulfillCall(calldata []byte) (*contracts.MayanFulfillMsg, error)
	GetOrder(
		msg *contracts.MayanFulfillMsg,
		swap *mayan.MayanSwap,
		srcTokenDecimals uint8,
	) (*contracts.MayanOrder, error)
}

type SwapFetcher interface {
	GetSwap(hash string) (*mayan.MayanSwap, error)
}

type MayanMessageHandler struct {
	client  EventFilterer
	chainID uint64

	pools               map[uint64]common.Address
	confirmationWatcher ConfirmationWatcher
	mayanDecoder        MayanContract
	swapFetcher         SwapFetcher

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan any
}

func NewMayanMessageHandler(
	chainID uint64,
	client EventFilterer,
	pools map[uint64]common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	confirmationWatcher ConfirmationWatcher,
	sigChn chan any,
) *MayanMessageHandler {
	return &MayanMessageHandler{
		chainID:             chainID,
		client:              client,
		pools:               pools,
		coordinator:         coordinator,
		host:                host,
		comm:                comm,
		fetcher:             fetcher,
		sigChn:              sigChn,
		confirmationWatcher: confirmationWatcher,
	}
}

func (h *MayanMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(fmt.Sprintf("%d-%s", h.chainID, comm.MayanSessionID), comm.MayanMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				mayanMsg, err := tssMessage.UnmarshalMayanMessage(wMsg.Payload)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling Mayan message: %s", err)
					continue
				}

				msg := NewMayanMessage(mayanMsg.Source, mayanMsg.Destination, &MayanData{
					Coordinator:   wMsg.From,
					LiquidityPool: common.HexToAddress(mayanMsg.LiquidityPool),
					Caller:        common.HexToAddress(mayanMsg.Caller),
					ErrChn:        make(chan error, 1),
				})
				_, err = h.HandleMessage(msg)
				if err != nil {
					log.Err(err).Msgf("Failed handling Mayan message %+v because of: %s", mayanMsg, err)
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

// HandleMessage finds the Mayan deposit with the according deposit ID and starts
// the MPC signature process for it. The result will be saved into the signature
// cache through the result channel.
func (h *MayanMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*MayanData)
	txHash := common.HexToHash(data.DepositTxHash)

	err := h.notify(m, data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	msg, err := h.mayanDecoder.DecodeFulfillCall(data.Calldata)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	swap, err := h.swapFetcher.GetSwap(txHash.Hex())
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	if swap.OrderHash != hex.EncodeToString(msg.OrderHash[:]) {
		err = fmt.Errorf("swap and msg hash not matching")
		data.ErrChn <- err
		return nil, err
	}

	_, token, err := h.confirmationWatcher.TokenConfig(common.HexToAddress(swap.FromTokenAddress))
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	// TODO: use calldata to verify it is correct
	order, err := h.mayanDecoder.GetOrder(msg, swap, token.Decimals)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	if order.Status != contracts.OrderCreated {
		err = fmt.Errorf("invalid order status %d", order.Status)
		data.ErrChn <- err
		return nil, err
	}

	err = h.confirmationWatcher.WaitForConfirmations(
		context.Background(),
		txHash,
		common.BytesToAddress(msg.TokenIn[12:]),
		new(big.Int).SetUint64(msg.PromisedAmount))
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	data.ErrChn <- nil

	destChainId := new(big.Int).SetUint64(uint64(msg.DestChainId))
	unlockHash, err := unlockHash(
		data.Calldata,
		new(big.Int).SetUint64(msg.PromisedAmount),
		common.BytesToAddress(msg.TokenOut[12:]),
		destChainId,
		h.pools[destChainId.Uint64()],
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
	return nil, nil
}

func (h *MayanMessageHandler) notify(m *message.Message, data *MayanData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := tssMessage.MarshalMayanMessage(
		data.Caller.Hex(),
		m.Source,
		m.Destination)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.MayanMsg, fmt.Sprintf("%d-%s", h.chainID, comm.MayanSessionID))
}
