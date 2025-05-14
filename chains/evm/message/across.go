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
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
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

type TokenMatcher interface {
	DestinationToken(destinationChainId *big.Int, symbol string) (common.Address, error)
}

type ConfirmationWatcher interface {
	WaitForConfirmations(
		ctx context.Context,
		chainID uint64,
		txHash common.Hash,
		token common.Address,
		amount *big.Int) error
}

type AcrossMessageHandler struct {
	client  EventFilterer
	chainID uint64

	pools               map[uint64]common.Address
	confirmationWatcher ConfirmationWatcher
	tokenMatcher        TokenMatcher
	tokenStore          config.TokenStore

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan any
}

func NewAcrossMessageHandler(
	chainID uint64,
	client EventFilterer,
	pools map[uint64]common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	tokenMatcher TokenMatcher,
	tokenStore config.TokenStore,
	confirmationWatcher ConfirmationWatcher,
	sigChn chan any,
) *AcrossMessageHandler {
	return &AcrossMessageHandler{
		chainID:             chainID,
		client:              client,
		pools:               pools,
		coordinator:         coordinator,
		host:                host,
		comm:                comm,
		fetcher:             fetcher,
		sigChn:              sigChn,
		confirmationWatcher: confirmationWatcher,
		tokenMatcher:        tokenMatcher,
		tokenStore:          tokenStore,
	}
}

// HandleMessage finds the Across deposit with the according deposit ID and starts
// the MPC signature process for it. The result will be saved into the signature
// cache through the result channel.
func (h *AcrossMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*AcrossData)

	log.Info().Str("depositId", data.DepositId.String()).Msgf("Handling across message %+v", data)

	sourceChainID := h.chainID
	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	d, err := h.deposit(data.DepositTxHash, data.DepositId)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.confirmationWatcher.WaitForConfirmations(
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
	).Calldata(d.DestinationChainId, data.LiquidityPool)
	if err != nil {
		return nil, err
	}

	unlockHash, err := unlockHash(
		calldata,
		d.OutputAmount,
		common.BytesToAddress(d.OutputToken[12:]),
		d.DestinationChainId,
		h.pools[d.DestinationChainId.Uint64()],
		uint64(d.FillDeadline),
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
				d := &AcrossData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling across message: %s", err)
					continue
				}

				d.ErrChn = make(chan error, 1)
				msg := NewAcrossMessage(d.Source, d.Destination, d)
				_, err = h.HandleMessage(msg)
				if err != nil {
					log.Err(err).Msgf("Failed handling across message %+v because of: %s", msg, err)
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

func (h *AcrossMessageHandler) deposit(hash common.Hash, depositId *big.Int) (*events.AcrossDeposit, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), TRANSACTION_TIMEOUT)
	defer cancel()

	receipt, err := h.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	for _, l := range receipt.Logs {
		if l.Removed {
			continue
		}

		if l.Topics[0] != events.AcrossDepositSig.GetTopic() {
			continue
		}

		if l.Topics[2] != common.HexToHash(common.Bytes2Hex(common.LeftPadBytes(depositId.Bytes(), 32))) {
			continue
		}

		d, err := h.parseDeposit(*l)
		if err != nil {
			return nil, err
		}
		return d, nil
	}

	return nil, fmt.Errorf("deposit with id %s not found", depositId)
}

func (h *AcrossMessageHandler) parseDeposit(l types.Log) (*events.AcrossDeposit, error) {
	d := &events.AcrossDeposit{}
	err := consts.SpokePoolABI.UnpackIntoInterface(d, "FundsDeposited", l.Data)
	if err != nil {
		return nil, err
	}

	if len(l.Topics) < 4 {
		return nil, fmt.Errorf("across deposit missing topics")
	}

	d.DestinationChainId = new(big.Int).SetBytes(l.Topics[1].Bytes())
	d.DepositId = new(big.Int).SetBytes(l.Topics[2].Bytes())
	copy(d.Depositor[:], l.Topics[3].Bytes())

	if common.Bytes2Hex(d.OutputToken[:]) == ZERO_HASH {
		symbol, _, err := h.tokenStore.ConfigByAddress(h.chainID, common.BytesToAddress(d.InputToken[12:]))
		if err != nil {
			return nil, err
		}

		address, err := h.tokenMatcher.DestinationToken(d.DestinationChainId, symbol)
		if err != nil {
			return nil, err
		}

		d.OutputToken = common.BytesToHash(address.Bytes())
	}

	return d, err
}
