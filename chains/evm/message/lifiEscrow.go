package message

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"

	"github.com/sprintertech/lifi-solver/pkg/pricing"
	"github.com/sprintertech/lifi-solver/pkg/protocols/lifi"
)

type OrderFetcher interface {
	GetOrder(orderID string) (*lifi.LifiOrder, error)
}

type OrderValidator interface {
	Validate(order *lifi.LifiOrder) error
}

type LifiEscrowMessageHandler struct {
	chainID             uint64
	validator           OrderValidator
	orderPricer         pricing.OrderPricer
	confirmationWatcher ConfirmationWatcher

	lifiAddresses map[uint64]common.Address
	tokenStore    config.TokenStore
	mpcAddress    common.Address

	orderFetcher OrderFetcher

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
	sigChn      chan any
}

func NewLifiEscrowMessageHandler(
	chainID uint64,
	mpcAddress common.Address,
	lifiAddresses map[uint64]common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	confirmationWatcher ConfirmationWatcher,
	tokenStore config.TokenStore,
	orderFetcher OrderFetcher,
	orderPricer pricing.OrderPricer,
	validator OrderValidator,
	sigChn chan any,
) *LifiEscrowMessageHandler {
	return &LifiEscrowMessageHandler{
		chainID:             chainID,
		lifiAddresses:       lifiAddresses,
		coordinator:         coordinator,
		host:                host,
		mpcAddress:          mpcAddress,
		comm:                comm,
		fetcher:             fetcher,
		confirmationWatcher: confirmationWatcher,
		tokenStore:          tokenStore,
		orderFetcher:        orderFetcher,
		orderPricer:         orderPricer,
		validator:           validator,
		sigChn:              sigChn,
	}
}

// HandleMessage verifies the lifi escrow order on-chain and signs
// the order if it is valid
func (h *LifiEscrowMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*LifiEscrowData)
	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	order, err := h.orderFetcher.GetOrder(data.OrderID)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.verifyOrder(order, data.BorrowAmount)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	orderValue, err := order.TotalInputsUSDValue(h.orderPricer)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	borrowToken, destChainID, err := h.borrowToken(order)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.confirmationWatcher.WaitForOrderConfirmations(
		context.Background(),
		h.chainID,
		common.HexToHash(data.DepositTxHash),
		orderValue,
	)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	data.ErrChn <- nil

	calldata, err := h.calldata(order)
	if err != nil {
		return nil, err
	}

	unlockHash, err := borrowManyUnlockHash(
		calldata,
		[]*big.Int{data.BorrowAmount},
		[]common.Address{borrowToken},
		new(big.Int).SetUint64(destChainID),
		h.lifiAddresses[destChainID],
		big.NewInt(order.Order.FillDeadline.Unix()).Uint64(),
		data.Caller,
		data.LiquidityPool,
		data.Nonce,
	)
	if err != nil {
		return nil, err
	}

	sessionID := fmt.Sprintf("%d-%s", h.chainID, order.Order.Nonce)
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

func (h *LifiEscrowMessageHandler) borrowToken(order *lifi.LifiOrder) (common.Address, uint64, error) {
	destChainID := order.Order.Outputs[0].ChainID
	tokenIn := common.BytesToAddress(order.GenericInputs[0].TokenAddress[:])
	symbol, _, err := h.tokenStore.ConfigByAddress(h.chainID, tokenIn)
	if err != nil {
		return common.Address{}, destChainID, err
	}

	destinationBorrowToken, err := h.tokenStore.ConfigBySymbol(destChainID, symbol)
	if err != nil {
		return common.Address{}, destChainID, err
	}

	return destinationBorrowToken.Address, destChainID, err
}

func (h *LifiEscrowMessageHandler) calldata(order *lifi.LifiOrder) ([]byte, error) {
	type output struct {
		Oracle    common.Hash
		Settler   common.Hash
		Recipient common.Hash
		ChainId   *big.Int
		Token     common.Hash
		Amount    *big.Int
		Call      []byte
		Context   []byte
	}
	outputs := make([]output, len(order.Order.Outputs))
	for i, o := range order.Order.Outputs {
		chainID := new(big.Int).SetUint64(o.ChainID)
		call, err := hexutil.Decode(o.Call)
		if err != nil {
			return nil, err
		}
		context, err := hexutil.Decode(o.Context)
		if err != nil {
			return nil, err
		}
		outputs[i] = output{
			Oracle:    *o.Oracle,
			Settler:   *o.Settler,
			ChainId:   chainID,
			Amount:    o.Amount.Int,
			Recipient: *o.Recipient,
			Call:      call,
			Context:   context,
		}
	}

	return consts.LifiABI.Pack(
		"fillOrderOutputs",
		common.HexToHash(order.Meta.OnChainOrderID),
		outputs,
		big.NewInt(order.Order.FillDeadline.Unix()),
		common.HexToHash(h.mpcAddress.Hex()).Bytes())
}

// verifyOrder verifies order based on these instructions https://docs.catalyst.exchange/solver/orderflow/#order-validation
func (h *LifiEscrowMessageHandler) verifyOrder(order *lifi.LifiOrder, borrowAmount *big.Int) error {
	if len(order.Order.Inputs) > 1 || len(order.Order.Inputs) == 0 {
		return fmt.Errorf("orders with multiple inputs not supported")
	}

	if len(order.Order.Outputs) > 1 {
		return fmt.Errorf("orders with multiple outputs not supported")
	}

	if order.GenericInputs[0].Amount.Cmp(borrowAmount) == -1 {
		return fmt.Errorf("order input is less than requested borrow amount")
	}

	return h.validator.Validate(order)
}

func (h *LifiEscrowMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(
		fmt.Sprintf("%d-%s", h.chainID, comm.LifiEscrowSessionID),
		comm.LifiEscrowMsg,
		msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				d := &LifiEscrowData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling Mayan message: %s", err)
					continue
				}

				d.ErrChn = make(chan error, 1)
				msg := NewLifiEscrowData(d.Source, d.Destination, d)
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

func (h *LifiEscrowMessageHandler) notify(data *LifiEscrowData) error {
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
		comm.LifiEscrowMsg,
		fmt.Sprintf("%d-%s", h.chainID, comm.LifiEscrowSessionID))
}
