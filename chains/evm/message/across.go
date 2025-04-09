package message

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	tssMessage "github.com/sprintertech/sprinter-signing/tss/message"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

const (
	FILTER_LOGS_TIMEOUT = 30 * time.Second
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
		txHash common.Hash,
		token common.Address,
		amount *big.Int) error
	TokenConfig(token common.Address) (string, evm.TokenConfig, error)
}

type AcrossMessageHandler struct {
	client  EventFilterer
	chainID uint64

	pools               map[uint64]common.Address
	confirmationWatcher ConfirmationWatcher
	tokenMatcher        TokenMatcher

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
	}
}

func (h *AcrossMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(fmt.Sprintf("%d-%s", h.chainID, comm.AcrossSessionID), comm.AcrossMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				acrossMsg, err := tssMessage.UnmarshalAcrossMessage(wMsg.Payload)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling across message: %s", err)
					continue
				}

				msg := NewAcrossMessage(acrossMsg.Source, acrossMsg.Destination, AcrossData{
					DepositId:     acrossMsg.DepositId,
					Nonce:         acrossMsg.Nonce,
					Coordinator:   wMsg.From,
					LiquidityPool: common.HexToAddress(acrossMsg.LiqudityPool),
					Caller:        common.HexToAddress(acrossMsg.Caller),
					ErrChn:        make(chan error, 1),
				})
				_, err = h.HandleMessage(msg)
				if err != nil {
					log.Err(err).Msgf("Failed handling across message %+v because of: %s", acrossMsg, err)
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

// HandleMessage finds the Across deposit with the according deposit ID and starts
// the MPC signature process for it. The result will be saved into the signature
// cache through the result channel.
func (h *AcrossMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(AcrossData)

	log.Info().Str("depositId", data.DepositId.String()).Msgf("Handling across message %+v", data)

	sourceChainID := h.chainID
	err := h.notify(m, data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	txHash, d, err := h.deposit(data.DepositId)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.confirmationWatcher.WaitForConfirmations(
		context.Background(),
		txHash,
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

func (h *AcrossMessageHandler) notify(m *message.Message, data AcrossData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := tssMessage.MarshalAcrossMessage(
		data.DepositId,
		data.Nonce,
		data.LiquidityPool.Hex(),
		data.Caller.Hex(),
		m.Source,
		m.Destination)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.AcrossMsg, fmt.Sprintf("%d-%s", h.chainID, comm.AcrossSessionID))
}

func (h *AcrossMessageHandler) deposit(depositId *big.Int) (common.Hash, *events.AcrossDeposit, error) {
	latestBlock, err := h.client.LatestBlock()
	if err != nil {
		return common.Hash{}, nil, err
	}

	q := ethereum.FilterQuery{
		ToBlock:   latestBlock,
		FromBlock: new(big.Int).Sub(latestBlock, big.NewInt(BLOCK_RANGE)),
		Addresses: []common.Address{
			h.pools[h.chainID],
		},
		Topics: [][]common.Hash{
			{
				events.AcrossDepositSig.GetTopic(),
			},
			{},
			{
				common.HexToHash(common.Bytes2Hex(common.LeftPadBytes(depositId.Bytes(), 32))),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), FILTER_LOGS_TIMEOUT)
	defer cancel()

	logs, err := h.client.FilterLogs(ctx, q)
	if err != nil {
		return common.Hash{}, nil, err
	}
	if len(logs) == 0 {
		return common.Hash{}, nil, fmt.Errorf("no deposit found with ID: %s", depositId)
	}
	if logs[0].Removed {
		return common.Hash{}, nil, fmt.Errorf("deposit log removed")
	}

	d, err := h.parseDeposit(logs[0])
	if err != nil {
		return common.Hash{}, nil, err
	}

	return logs[0].TxHash, d, nil
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
		symbol, _, err := h.confirmationWatcher.TokenConfig(common.BytesToAddress(d.InputToken[12:]))
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
