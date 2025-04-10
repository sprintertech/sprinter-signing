// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package contracts

import (
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
	"github.com/sprintertech/sprinter-signing/protocol/mayan"
	"github.com/sygmaprotocol/sygma-core/chains/evm/client"
	"github.com/sygmaprotocol/sygma-core/chains/evm/contracts"
)

type OrderStatus uint8

const (
	OrderCreated OrderStatus = 1

	WORMHOLE_DECIMALS = 8
)

type MayanOrder struct {
	Status      OrderStatus
	AmountIn    uint64
	DestChainId uint16
}

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
}

func NewMayanSwiftContract(
	client client.Client,
	address common.Address,
) *MayanSwiftContract {
	return &MayanSwiftContract{
		Contract: contracts.NewContract(address, consts.MayanSwiftABI, nil, client, nil),
		client:   client,
	}
}

func (c *MayanSwiftContract) GetOrder(
	msg *MayanFulfillMsg,
	swap *mayan.MayanSwap,
	srcTokenDecimals uint8) (*MayanOrder, error) {
	res, err := c.CallContract("encodeKey", &MayanKey{
		Trader:       common.HexToHash(swap.Trader),
		SrcChainId:   msg.SrcChainId,
		TokenIn:      msg.TokenIn,
		DestAddr:     msg.DestAddr,
		DestChainId:  msg.DestChainId,
		TokenOut:     msg.TokenOut,
		MinAmountOut: msg.PromisedAmount,
		GasDrop:      msg.GasDrop,
		CancelFee:    convertFloatToUint(swap.RedeemRelayerFee, srcTokenDecimals),
		RefundFee:    convertFloatToUint(swap.RefundRelayerFee, srcTokenDecimals),
		Deadline:     msg.Deadline,
		ReferrerAddr: msg.ReferrerAddr,
		ReferrerBps:  msg.ReferrerBps,
		ProtocolBps:  swap.MayanBps,
		AuctionMode:  swap.AuctionMode,
		Random:       common.HexToHash(swap.RandomKey),
	})
	if err != nil {
		return nil, err
	}
	key, ok := res[0].([32]byte)
	if !ok {
		return nil, fmt.Errorf("cannot convert key to [32]byte")
	}

	res, err = c.CallContract("orders", crypto.Keccak256(key[:]))
	if err != nil {
		return nil, err
	}

	fmt.Println("RESULT")
	fmt.Printf("%+v", res[0])

	o, ok := res[0].(*MayanOrder)
	if !ok {
		return nil, fmt.Errorf("cannot convert fullfill payload to msg")
	}
	return o, nil
}

func (c *MayanSwiftContract) DecodeFulfillCall(calldata []byte) (*MayanFulfillMsg, error) {
	fmt.Println("DECODING CALL")
	method, ok := c.ABI.Methods["fulfillOrder"]
	if !ok {
		return nil, fmt.Errorf("no method fulfillOrder")
	}

	params := make(map[string]interface{})
	err := method.Inputs.UnpackIntoMap(params, calldata[4:])
	if err != nil {
		return nil, err
	}

	encodedVM, ok := params["encodedVm"].([]byte)
	if !ok {
		return nil, fmt.Errorf("failed decoding VM data")
	}

	fmt.Printf("%+v \n", encodedVM)

	return c.ParseFulfillPayload(encodedVM)
}

func (c *MayanSwiftContract) ParseFulfillPayload(calldata []byte) (*MayanFulfillMsg, error) {
	res, err := c.CallContract("parseFulfillPayload", calldata)
	if err != nil {
		return nil, err
	}

	fmt.Println("RESULT 0")
	fmt.Printf("%+v", res[0])

	out := abi.ConvertType(res[0], new(MayanFulfillMsg)).(*MayanFulfillMsg)
	return out, nil
}

func convertFloatToUint(amount string, decimals uint8) uint64 {
	floatAmount, _ := strconv.ParseFloat(amount, 64)
	minDecimals := uint8(math.Min(float64(decimals), float64(WORMHOLE_DECIMALS)))

	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(minDecimals)), nil)

	amountBigFloat := new(big.Float).SetFloat64(floatAmount)

	multiplierBigFloat := new(big.Float).SetInt(multiplier)
	scaledAmountBigFloat := new(big.Float).Mul(amountBigFloat, multiplierBigFloat)

	scaledAmountBigInt := new(big.Int)
	scaledAmountBigFloat.Int(scaledAmountBigInt)

	return scaledAmountBigInt.Uint64()
}
