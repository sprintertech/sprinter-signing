// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package events

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type EventSig string

func (es EventSig) GetTopic() common.Hash {
	return crypto.Keccak256Hash([]byte(es))
}

const (
	StartKeygenSig EventSig = "StartKeygen()"
	KeyRefreshSig  EventSig = "KeyRefresh(string)"

	AcrossDepositSig EventSig = "FundsDeposited(address,address,uint256,uin256,uint256,uint32,uint32,uint32,uint32,address,address,address,bytes)"
)

// Refresh struct holds key refresh event data
type Refresh struct {
	// SHA1 hash of topology file
	Hash string
}

type AcrossDeposit struct {
	InputToken          [32]byte
	OutputToken         [32]byte
	InputAmount         *big.Int
	OutputAmount        *big.Int
	RepaymentChainId    *big.Int
	OriginChainId       *big.Int
	DepositId           *big.Int
	QuoteTimestamp      uint32
	FillDeadline        uint32
	ExclusivityDeadline uint32
	Depositor           [32]byte
	Recipient           [32]byte
	ExclusiveRelayer    common.Address
	Message             []byte
}
