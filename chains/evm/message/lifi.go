package message

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/protocol/lifi"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

const (
	EXPIRY          = time.Hour
	FILL_DEADLINE   = time.Minute * 5
	MAX_CALL_LENGTH = 65535
)

type OrderFetcher interface {
	GetOrder(orderID string) (*lifi.LifiOrder, error)
}

type AllocatorFetcher interface {
	Allocator(allocatorID *big.Int) (common.Address, error)
}

type LifiCompactMessageHandler struct {
	chainID uint64

	lifiAddresses  map[uint64]common.Address
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

	err = h.verifyOrder(order, data)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	data.ErrChn <- nil

	chainID, _ := strconv.ParseUint(order.Order.Outputs[0].ChainID, 10, 64)
	unlockHash, err := unlockHash(
		[]byte{}, // TODO
		data.BorrowAmount,
		common.HexToAddress(order.Order.Outputs[0].Token),
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

// verifyOrder verifies order based on: https://docs.catalyst.exchange/solver/orderflow/#order-validation
func (h *LifiCompactMessageHandler) verifyOrder(order *lifi.LifiOrder, data *LifiData) error {
	err := h.verifySignatures(order)
	if err != nil {
		return err
	}

	err = h.verifyDeadline(order)
	if err != nil {
		return err
	}

	err = h.verifyInputs(order)
	if err != nil {
		return err
	}

	err = h.verifyOutput(order, data.Caller, data.BorrowAmount)
	if err != nil {
		return err
	}

	return nil
}

func (h *LifiCompactMessageHandler) verifyInputs(order *lifi.LifiOrder) error {
	for _, token := range order.Order.Inputs {
		address := lifi.ExtractTokenAddress(token[0].Int)
		_, _, err := h.tokenStore.ConfigByAddress(h.chainID, address)
		if err != nil {
			return fmt.Errorf("token %s not configured", address)
		}
	}

	return nil
}

func (h *LifiCompactMessageHandler) verifyOutput(
	order *lifi.LifiOrder,
	caller common.Address,
	borrowAmount *big.Int,
) error {
	token := order.Order.Outputs[0].Token
	destination := order.Order.Outputs[0].ChainID
	amount := new(big.Int)
	for _, output := range order.Order.Outputs {
		chainID, err := strconv.ParseUint(output.ChainID, 10, 64)
		if err != nil {
			return err
		}

		if destination != output.ChainID {
			return fmt.Errorf("order has different destinations")
		}

		if output.Token != token {
			return fmt.Errorf("order has different output tokens")
		}

		_, _, err = h.tokenStore.ConfigByAddress(chainID, common.HexToAddress(output.Token))
		if err != nil {
			return fmt.Errorf("token %s not configured", token)
		}

		if len(output.Call) > MAX_CALL_LENGTH || len(output.Context) > MAX_CALL_LENGTH {
			return fmt.Errorf("output call exceeds max length")
		}

		if common.HexToAddress(output.Settler).Hex() != caller.Hex() {
			return fmt.Errorf("output settler %s is not caller %s", output.Settler, caller)
		}

		amount = new(big.Int).Add(amount, output.Amount.Int)
	}

	if amount.Cmp(borrowAmount) != 1 {
		return fmt.Errorf("requested borrow amount exceeds output amount")
	}

	return nil
}

func (h *LifiCompactMessageHandler) verifyDeadline(order *lifi.LifiOrder) error {
	fillDeadline := time.Unix(order.Order.FillDeadline, 0)
	if time.Until(fillDeadline) < FILL_DEADLINE {
		return fmt.Errorf("fill deadline %s too short", time.Until(fillDeadline))
	}

	expiryDeadline := time.Unix(order.Order.Expires, 0)
	if time.Until(expiryDeadline) < FILL_DEADLINE {
		return fmt.Errorf("order expiry %s too short", time.Until(expiryDeadline))
	}

	return nil
}

func (h *LifiCompactMessageHandler) verifySignatures(order *lifi.LifiOrder) error {
	digest, b, err := lifi.GenerateCompactDigest(
		new(big.Int).SetUint64(h.chainID),
		h.lifiAddresses[h.chainID],
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

		resetPeriod, err := lock.Period()
		if err != nil {
			return err
		}

		expiryDeadline := time.Unix(order.Order.Expires, 0)
		if time.Until(expiryDeadline) > resetPeriod {
			return fmt.Errorf("expiry less than reset period")
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
