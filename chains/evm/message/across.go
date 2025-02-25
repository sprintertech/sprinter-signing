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

	DOMAIN_NAME     = "LiquidityPool"
	VERSION         = "v1.0.0"
	BORROW_TYPEHASH = "Borrow(address borrowToken,uint256 amount,address target,bytes targetCallData,uint256 nonce,uint256 deadline)"
	PROTOCOL_ID     = 1
	LIQUIDITY_POOL  = "0x5c7BCd6E7De5423a257D81B442095A1a6ced35C5"
	BLOCK_RANGE     = 1000
)

type EventFilterer interface {
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	LatestBlock() (*big.Int, error)
}

type AcrossData struct {
	DepositId     *big.Int
	Coordinator   peer.ID
	SourceChainId *big.Int
}

func NewAcrossMessage(source uint8, destination uint8, acrossData AcrossData) *message.Message {
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

type AcrossMessageHandler struct {
	client EventFilterer

	pools map[uint64]common.Address

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan interface{}
}

func NewAcrossMessageHandler(
	client EventFilterer,
	pools map[uint64]common.Address,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	sigChn chan interface{},
) *AcrossMessageHandler {
	return &AcrossMessageHandler{
		client:      client,
		pools:       pools,
		coordinator: coordinator,
		host:        host,
		comm:        comm,
		fetcher:     fetcher,
		sigChn:      sigChn,
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
					SourceChainId: acrossMsg.SourceChainId,
					Coordinator:   wMsg.From,
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
	err := h.notify(m, data)
	if err != nil {
		return nil, err
	}

	d, err := h.Deposit(data.DepositId, data.SourceChainId)
	if err != nil {
		return nil, err
	}

	unlockHash, err := h.unlockHash(d, data.SourceChainId)
	if err != nil {
		return nil, err
	}

	signing, err := signing.NewSigning(
		new(big.Int).SetBytes(unlockHash),
		data.DepositId.Text(16),
		data.DepositId.Text(16),
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
	msgBytes, err := tssMessage.MarshalAcrossMessage(data.DepositId, data.SourceChainId, m.Source, m.Destination)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.AcrossMsg, comm.AcrossSessionID)
}

func (h *AcrossMessageHandler) Deposit(depositId, sourceChainId *big.Int) (*events.AcrossDeposit, error) {
	latestBlock, err := h.client.LatestBlock()
	if err != nil {
		return nil, err
	}

	q := ethereum.FilterQuery{
		ToBlock:   latestBlock,
		FromBlock: new(big.Int).Sub(latestBlock, big.NewInt(BLOCK_RANGE)),
		Addresses: []common.Address{
			h.pools[sourceChainId.Uint64()],
		},
		Topics: [][]common.Hash{
			{
				events.AcrossDepositSig.GetTopic(),
			},
			{},
			{
				common.HexToHash(depositId.Text(16)),
			},
		},
	}
	logs, err := h.client.FilterLogs(context.Background(), q)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, fmt.Errorf("no deposit found with ID: %s", depositId)
	}
	if logs[0].Removed {
		return nil, fmt.Errorf("deposit log removed")
	}

	return h.parseDeposit(logs[0])
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

func (h *AcrossMessageHandler) unlockHash(deposit *events.AcrossDeposit, sourceChainId *big.Int) ([]byte, error) {
	lpAddress := common.HexToAddress(LIQUIDITY_POOL)
	calldata, err := deposit.ToV3RelayData(sourceChainId).Calldata(deposit.DestinationChainId, lpAddress)
	if err != nil {
		return []byte{}, err
	}
	msg := apitypes.TypedDataMessage{
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
			VerifyingContract: lpAddress.Hex(),
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
func (h *AcrossMessageHandler) nonce(deposit *events.AcrossDeposit, sourceChainId *big.Int) *big.Int {
	// Create a new big.Int
	nonce := new(big.Int)

	// Set originChainID (64 bits)
	nonce.SetInt64(sourceChainId.Int64())
	nonce.Lsh(nonce, 248) // Shift left by 248 bits (240 + 8)

	// Add protocolID in the middle (shifted left by 240 bits)
	protocolInt := big.NewInt(PROTOCOL_ID)
	protocolInt.Lsh(protocolInt, 240)
	nonce.Or(nonce, protocolInt)

	// Add nonce at the end
	nonce.Or(nonce, deposit.DepositId)

	return nonce
}
