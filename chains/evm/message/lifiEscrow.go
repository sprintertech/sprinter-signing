package message

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

	"github.com/sprintertech/lifi-solver/pkg/protocols/lifi"
	lifiValidation "github.com/sprintertech/lifi-solver/pkg/protocols/lifi/validation"
)

const (
	EXPIRY          = time.Hour
	FILL_DEADLINE   = time.Minute * 5
	MAX_CALL_LENGTH = 65535
)

type OrderFetcher interface {
	GetOrder(orderID string) (*lifi.LifiOrder, error)
}

type LifiEscrowMessageHandler struct {
	chainID             uint64
	validator           lifiValidation.LifiEscrowOrderValidator[lifi.LifiOrder]
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

// HandleMessage verifies the lifi order on-chain signs
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

	err = h.verifyOrder(order)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	orderValue, err := order.TotalInputsUSDValue(nil)
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

	chainID, _ := strconv.ParseUint(order.Order.Outputs[0].ChainID, 10, 64)
	unlockHash, err := unlockHash(
		calldata,
		data.BorrowAmount,
		common.BytesToAddress(order.Order.Outputs[0].Token[:]),
		new(big.Int).SetUint64(chainID),
		h.lifiAddresses[chainID],
		uint64(order.Order.FillDeadline),
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
		chainID, _ := new(big.Int).SetString(o.ChainID, 10)
		call, _ := hex.DecodeString(o.Call)
		context, _ := hex.DecodeString(o.Context)
		outputs[i] = output{
			Oracle:    common.HexToHash(o.Oracle),
			Settler:   common.HexToHash(o.Settler),
			ChainId:   chainID,
			Amount:    o.Amount.Int,
			Recipient: common.HexToHash(o.Recipient),
			Call:      call,
			Context:   context,
		}
	}

	return consts.LifiABI.Pack(
		"fillOrderOutputs",
		order.Order.FillDeadline,
		common.HexToHash(order.Meta.OnChainOrderID),
		outputs,
		common.HexToHash(h.mpcAddress.Hex()))
}

// verifyOrder verifies order based on these instructions https://docs.catalyst.exchange/solver/orderflow/#order-validation
func (h *LifiEscrowMessageHandler) verifyOrder(order *lifi.LifiOrder) error {
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
