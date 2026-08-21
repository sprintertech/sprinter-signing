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
	"github.com/sprintertech/sprinter-signing/chains"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/config"
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
	chainID    uint64
	tokenStore config.TokenStore

	pools               map[uint64]common.Address
	repayers            map[uint64]common.Address
	confirmationWatcher ConfirmationWatcher
	depositFetcher      DepositFetcher

	host host.Host
	comm comm.Communication

	sigChn  chan any
	msgChan chan []*message.Message
}

func NewAcrossMessageHandler(
	chainID uint64,
	tokenStore config.TokenStore,
	pools map[uint64]common.Address,
	repayers map[uint64]common.Address,
	host host.Host,
	comm comm.Communication,
	depositFetcher DepositFetcher,
	confirmationWatcher ConfirmationWatcher,
	sigChn chan any,
	msgChan chan []*message.Message,
) *AcrossMessageHandler {
	return &AcrossMessageHandler{
		chainID:             chainID,
		tokenStore:          tokenStore,
		pools:               pools,
		repayers:            repayers,
		host:                host,
		comm:                comm,
		sigChn:              sigChn,
		msgChan:             msgChan,
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

	sourceTokenAddress := common.BytesToAddress(d.InputToken[:])
	symbol, srcToken, err := h.tokenStore.ConfigByAddress(h.chainID, sourceTokenAddress)
	if err != nil {
		err = fmt.Errorf(
			"failed to get source token for address %s on chain %d: %w",
			sourceTokenAddress.Hex(),
			h.chainID,
			err,
		)
		data.ErrChn <- err
		return nil, err
	}

	destToken, err := h.tokenStore.ConfigBySymbol(
		d.DestinationChainId.Uint64(),
		symbol,
	)
	if err != nil {
		err = fmt.Errorf(
			"failed to get destination token by symbol %s on chain %d: %w",
			symbol,
			d.DestinationChainId.Uint64(),
			err,
		)
		data.ErrChn <- err
		return nil, err
	}

	scaledInputAmount := chains.ScaleTokenAmount(d.InputAmount, int64(srcToken.Decimals), int64(destToken.Decimals))
	if data.BorrowAmount.Cmp(scaledInputAmount) > 0 {
		err := fmt.Errorf("borrow amount exceeds input amount")
		data.ErrChn <- err
		return nil, err
	}

	destChainID := d.DestinationChainId.Uint64()
	target, ok := h.pools[destChainID]
	if !ok {
		data.ErrChn <- err
		return nil, fmt.Errorf("no across pool configured for chain %d", destChainID)
	}

	err = h.confirmationWatcher.WaitForTokenConfirmations(
		context.Background(),
		h.chainID,
		data.DepositTxHash,
		sourceTokenAddress,
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

	sessionID := signing.SessionID(sourceChainID, data.DepositId.String())
	h.msgChan <- []*message.Message{NewSignMessage(0, destChainID, &SignRequest{
		Calldata:      calldata,
		BorrowAmount:  data.BorrowAmount,
		BorrowToken:   common.BytesToAddress(d.OutputToken[:]).Hex(),
		Target:        target.Hex(),
		Deadline:      data.Deadline,
		Caller:        data.Caller,
		LiquidityPool: data.LiquidityPool,
		Nonce:         data.Nonce,
		SessionID:     sessionID,
		Coordinator:   data.Coordinator,
		ResultChn:     h.sigChn,
	})}
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
