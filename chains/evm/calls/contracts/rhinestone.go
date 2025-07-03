// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package contracts

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
)

type SegmentData struct {
	TokenIn           [2][]*big.Int  `abi:"tokenIn"`
	TokenOut          [2][]*big.Int  `abi:"tokenOut"`
	OriginModule      common.Address `abi:"originModule"`
	OriginWETHAddress common.Address `abi:"originWETHAddress"`
	OriginChainId     *big.Int       `abi:"originChainId"`
	CompactNonce      *big.Int       `abi:"compactNonce"`
}

type IntentFillPayload struct {
	Segments        []SegmentData `abi:"segments"`
	Message         []byte        `abi:"message"`
	OrchestratorSig []byte        `abi:"orchestratorSig"`
}

type AccountCreation struct {
	Account  common.Address `abi:"account"`
	InitCode []byte         `abi:"initCode"`
}

type FillInput struct {
	Payload            IntentFillPayload `abi:"payload"`
	ExclusiveRelayer   common.Address    `abi:"exclusiveRelayer"`
	RepaymentAddresses []common.Address  `abi:"repaymentAddresses"`
	RepaymentChainIds  []*big.Int        `abi:"repaymentChainIds"`
	AccountCreation    AccountCreation   `abi:"accountCreation"`
}

type RhinestoneContract struct {
	abi abi.ABI
}

func NewRhinestoneContract() *RhinestoneContract {
	return &RhinestoneContract{
		abi: consts.RhinestoneABI,
	}

}

func (c *RhinestoneContract) DecodeFillCall(calldata []byte) (*FillInput, error) {
	var fillInput FillInput
	method := c.abi.Methods["fill"]
	res, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return nil, err
	}

	err = method.Inputs.Copy(&fillInput, res)
	if err != nil {
		return nil, err
	}

	return &fillInput, nil
}
