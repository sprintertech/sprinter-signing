package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/big"
	"slices"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
	evmMessage "github.com/sprintertech/sprinter-signing/chains/evm/message"
	lighterChain "github.com/sprintertech/sprinter-signing/chains/lighter"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/protocol/lighter"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

var (
	ARBITRUM_CHAIN_ID          = big.NewInt(42161)
	USDC_ACCOUNT_INDEX uint64  = 3
	USDC_DECIMALS      float64 = 6
)

type TxFetcher interface {
	GetTx(hash string) (*lighter.LighterTx, error)
}

type LighterMessageHandler struct {
	host    host.Host
	comm    comm.Communication
	sigChn  chan any
	msgChan chan []*message.Message

	lighterAddress   common.Address
	usdcAddress      common.Address
	repaymentAccount string
	txFetcher        TxFetcher
	confirmations    map[uint64]uint64
}

func NewLighterMessageHandler(
	lighterAddress common.Address,
	usdcAddress common.Address,
	repaymentAccount string,
	confirmations map[uint64]uint64,
	txFetcher TxFetcher,
	host host.Host,
	comm comm.Communication,
	sigChn chan any,
	msgChan chan []*message.Message,
) *LighterMessageHandler {
	return &LighterMessageHandler{
		txFetcher:        txFetcher,
		usdcAddress:      usdcAddress,
		repaymentAccount: repaymentAccount,
		lighterAddress:   lighterAddress,
		host:             host,
		comm:             comm,
		sigChn:           sigChn,
		msgChan:          msgChan,
		confirmations:    confirmations,
	}
}

// HandleMessage finds the Lighter deposit with the according deposit ID and starts
// the MPC signature process for it. The result will be saved into the signature
// cache through the result channel.
func (h *LighterMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*LighterData)

	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	tx, err := h.txFetcher.GetTx(data.DepositTxHash)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	if err = h.verifyWithdrawal(tx); err != nil {
		data.ErrChn <- err
		return nil, err
	}

	data.ErrChn <- nil

	calldata, err := h.calldata(tx)
	if err != nil {
		return nil, err
	}

	sessionID := signing.SessionID(lighterChain.LIGHTER_DOMAIN_ID, data.OrderHash)
	h.msgChan <- []*message.Message{evmMessage.NewSignMessage(0, ARBITRUM_CHAIN_ID.Uint64(), &evmMessage.SignRequest{
		Calldata:      calldata,
		BorrowAmount:  new(big.Int).SetUint64(tx.Transfer.Amount),
		BorrowToken:   h.usdcAddress.Hex(),
		Target:        h.lighterAddress.Hex(),
		Deadline:      data.Deadline,
		Caller:        h.lighterAddress.Hex(),
		LiquidityPool: data.LiquidityPool,
		Nonce:         data.Nonce,
		SessionID:     sessionID,
		Coordinator:   data.Coordinator,
		ResultChn:     h.sigChn,
	})}
	return nil, nil
}

func (h *LighterMessageHandler) verifyWithdrawal(tx *lighter.LighterTx) error {
	if tx.Type != lighter.TxTypeL2Transfer {
		return errors.New("invalid transaction type")
	}

	if strconv.Itoa(tx.Transfer.ToAccountIndex) != h.repaymentAccount {
		return errors.New("transfer account index invalid")
	}

	if tx.Transfer.AssetIndex != USDC_ACCOUNT_INDEX {
		return errors.New("only usdc asset supported on lighter")
	}

	if err := h.verifyOrderSize(tx.Transfer.Amount / uint64(math.Pow(10, USDC_DECIMALS))); err != nil {
		return err
	}

	return nil
}

func (h *LighterMessageHandler) verifyOrderSize(orderValue uint64) error {
	buckets := slices.Collect(maps.Keys(h.confirmations))
	slices.Sort(buckets)
	for _, bucket := range buckets {
		if orderValue < bucket {
			return nil
		}
	}

	return fmt.Errorf("order value %d exceeds confirmation buckets", orderValue)
}

func (h *LighterMessageHandler) calldata(tx *lighter.LighterTx) ([]byte, error) {
	return consts.LighterABI.Pack(
		"fulfillWithdraw",
		common.HexToHash(tx.Hash),
		common.BytesToAddress(tx.Transfer.Memo[:20]),
		new(big.Int).SetUint64(tx.Transfer.Amount))
}

func (h *LighterMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(comm.LighterSessionID, comm.LighterMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				d := &LighterData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling Lighter message: %s", err)
					continue
				}

				d.ErrChn = make(chan error, 1)
				msg := NewLighterMessage(d.Source, d.Destination, d)
				_, err = h.HandleMessage(msg)
				if err != nil {
					log.Err(err).Msgf("Failed handling Lighter message %+v because of: %s", msg, err)
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

func (h *LighterMessageHandler) notify(data *LighterData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.LighterMsg, comm.LighterSessionID)
}
