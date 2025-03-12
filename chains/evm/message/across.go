package message

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
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
	AcrossMessage = "AcrossMessage"

	DOMAIN_NAME = "LiquidityPool"
	VERSION     = "1.0.0"
	PROTOCOL_ID = 1
	BLOCK_RANGE = 1000

	TIMEOUT = 10 * time.Minute
)

type EventFilterer interface {
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	LatestBlock() (*big.Int, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type AcrossData struct {
	DepositId     *big.Int
	LiquidityPool common.Address
	Caller        common.Address
	Coordinator   peer.ID
	ErrChn        chan error
}

func NewAcrossMessage(source, destination uint64, acrossData AcrossData) *message.Message {
	return &message.Message{
		Source:      source,
		Destination: destination,
		Data:        acrossData,
		Type:        AcrossMessage,
		Timestamp:   time.Now(),
	}
}

type Coordinator interface {
	Execute(ctx context.Context, tssProcesses []tss.TssProcess, resultChn chan interface{}, coordinator peer.ID) error
}

type TokenPricer interface {
	TokenPrice(symbol string) (float64, error)
}

type AcrossMessageHandler struct {
	client EventFilterer

	tokens        map[string]evm.TokenConfig
	confirmations map[uint64]uint64
	blocktime     time.Duration
	tokenPricer   TokenPricer
	pool          common.Address

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan interface{}
}

func NewAcrossMessageHandler(
	client EventFilterer,
	pool common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	sigChn chan interface{},
	tokens map[string]evm.TokenConfig,
	confirmations map[uint64]uint64,
	blocktime time.Time,
) *AcrossMessageHandler {
	return &AcrossMessageHandler{
		client:        client,
		pool:          pool,
		coordinator:   coordinator,
		host:          host,
		comm:          comm,
		fetcher:       fetcher,
		sigChn:        sigChn,
		tokens:        tokens,
		confirmations: confirmations,
	}
}

func (h *AcrossMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(comm.AcrossSessionID, comm.AcrossMsg, msgChn)

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
					Coordinator:   wMsg.From,
					LiquidityPool: common.HexToAddress(acrossMsg.LiqudityPool),
					Caller:        common.HexToAddress(acrossMsg.Caller),
					ErrChn:        make(chan error),
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

	sourceChainID := m.Destination
	if data.Coordinator == peer.ID("") {
		data.Coordinator = h.host.ID()

		err := h.notify(m, data)
		if err != nil {
			log.Warn().Msgf("Failed to notify relayers because of %s", err)
		}
	}

	txHash, d, err := h.deposit(data.DepositId)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	confirmations, err := h.minimalConfirmations(d)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	data.ErrChn <- nil

	err = h.waitForConfirmations(txHash, confirmations)
	if err != nil {
		return nil, err
	}

	unlockHash, err := h.unlockHash(d, sourceChainID, data)
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
	msgBytes, err := tssMessage.MarshalAcrossMessage(
		data.DepositId,
		data.LiquidityPool.Hex(),
		data.Caller.Hex(),
		m.Source,
		m.Destination)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.AcrossMsg, comm.AcrossSessionID)
}

func (h *AcrossMessageHandler) minimalConfirmations(d *events.AcrossDeposit) (uint64, error) {
	symbol, c, err := h.tokenConfig(d)
	if err != nil {
		return 0, err
	}

	price, err := h.tokenPricer.TokenPrice(symbol)
	if err != nil {
		return 0, err
	}

	fmt.Println(price)

	orderValue, _ := new(big.Float).Quo(
		new(big.Float).Mul(big.NewFloat(price), new(big.Float).SetInt(d.InputAmount)),
		new(big.Float).SetUint64(uint64(c.Decimals)),
	).Float64()

	for bucket, confirmation := range h.confirmations {
		if uint64(orderValue) < bucket {
			return confirmation, nil
		}
	}

	fmt.Println(orderValue)

	return 0, fmt.Errorf("order value %f exceeds confirmation buckets", orderValue)
}

func (h *AcrossMessageHandler) waitForConfirmations(
	txHash common.Hash,
	requiredConfirmations uint64,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for confirmations")
		default:
			txReceipt, err := h.client.TransactionReceipt(ctx, txHash)
			if err != nil {
				log.Warn().Msgf("Error fetching transaction receipt: %v\n", err)
				time.Sleep(h.blocktime)
				continue
			}

			if txReceipt == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			currentBlock, err := h.client.LatestBlock()
			if err != nil {
				log.Warn().Msgf("Error fetching current block: %v\n", err)
				time.Sleep(h.blocktime)
				continue
			}

			confirmations := new(big.Int).Sub(currentBlock, txReceipt.BlockNumber)
			if confirmations.Cmp(new(big.Int).SetUint64(requiredConfirmations)) != -1 {
				return nil
			}

			fmt.Println(time.Duration(uint64(h.blocktime) * (requiredConfirmations - confirmations.Uint64())))

			time.Sleep(time.Duration(uint64(h.blocktime) * (requiredConfirmations - confirmations.Uint64())))
		}
	}
}

