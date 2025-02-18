package message

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
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

	DOMAIN_NAME     = ""
	VERSION         = "v1.0.0"
	BORROW_TYPEHASH = "Borrow(address borrowToken,uint256 amount,address target,bytes targetCallData,uint256 nonce,uint256 deadline)"
	PROTOCOL_ID     = 1
)

type EventFilterer interface {
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
}

type AcrossData struct {
	depositId   *big.Int
	coordinator peer.ID
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
	client        EventFilterer
	sourceChainID *big.Int

	across common.Address
	pools  map[uint64]common.Address
	abi    abi.ABI

	coordinator *tss.Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan interface{}
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
					depositId:   acrossMsg.DepositId,
					coordinator: wMsg.From,
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

	d, err := h.deposit(data.depositId)
	if err != nil {
		return nil, err
	}

	unlockHash, err := h.unlockHash(d)
	if err != nil {
		return nil, err
	}

	signing, err := signing.NewSigning(
		new(big.Int).SetBytes(unlockHash),
		data.depositId.Text(16),
		data.depositId.Text(16),
		h.host,
		h.comm,
		h.fetcher)
	if err != nil {
		return nil, err
	}

	err = h.coordinator.Execute(context.Background(), []tss.TssProcess{signing}, h.sigChn, data.coordinator)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *AcrossMessageHandler) notify(m *message.Message, data AcrossData) error {
	msgBytes, err := tssMessage.MarshalAcrossMessage(data.depositId, m.Source, m.Destination)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.AcrossMsg, comm.AcrossSessionID)
}

func (h *AcrossMessageHandler) deposit(depositId *big.Int) (*events.AcrossDeposit, error) {
	q := ethereum.FilterQuery{
		Addresses: []common.Address{
			h.across,
		},
		Topics: [][]common.Hash{
			{events.AcrossDepositSig.GetTopic()},
			nil,
			{common.HexToHash(depositId.Text(16))},
		},
	}
	logs, err := h.client.FilterLogs(context.Background(), q)
	if err != nil {
		return nil, err
	}

	if len(logs) == 0 {
		return nil, fmt.Errorf("no deposit found with ID: %s", depositId)
	}

	return h.parseDeposit(logs[0])
}

func (h *AcrossMessageHandler) parseDeposit(l types.Log) (*events.AcrossDeposit, error) {
	var d *events.AcrossDeposit
	err := h.abi.UnpackIntoInterface(&d, "V3FundsDeposited", l.Data)
	return d, err
}

func (h *AcrossMessageHandler) unlockHash(deposit *events.AcrossDeposit) ([]byte, error) {
	calldata, err := deposit.ToV3RelayData(h.sourceChainID).Calldata()
	if err != nil {
		return []byte{}, nil
	}

	encodedData := crypto.Keccak256(
		crypto.Keccak256(
			[]byte(
				"Borrow(address borrowToken,uint256 amount,address target,bytes targetCallData,uint256 nonce,uint256 deadline)",
			),
		),
		deposit.OutputToken[12:],
		deposit.OutputAmount.Bytes(),
		deposit.Recipient[12:],
		calldata,
		common.LeftPadBytes(h.nonce(deposit).Bytes(), 32),
		new(big.Int).SetUint64(uint64(deposit.FillDeadline)).Bytes(),
	)

	poolAddress := h.pools[h.sourceChainID.Uint64()]
	typedData := apitypes.TypedData{
		Domain: apitypes.TypedDataDomain{
			Name:              DOMAIN_NAME,
			ChainId:           math.NewHexOrDecimal256(deposit.DestinationChainId.Int64()),
			Version:           VERSION,
			VerifyingContract: poolAddress.Hex(),
		},
	}
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return []byte{}, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(encodedData)))
	return crypto.Keccak256(rawData), nil
}

// nonce creates a unique ID from the across deposit id, origin chain id and protocol id.
// Resulting id has this format: [originChainID (8 bits)][protocolID (8 bits)][nonce (240 bits)].
func (h *AcrossMessageHandler) nonce(deposit *events.AcrossDeposit) *big.Int {
	// Create a new big.Int
	nonce := new(big.Int)

	// Set originChainID (64 bits)
	nonce.SetInt64(h.sourceChainID.Int64())
	nonce.Lsh(nonce, 256) // Shift left by 320 bits (248 + 8)

	// Add protocolID in the middle (shifted left by 248 bits)
	protocolInt := big.NewInt(PROTOCOL_ID)
	protocolInt.Lsh(protocolInt, 248)
	nonce.Or(nonce, protocolInt)

	// Add nonce at the end
	nonce.Or(nonce, deposit.DepositId)

	return nonce
}
