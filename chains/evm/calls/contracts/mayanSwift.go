// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package contracts

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sprintertech/sprinter-signing/chains/evm"
	"github.com/sygmaprotocol/sygma-core/chains/evm/client"
	"github.com/sygmaprotocol/sygma-core/chains/evm/contracts"
)

type MayanKey struct {
	Trader       common.Hash
	SrcChainId   uint16
	TokenIn      common.Hash
	DestAddr     common.Hash
	DestChainId  uint16
	TokenOut     common.Hash
	MinAmountOut uint64
	GasDrop      uint64
	CancelFee    uint64
	RefundFee    uint64
	Deadline     uint64
	ReferrerAddr common.Hash
	ReferrerBps  uint8
	ProtocolBps  uint8
	AuctionMode  uint8
	Random       common.Hash
}

type MayanFulfillMsg struct {
	Action         uint8
	OrderHash      [32]byte
	DestChainId    uint16
	DestAddr       [32]byte
	Driver         [32]byte
	TokenOut       [32]byte
	PromisedAmount uint64
	GasDrop        uint64
	Deadline       uint64
	ReferrerAddr   [32]byte
	ReferrerBps    uint8
	ProtocolBps    uint8
	SrcChainId     uint16
	TokenIn        [32]byte
}

type MayanSwiftContract struct {
	contracts.Contract
	client client.Client
	tokens map[string]evm.TokenConfig
}

func NewMayanSwiftContract(
	client client.Client,
	address common.Address,
	l1Tokens map[string]evm.TokenConfig,
) *MayanSwiftContract {
	return &MayanSwiftContract{
		Contract: contracts.NewContract(address, abi.ABI{}, nil, client, nil),
		client:   client,
		tokens:   l1Tokens,
	}
}

func (c *MayanSwiftContract) DecodeFulfillCall(calldata []byte) (*MayanFulfillMsg, error) {
	method, ok := c.ABI.Methods["fullfillOrder"]
	if !ok {
		return nil, fmt.Errorf("no method fulfillOrder")
	}

	params := make(map[string]interface{})
	err := method.Inputs.UnpackIntoMap(params, calldata)
	if err != nil {
		return nil, err
	}

	encodedVM, ok := params["encodedVM"].([]byte)
	if !ok {
		return nil, fmt.Errorf("failed decoding VM data")
	}

	return c.ParseFulfillPayload(encodedVM)
}

func (c *MayanSwiftContract) ParseFulfillPayload(calldata []byte) (*MayanFulfillMsg, error) {
	res, err := c.CallContract("parseFulfillPayload", calldata)
	if err != nil {
		return nil, err
	}

	msg, ok := res[0].(*MayanFulfillMsg)
	if !ok {
		return nil, fmt.Errorf("cannot convert fullfill payload to msg")
	}

	return msg, nil
}
