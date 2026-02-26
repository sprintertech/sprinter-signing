package message

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
	"github.com/sprintertech/sprinter-signing/chains/evm/signature"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

const (
	TRANSACTION_TIMEOUT = 30 * time.Second
)

type EventFilterer interface {
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	LatestBlock() (*big.Int, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type Coordinator interface {
	Execute(ctx context.Context, tssProcesses []tss.TssProcess, resultChn chan interface{}, coordinator peer.ID) error
}

type ConfirmationWatcher interface {
	WaitForTokenConfirmations(
		ctx context.Context,
		chainID uint64,
		txHash common.Hash,
		token common.Address,
		amount *big.Int) error
	WaitForOrderConfirmations(
		ctx context.Context,
		chainID uint64,
		txHash common.Hash,
		orderValue float64) error
}

type DepositFetcher interface {
	Deposit(ctx context.Context, hash common.Hash, depositID *big.Int) (*events.AcrossDeposit, error)
}

type AcrossMessageHandler struct {
	chainID uint64

	pools               map[uint64]common.Address
	repayers            map[uint64]common.Address
	confirmationWatcher ConfirmationWatcher
	depositFetcher      DepositFetcher

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan any
}

func NewAcrossMessageHandler(
	chainID uint64,
	pools map[uint64]common.Address,
	repayers map[uint64]common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	depositFetcher DepositFetcher,
	confirmationWatcher ConfirmationWatcher,
	sigChn chan any,
) *AcrossMessageHandler {
	return &AcrossMessageHandler{
		chainID:             chainID,
		pools:               pools,
		repayers:            repayers,
		coordinator:         coordinator,
		host:                host,
		comm:                comm,
		fetcher:             fetcher,
		sigChn:              sigChn,
		confirmationWatcher: confirmationWatcher,
		depositFetcher:      depositFetcher,
	}
}

// HandleMessage finds the Across deposit with the according deposit ID and starts
// the MPC signature process for it. The result will be saved into the signature
// cache through the result channel.
func (h *AcrossMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*AcrossData)

	log.Info().Str("depositId", data.DepositId.String()).Msgf("Handling across message %+v", data)

	sourceChainID := h.chainID
	repaymentAddress, ok := h.repayers[data.RepaymentChainID]
	if !ok {
		err := fmt.Errorf("invalid repayment chain %d", data.RepaymentChainID)
		data.ErrChn <- err
		return nil, err
	}

	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	d, err := h.depositFetcher.Deposit(context.Background(), data.DepositTxHash, data.DepositId)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	if data.BorrowAmount.Cmp(d.InputAmount) > 0 {
		err := fmt.Errorf("borrow amount exceeds input amount")
		data.ErrChn <- err
		return nil, err
	}

	err = h.confirmationWatcher.WaitForTokenConfirmations(
		context.Background(),
		h.chainID,
		data.DepositTxHash,
		common.BytesToAddress(d.InputToken[12:]),
		d.InputAmount)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	data.ErrChn <- nil

	calldata, err := d.ToV3RelayData(
		new(big.Int).SetUint64(sourceChainID),
	).Calldata(new(big.Int).SetUint64(data.RepaymentChainID), repaymentAddress)
	if err != nil {
		return nil, err
	}

	unlockHash, err := signature.BorrowUnlockHash(
		calldata,
		data.BorrowAmount,
		common.BytesToAddress(d.OutputToken[12:]),
		d.DestinationChainId,
		h.pools[d.DestinationChainId.Uint64()],
		data.Deadline,
		data.Caller,
		data.LiquidityPool,
		data.Nonce,
	)
	if err != nil {
		return nil, err
	}

	sessionID := fmt.Sprintf("%d-%s", sourceChainID, data.DepositId)
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

func (h *AcrossMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(fmt.Sprintf("%d-%s", h.chainID, comm.AcrossSessionID), comm.AcrossMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				go func(wMsg *comm.WrappedMessage) {
					d := &AcrossData{}
					err := json.Unmarshal(wMsg.Payload, d)
					if err != nil {
						log.Warn().Msgf("Failed unmarshaling across message: %s", err)
						return
					}

					d.ErrChn = make(chan error, 1)
					msg := NewAcrossMessage(d.Source, d.Destination, d)
					_, err = h.HandleMessage(msg)
					if err != nil {
						log.Err(err).Msgf("Failed handling across message %+v because of: %s", msg, err)
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

func (h *AcrossMessageHandler) notify(data *AcrossData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.AcrossMsg, fmt.Sprintf("%d-%s", h.chainID, comm.AcrossSessionID))
}
