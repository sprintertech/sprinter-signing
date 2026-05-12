package message

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
	"github.com/sprintertech/sprinter-signing/chains/evm/signature"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"

	"github.com/sprintertech/lifi-solver/pkg/pricing"
	"github.com/sprintertech/lifi-solver/pkg/protocols/lifi"
	"github.com/sprintertech/lifi-solver/pkg/router"
	"github.com/sprintertech/lifi-solver/pkg/token"
)

type OrderFetcher interface {
	Order(ctx context.Context, hash common.Hash, orderID common.Hash) (*lifi.LifiOrder, error)
}

type OrderValidator interface {
	Validate(order *lifi.AugmentedLifiOrder) error
}

type LifiEscrowMessageHandler struct {
	chainID             uint64
	validator           OrderValidator
	orderPricer         pricing.OrderPricer
	router              router.OrderRouter
	confirmationWatcher ConfirmationWatcher

	lifiAddresses map[uint64]common.Address
	tokenResolver token.TokenResolver
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
	tokenResolver token.TokenResolver,
	orderFetcher OrderFetcher,
	orderPricer pricing.OrderPricer,
	router router.OrderRouter,
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
		tokenResolver:       tokenResolver,
		orderFetcher:        orderFetcher,
		orderPricer:         orderPricer,
		validator:           validator,
		sigChn:              sigChn,
		router:              router,
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

	log.Info().Str("depositId", data.OrderID).Msgf("Handling lifi escrow message %+v", data)

	order, err := h.orderFetcher.Order(
		context.Background(),
		common.HexToHash(data.DepositTxHash),
		common.HexToHash(data.OrderID))
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.verifyOrder(order)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	orderValue, err := order.TotalInputsUSDValue(h.orderPricer)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	borrowToken, destChainID, err := h.borrowToken(
		data,
		order,
		orderValue,
	)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.confirmationWatcher.WaitForOrderConfirmations(
		context.Background(),
		h.chainID,
		*order.Meta.OrderInitiatedTxHash,
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

	log.Debug().Msgf(`
		Singing lifi unlock hash.
		Calldata: %s
		Amount: %s
		Borrow token: %s,
		Target: %s
		Nonce: %s
		Filldeadline: %d
	`,
		hex.EncodeToString(calldata),
		data.BorrowAmount,
		borrowToken.Hex(),
		h.lifiAddresses[destChainID].Hex(),
		data.Nonce,
		data.Deadline,
	)

	unlockHash, err := signature.BorrowUnlockHash(
		calldata,
		data.BorrowAmount,
		borrowToken,
		new(big.Int).SetUint64(destChainID),
		h.lifiAddresses[destChainID],
		data.Deadline,
		data.Caller,
		data.LiquidityPool,
		data.Nonce,
	)
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

	err = h.coordinator.Execute(context.Background(), []tss.TssProcess{signing}, h.sigChn, data.Coordinator)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *LifiEscrowMessageHandler) borrowToken(
	data *LifiEscrowData,
	order *lifi.LifiOrder,
	amountInValue float64,
) (common.Address, uint64, error) {
	destChainID := order.Order.Outputs[0].ChainID

	tokenIn, err := h.tokenResolver.Token(
		order.GenericInputs[0].ChainID,
		*order.GenericInputs[0].TokenAddress)
	if err != nil {
		return common.Address{}, destChainID, err
	}

	tokenOut, err := h.tokenResolver.Token(
		order.GenericOutputs[0].ChainID,
		*order.GenericOutputs[0].TokenAddress)
	if err != nil {
		return common.Address{}, destChainID, err
	}

	if data.BorrowToken != tokenIn.Symbol && data.BorrowToken != tokenOut.Symbol {
		return common.Address{}, destChainID, fmt.Errorf(
			"borrow token %s must be either input %s or output token symbol %s",
			data.BorrowToken,
			tokenIn.Symbol,
			tokenOut.Symbol)
	}

	if data.BorrowToken == tokenIn.Symbol {
		if order.GenericInputs[0].Amount.Cmp(data.BorrowAmount) == -1 {
			return common.Address{}, destChainID, fmt.Errorf(
				"order input is less than requested borrow amount")
		}
		return common.BytesToAddress(tokenIn.Address[:]), destChainID, nil
	} else {
		amountOutValue := tokenOut.AmountToUSD(data.BorrowAmount)
		if amountInValue < amountOutValue {
			return common.Address{}, destChainID, fmt.Errorf(
				"order with destination borrow token has lower input amount value %f:%f",
				amountInValue,
				amountOutValue,
			)
		}

		return common.BytesToAddress(tokenOut.Address[:]), destChainID, nil
	}
}

func (h *LifiEscrowMessageHandler) calldata(order *lifi.LifiOrder) ([]byte, error) {
	type output struct {
		Oracle       common.Hash
		Settler      common.Hash
		Recipient    common.Hash
		ChainId      *big.Int
		Token        common.Hash
		Amount       *big.Int
		CallbackData []byte
		Context      []byte
	}
	outputs := make([]output, len(order.Order.Outputs))
	for i, o := range order.Order.Outputs {
		chainID := new(big.Int).SetUint64(o.ChainID)
		callbackData, err := hexutil.Decode(o.CallbackData)
		if err != nil {
			return nil, err
		}
		context, err := hexutil.Decode(o.Context)
		if err != nil {
			return nil, err
		}
		outputs[i] = output{
			Oracle:       *o.Oracle,
			Settler:      *o.Settler,
			ChainId:      chainID,
			Amount:       o.Amount.Int,
			Recipient:    *o.Recipient,
			CallbackData: callbackData,
			Context:      context,
			Token:        *o.Token,
		}
	}

	return consts.LifiABI.Pack(
		"fillOrderOutputs",
		order.Meta.OnChainOrderID,
		outputs,
		big.NewInt(order.Order.FillDeadline.Unix()),
		common.HexToHash(h.mpcAddress.Hex()).Bytes())
}

// verifyOrder verifies order based on these instructions https://docs.catalyst.exchange/solver/orderflow/#order-validation
func (h *LifiEscrowMessageHandler) verifyOrder(order *lifi.LifiOrder) error {
	if len(order.Order.Inputs) > 1 || len(order.Order.Inputs) == 0 {
		return fmt.Errorf("orders with multiple inputs not supported")
	}

	if len(order.Order.Outputs) > 1 {
		return fmt.Errorf("orders with multiple outputs not supported")
	}

	augmentedOrder, err := order.AugmentedOrder(h.orderPricer, h.router)
	if err != nil {
		return err
	}
	return h.validator.Validate(augmentedOrder)
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
				go func(wMsg *comm.WrappedMessage) {
					d := &LifiEscrowData{}
					err := json.Unmarshal(wMsg.Payload, d)
					if err != nil {
						log.Warn().Msgf("Failed unmarshaling LiFi message: %s", err)
						return
					}

					d.ErrChn = make(chan error, 1)
					msg := NewLifiEscrowMessage(d.Source, d.Destination, d)
					_, err = h.HandleMessage(msg)
					if err != nil {
						log.Err(err).Msgf("Failed handling LiFi message %+v because of: %s", msg, err)
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
