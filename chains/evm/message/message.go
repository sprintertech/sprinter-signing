package message

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

const (
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
		Type:        message.MessageType(comm.AcrossMsg.String()),
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
		Type:        message.MessageType(comm.MayanMsg.String()),
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
		Type:        message.MessageType(comm.RhinestoneMsg.String()),
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
		Type:        message.MessageType(comm.LifiEscrowMsg.String()),
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
		Type:        message.MessageType(comm.LifiUnlockMsg.String()),
		Timestamp:   time.Now(),
	}
}
