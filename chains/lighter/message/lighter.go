package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
	"github.com/sprintertech/sprinter-signing/chains/evm/signature"
	lighterChain "github.com/sprintertech/sprinter-signing/chains/lighter"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/protocol/lighter"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

var (
	ARBITRUM_CHAIN_ID = big.NewInt(42161)
	FEE               = big.NewInt(2000000)
)

type Coordinator interface {
	Execute(ctx context.Context, tssProcesses []tss.TssProcess, resultChn chan interface{}, coordinator peer.ID) error
}

type TxFetcher interface {
	GetTx(hash string) (*lighter.LighterTx, error)
}

type LighterMessageHandler struct {
	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
	sigChn      chan any

	lighterAddress   common.Address
	usdcAddress      common.Address
	repaymentAccount string
	txFetcher        TxFetcher
}

func NewLighterMessageHandler(
	lighterAddress common.Address,
	usdcAddress common.Address,
	repaymentAccount string,
	txFetcher TxFetcher,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	sigChn chan any,
) *LighterMessageHandler {
	return &LighterMessageHandler{
		txFetcher:        txFetcher,
		usdcAddress:      usdcAddress,
		repaymentAccount: repaymentAccount,
		lighterAddress:   lighterAddress,
		coordinator:      coordinator,
		host:             host,
		comm:             comm,
		fetcher:          fetcher,
		sigChn:           sigChn,
	}
}

// HandleMessage finds the Mayan deposit with the according deposit ID and starts
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

	unlockHash, err := signature.BorrowUnlockHash(
		calldata,
		new(big.Int).SetUint64(tx.Transfer.USDCAmount),
		h.usdcAddress,
		ARBITRUM_CHAIN_ID,
		h.lighterAddress,
		data.Deadline,
		h.lighterAddress,
		data.LiquidityPool,
		data.Nonce)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	sessionID := fmt.Sprintf("%d-%s", lighterChain.LIGHTER_DOMAIN_ID, data.OrderHash)
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

func (h *LighterMessageHandler) verifyWithdrawal(tx *lighter.LighterTx) error {
	if tx.Type != lighter.TxTypeL2Transfer {
		return errors.New("invalid transaction type")
	}

	if strconv.Itoa(tx.Transfer.ToAccountIndex) != h.repaymentAccount {
		return errors.New("transfer account index invalid")
	}

	if tx.Transfer.USDCAmount <= FEE.Uint64() {
		return errors.New("fee higher than withdrawal amount")
	}

	return nil
}

func (h *LighterMessageHandler) calldata(tx *lighter.LighterTx) ([]byte, error) {
	borrowAmount := new(big.Int).Sub(new(big.Int).SetUint64(tx.Transfer.USDCAmount), FEE)
	return consts.LighterABI.Pack(
		"fulfillWithdraw",
		common.HexToHash(tx.Hash),
		common.BytesToAddress(tx.Transfer.Memo[:20]),
		borrowAmount)
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
