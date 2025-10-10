package message

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

const (
	AcrossMessage     = "AcrossMessage"
	MayanMessage      = "MayanMessage"
	LifiEscrowMessage = "LifiEscrowMessage"
	LifiUnlockMessage = "LifiUnlockMessage"

	DOMAIN_NAME = "LiquidityPool"
	VERSION     = "1.0.0"
	TIMEOUT     = 10 * time.Minute
	BLOCK_RANGE = 1000
)

type AcrossData struct {
	ErrChn chan error `json:"-"`

	DepositTxHash    common.Hash
	DepositId        *big.Int
	Nonce            *big.Int
	LiquidityPool    common.Address
	RepaymentChainID uint64
	Caller           common.Address
	Coordinator      peer.ID
	Source           uint64
	Destination      uint64
}

func NewAcrossMessage(source, destination uint64, acrossData *AcrossData) *message.Message {
	return &message.Message{
		Source:      source,
		Destination: destination,
		Data:        acrossData,
		Type:        AcrossMessage,
		Timestamp:   time.Now(),
	}
}

type MayanData struct {
	ErrChn chan error `json:"-"`

	OrderHash     string
	Coordinator   peer.ID
	LiquidityPool common.Address
	Caller        common.Address
	DepositTxHash string
	Calldata      string
	Nonce         *big.Int
	BorrowAmount  *big.Int
	Source        uint64
	Destination   uint64
}

func NewMayanMessage(source, destination uint64, mayanData *MayanData) *message.Message {
	return &message.Message{
		Source:      source,
		Destination: destination,
		Data:        mayanData,
		Type:        MayanMessage,
		Timestamp:   time.Now(),
	}
}

type RhinestoneData struct {
	ErrChn chan error `json:"-"`

	BundleID      string
	Coordinator   peer.ID
	LiquidityPool common.Address
	Caller        common.Address
	BorrowAmount  *big.Int
	Nonce         *big.Int
	Source        uint64
	Destination   uint64
}

func NewRhinestoneMessage(source, destination uint64, rhinestoneData *RhinestoneData) *message.Message {
	return &message.Message{
		Source:      source,
		Destination: destination,
		Data:        rhinestoneData,
		Type:        MayanMessage,
		Timestamp:   time.Now(),
	}
}

type LifiEscrowData struct {
	ErrChn chan error `json:"-"`

	OrderID       string
	Coordinator   peer.ID
	LiquidityPool common.Address
	Caller        common.Address
	DepositTxHash string
	BorrowAmount  *big.Int
	Nonce         *big.Int
	Source        uint64
	Destination   uint64
}

func NewLifiEscrowData(source, destination uint64, lifiData *LifiEscrowData) *message.Message {
	return &message.Message{
		Source:      source,
		Destination: destination,
		Data:        lifiData,
		Type:        LifiEscrowMessage,
		Timestamp:   time.Now(),
	}
}

type LifiUnlockData struct {
	SigChn chan interface{} `json:"-"`

	OrderID string
	Settler common.Address

	Coordinator peer.ID
	Source      uint64
	Destination uint64
}

func NewLifiUnlockMessage(source, destination uint64, lifiData *LifiUnlockData) *message.Message {
	return &message.Message{
		Source:      source,
		Destination: destination,
		Data:        lifiData,
		Type:        LifiUnlockMessage,
		Timestamp:   time.Now(),
	}
}

// borrowUnlockHash calculates the hash that has to be signed and submitted on-chain to the liquidity
// pool contract.
func borrowUnlockHash(
	calldata []byte,
	outputAmount *big.Int,
	outputToken common.Address,
	destinationChainId *big.Int,
	target common.Address,
	deadline uint64,
	caller common.Address,
	liquidityPool common.Address,
	nonce *big.Int,
) ([]byte, error) {
	msg := apitypes.TypedDataMessage{
		"caller":         caller.Hex(),
		"borrowToken":    outputToken.Hex(),
		"amount":         outputAmount,
		"target":         target.Hex(),
		"targetCallData": calldata,
		"nonce":          nonce,
		"deadline":       new(big.Int).SetUint64(deadline),
	}

	chainId := math.HexOrDecimal256(*destinationChainId)
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
			ChainId:           &chainId,
			Version:           VERSION,
			VerifyingContract: liquidityPool.Hex(),
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

// borrowManyUnlockHash calculates the hash that has to be signed and submitted on-chain to the liquidity
// pool contract.
func borrowManyUnlockHash(
	calldata []byte,
	outputAmounts []*big.Int,
	outputTokens []common.Address,
	destinationChainId *big.Int,
	target common.Address,
	deadline uint64,
	caller common.Address,
	liquidityPool common.Address,
	nonce *big.Int,
) ([]byte, error) {
	hexOutputTokens := make([]string, len(outputTokens))
	for i, token := range outputTokens {
		hexOutputTokens[i] = token.Hex()
	}

	msg := apitypes.TypedDataMessage{
		"caller":         caller.Hex(),
		"borrowTokens":   hexOutputTokens,
		"amounts":        outputAmounts,
		"target":         target.Hex(),
		"targetCallData": calldata,
		"nonce":          nonce,
		"deadline":       new(big.Int).SetUint64(deadline),
	}

	chainId := math.HexOrDecimal256(*destinationChainId)
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"BorrowMany": []apitypes.Type{
				{Name: "caller", Type: "address"},
				{Name: "borrowTokens", Type: "address[]"},
				{Name: "amounts", Type: "uint256[]"},
				{Name: "target", Type: "address"},
				{Name: "targetCallData", Type: "bytes"},
				{Name: "nonce", Type: "uint256"},
				{Name: "deadline", Type: "uint256"},
			},
		},
		PrimaryType: "BorrowMany",
		Domain: apitypes.TypedDataDomain{
			Name:              DOMAIN_NAME,
			ChainId:           &chainId,
			Version:           VERSION,
			VerifyingContract: liquidityPool.Hex(),
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
