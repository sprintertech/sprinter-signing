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
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/contracts"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/protocol/mayan"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

var (
	BPS_DENOMINATOR = big.NewInt(10000)
)

type MayanContract interface {
	DecodeFulfillCall(calldata []byte) (*contracts.MayanFulfillParams, *contracts.MayanFulfillMsg, error)
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

	mayanPools          map[uint64]common.Address
	liqudityPools       map[uint64]common.Address
	confirmationWatcher ConfirmationWatcher
	tokenStore          config.TokenStore
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
	liqudityPools map[uint64]common.Address,
	mayanPools map[uint64]common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	confirmationWatcher ConfirmationWatcher,
	tokenStore config.TokenStore,
	mayanDecoder MayanContract,
	swapFetcher SwapFetcher,
	sigChn chan any,
) *MayanMessageHandler {
	return &MayanMessageHandler{
		chainID:             chainID,
		client:              client,
		mayanPools:          mayanPools,
		liqudityPools:       liqudityPools,
		coordinator:         coordinator,
		host:                host,
		comm:                comm,
		fetcher:             fetcher,
		sigChn:              sigChn,
		confirmationWatcher: confirmationWatcher,
		mayanDecoder:        mayanDecoder,
		swapFetcher:         swapFetcher,
		tokenStore:          tokenStore,
	}
}

// HandleMessage finds the Mayan deposit with the according deposit ID and starts
// the MPC signature process for it. The result will be saved into the signature
// cache through the result channel.
func (h *MayanMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*MayanData)
	txHash := common.HexToHash(data.DepositTxHash)

	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	calldataBytes, err := hex.DecodeString(data.Calldata)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	params, msg, err := h.mayanDecoder.DecodeFulfillCall(calldataBytes)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	swap, err := h.swapFetcher.GetSwap(txHash.Hex())
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	symbol, token, err := h.tokenStore.ConfigByAddress(h.chainID, common.BytesToAddress(msg.TokenIn[12:]))
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	destinationBorrowToken, err := h.tokenStore.ConfigBySymbol(uint64(msg.DestChainId), symbol)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	order, err := h.mayanDecoder.GetOrder(msg, swap, token.Decimals)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.verifyOrder(msg, params, order, swap, data)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.confirmationWatcher.WaitForConfirmations(
		context.Background(),
		h.chainID,
		txHash,
		common.BytesToAddress(msg.TokenIn[12:]),
		new(big.Int).SetUint64(msg.PromisedAmount))
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	destChainId, err := mayan.WormholeToEVMChainID(msg.DestChainId)
	if err != nil {
		return nil, err
	}

	data.ErrChn <- nil

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
	return nil, nil
}

func (h *MayanMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(fmt.Sprintf("%d-%s", h.chainID, comm.MayanSessionID), comm.MayanMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				d := &MayanData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling Mayan message: %s", err)
					continue
				}

				msg := NewMayanMessage(d.Source, d.Destination, d)
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

func (h *MayanMessageHandler) verifyOrder(
	msg *contracts.MayanFulfillMsg,
	params *contracts.MayanFulfillParams,
	order *contracts.MayanOrder,
	swap *mayan.MayanSwap,
	data *MayanData) error {
	srcChainId, err := mayan.WormholeToEVMChainID(msg.SrcChainId)
	if err != nil {
		return err
	}

	if srcChainId != h.chainID {
		return fmt.Errorf("msg and handler chainID not matching")
	}

	if swap.OrderHash != "0x"+hex.EncodeToString(msg.OrderHash[:]) {
		return fmt.Errorf("swap and msg hash not matching")
	}

	if common.BytesToAddress(msg.ReferrerAddr[12:]) != data.Caller {
		return fmt.Errorf("referrer and caller address is not the same")
	}

	if order.Status != contracts.OrderCreated {
		return fmt.Errorf("invalid order status %d", order.Status)
	}

	srcLiquidityPool, ok := h.liqudityPools[h.chainID]
	if !ok {
		return fmt.Errorf("no source liqudity recipient configured")
	}
	if common.BytesToAddress(params.Recipient[12:]) != srcLiquidityPool {
		return fmt.Errorf("invalid recipient")
	}

	_, tc, err := h.tokenStore.ConfigByAddress(h.chainID, common.BytesToAddress(msg.TokenIn[12:]))
	if err != nil {
		return err
	}

	promisedAmount := contracts.DenormalizeAmount(
		new(big.Int).SetUint64(msg.PromisedAmount),
		tc.Decimals)
	netAmount, err := calculateNetAmount(
		params.FulfillAmount,
		msg.ReferrerBps,
		msg.ProtocolBps)
	if err != nil {
		return err
	}
	if netAmount.Cmp(promisedAmount) == -1 {
		return fmt.Errorf(
			"net amount %s smaller than promised amount %s",
			netAmount,
			promisedAmount)
	}

	return nil
}

func (h *MayanMessageHandler) notify(data *MayanData) error {
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

func calculateNetAmount(
	fulfillAmount *big.Int,
	referrerBps uint8,
	protocolBps uint8,
) (*big.Int, error) {
	referrerAmount := new(big.Int).Div(
		new(big.Int).Mul(fulfillAmount, big.NewInt(int64(referrerBps))),
		BPS_DENOMINATOR,
	)

	protocolAmount := new(big.Int).Div(
		new(big.Int).Mul(fulfillAmount, big.NewInt(int64(protocolBps))),
		BPS_DENOMINATOR,
	)

	netAmount := new(big.Int).Sub(fulfillAmount, referrerAmount)
	netAmount.Sub(netAmount, protocolAmount)

	return netAmount, nil
}
