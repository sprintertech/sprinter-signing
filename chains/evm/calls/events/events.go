// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package events

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
)

type EventSig string

func (es EventSig) GetTopic() common.Hash {
	return crypto.Keccak256Hash([]byte(es))
}

const (
	StartKeygenSig EventSig = "StartKeygen()"
	KeyRefreshSig  EventSig = "KeyRefresh(string)"

	AcrossDepositSig EventSig = "FundsDeposited(bytes32,bytes32,uint256,uint256,uint256,uint256,uint32,uint32,uint32,bytes32,bytes32,bytes32,bytes)"
	LifiOpenSig      EventSig = "Open(bytes32,bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes,bytes)"
	MayanDepositSig  EventSig = "OrderCreated(bytes32)"
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
	DestinationChainId  *big.Int
	DepositId           *big.Int
	QuoteTimestamp      uint32
	ExclusivityDeadline uint32
	FillDeadline        uint32
	Depositor           [32]byte
	Recipient           [32]byte
	ExclusiveRelayer    [32]byte
	Message             []byte
}

func (a *AcrossDeposit) ToV3RelayData(originChainID *big.Int) *AcrossV3RelayData {
	return &AcrossV3RelayData{
		Depositor:           a.Depositor,
		Recipient:           a.Recipient,
		ExclusiveRelayer:    a.ExclusiveRelayer,
		InputToken:          a.InputToken,
		OutputToken:         a.OutputToken,
		InputAmount:         a.InputAmount,
		OutputAmount:        a.OutputAmount,
		OriginChainId:       originChainID,
		DepositId:           a.DepositId,
		FillDeadline:        a.FillDeadline,
		ExclusivityDeadline: a.ExclusivityDeadline,
		Message:             a.Message,
	}
}

type AcrossV3RelayData struct {
	Depositor           [32]byte
	Recipient           [32]byte
	ExclusiveRelayer    [32]byte
	InputToken          [32]byte
	OutputToken         [32]byte
	InputAmount         *big.Int
	OutputAmount        *big.Int
	OriginChainId       *big.Int
	DepositId           *big.Int
	FillDeadline        uint32
	ExclusivityDeadline uint32
	Message             []byte
}

func (a *AcrossV3RelayData) Calldata(repaymentChainID *big.Int, repaymentAddress common.Address) ([]byte, error) {
	input, err := consts.SpokePoolABI.Pack("fillRelay", a, repaymentChainID, [32]byte(common.LeftPadBytes(repaymentAddress.Bytes(), 32)))
	if err != nil {
		return []byte{}, err
	}

	return input, nil
}