func (h *AcrossMessageHandler) tokenConfig(d *events.AcrossDeposit) (string, evm.TokenConfig, error) {
	for symbol, c := range h.tokens {
		if c.Address == common.BytesToAddress(d.InputToken[12:]) {
			return symbol, c, nil
		}
	}

	return "", evm.TokenConfig{}, fmt.Errorf("token %s not supported", common.Bytes2Hex(d.InputToken[:]))
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
			h.pool,
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
	logs, err := h.client.FilterLogs(context.Background(), q)
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
	return d, err
}

func (h *AcrossMessageHandler) unlockHash(
	deposit *events.AcrossDeposit,
	sourceChainId uint64,
	data AcrossData,
) ([]byte, error) {
	calldata, err := deposit.ToV3RelayData(
		new(big.Int).SetUint64(sourceChainId),
	).Calldata(deposit.DestinationChainId, data.LiquidityPool)
	if err != nil {
		return []byte{}, err
	}

	msg := apitypes.TypedDataMessage{
		"caller":         data.Caller.Hex(),
		"borrowToken":    common.BytesToAddress(deposit.OutputToken[12:]).Hex(),
		"amount":         deposit.OutputAmount,
		"target":         common.BytesToAddress(deposit.Recipient[12:]).Hex(),
		"targetCallData": hexutil.Encode(calldata),
		"nonce":          h.nonce(deposit, sourceChainId),
		"deadline":       new(big.Int).SetUint64(uint64(deposit.FillDeadline)),
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Borrow": []apitypes.Type{
				{Name: "caller", Type: "address"},
				{Name: "borrowToken", Type: "address"},
				{Name: "amount", Type: "uint256"},
				{Name: "target", Type: "address"},
				{Name: "targetCallData", Type: "bytes"},
				{Name: "nonce", Type: "uint256"},
				{Name: "deadline", Type: "uint256"},
			},
		},
		PrimaryType: "Borrow",
		Domain: apitypes.TypedDataDomain{
			Name:              DOMAIN_NAME,
			ChainId:           math.NewHexOrDecimal256(deposit.DestinationChainId.Int64()),
			Version:           VERSION,
			VerifyingContract: data.LiquidityPool.Hex(),
		},
		Message: msg,
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return []byte{}, err
	}

	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return []byte{}, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(messageHash)))
	return crypto.Keccak256(rawData), nil
}

// nonce creates a unique ID from the across deposit id, origin chain id and protocol id.
// Resulting id has this format: [originChainID (8 bits)][protocolID (8 bits)][nonce (240 bits)].
func (h *AcrossMessageHandler) nonce(deposit *events.AcrossDeposit, sourceChainId uint64) *big.Int {
	// Create a new big.Int
	nonce := new(big.Int)

	// Set originChainID (8 bits)
	originChainID := new(big.Int).SetUint64(sourceChainId)
	originChainID.And(originChainID, new(big.Int).SetUint64(0xFF)) // Ensure only 8 bits are used
	nonce.Lsh(originChainID, 248)                                  // Shift left by 248 bits

	// Add protocolID in the middle (8 bits, shifted left by 240 bits)
	protocolInt := big.NewInt(PROTOCOL_ID)
	protocolInt.And(protocolInt, new(big.Int).SetUint64(0xFF)) // Ensure only 8 bits are used
	protocolInt.Lsh(protocolInt, 240)
	nonce.Or(nonce, protocolInt)

	// Add depositId at the end (240 bits)
	depositIdMask := new(big.Int).Lsh(big.NewInt(1), 240)
	depositIdMask.Sub(depositIdMask, big.NewInt(1))                 // Create a mask of 240 1's
	depositId := new(big.Int).And(deposit.DepositId, depositIdMask) // Ensure only 240 bits are used
	nonce.Or(nonce, depositId)

	return nonce
}
