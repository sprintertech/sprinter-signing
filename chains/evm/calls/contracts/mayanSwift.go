// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package contracts

import (
	"encoding/binary"
	"fmt"
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
	amountOut, err := strconv.ParseUint(swap.MinAmountOut64, 10, 64)
	if err != nil {
		return nil, err
	}
	key := &MayanKey{
		Trader:       common.HexToHash(swap.Trader),
		SrcChainId:   msg.SrcChainId,
		TokenIn:      msg.TokenIn,
		DestAddr:     msg.DestAddr,
		DestChainId:  msg.DestChainId,
		TokenOut:     msg.TokenOut,
		MinAmountOut: amountOut,
		GasDrop:      msg.GasDrop,
		CancelFee:    convertFloatToUint(swap.RedeemRelayerFee, srcTokenDecimals),
		RefundFee:    convertFloatToUint(swap.RefundRelayerFee, srcTokenDecimals),
		Deadline:     msg.Deadline,
		ReferrerAddr: msg.ReferrerAddr,
		ReferrerBps:  msg.ReferrerBps,
		ProtocolBps:  swap.MayanBps,
		AuctionMode:  swap.AuctionMode,
		Random:       common.HexToHash(swap.RandomKey),
	}

	res, err := c.CallContract("orders", common.BytesToHash(crypto.Keccak256(encodeKey(key))))
	if err != nil {
		return nil, err
	}

	status := abi.ConvertType(res[0], new(uint8)).(*uint8)
	amountIn := abi.ConvertType(res[1], new(uint64)).(*uint64)
	destChainId := abi.ConvertType(res[2], new(uint16)).(*uint16)
	return &MayanOrder{
		Status:      OrderStatus(*status),
		AmountIn:    *amountIn,
		DestChainId: *destChainId,
	}, nil
}

func (c *MayanSwiftContract) DecodeFulfillCall(calldata []byte) (*MayanFulfillMsg, error) {
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

	return c.ParseFulfillPayload(extractWormholeVMPayload(encodedVM))
}

func (c *MayanSwiftContract) ParseFulfillPayload(calldata []byte) (*MayanFulfillMsg, error) {
	res, err := c.CallContract("parseFulfillPayload", calldata)
	if err != nil {
		return nil, err
	}

	out := abi.ConvertType(res[0], new(MayanFulfillMsg)).(*MayanFulfillMsg)
	return out, nil
}

func convertFloatToUint(amount string, decimals uint8) uint64 {
	f, _, _ := big.ParseFloat(amount, 10, 256, big.ToNearestEven)

	minDecimals := min(WORMHOLE_DECIMALS, decimals)
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(minDecimals)), nil)

	scaled := new(big.Float).Mul(f, new(big.Float).SetInt(multiplier))
	scaled.Add(scaled, big.NewFloat(0.5)) // Rounding adjustment

	result := new(big.Int)
	scaled.Int(result)
	return result.Uint64()
}

func extractWormholeVMPayload(encodedVM []byte) []byte {
	signersLen := int(encodedVM[5])            // Read signature count from byte 5
	payloadStart := 6 + (signersLen * 66) + 51 // Calculate payload offset
	return encodedVM[payloadStart:]
}

// encodeKey encodes mayan key into the order hash expected on-chain
func encodeKey(key *MayanKey) []byte {
	data := make([]byte, 239)
	offset := 0

	copy(data[offset:], key.Trader[:]) // 0-31 (32 bytes)
	offset += 32

	binary.BigEndian.PutUint16(data[offset:], key.SrcChainId) // 32-33 (2 bytes)
	offset += 2

	copy(data[offset:], key.TokenIn[:]) // 34-65 (32 bytes)
	offset += 32

	copy(data[offset:], key.DestAddr[:]) // 66-97 (32 bytes)
	offset += 32

	binary.BigEndian.PutUint16(data[offset:], key.DestChainId) // 98-99 (2 bytes)
	offset += 2

	copy(data[offset:], key.TokenOut[:]) // 100-131 (32 bytes)
	offset += 32

	// uint64 sequence (40 bytes total)
	binary.BigEndian.PutUint64(data[offset:], key.MinAmountOut) // 132-139
	offset += 8
	binary.BigEndian.PutUint64(data[offset:], key.GasDrop) // 140-147
	offset += 8
	binary.BigEndian.PutUint64(data[offset:], key.CancelFee) // 148-155
	offset += 8
	binary.BigEndian.PutUint64(data[offset:], key.RefundFee) // 156-163
	offset += 8
	binary.BigEndian.PutUint64(data[offset:], key.Deadline) // 164-171
	offset += 8

	copy(data[offset:], key.ReferrerAddr[:]) // 172-203 (32 bytes)
	offset += 32

	data[offset] = key.ReferrerBps // 204 (1 byte)
	offset += 1

	// Final group (protocolBps + auctionMode + random)
	data[offset] = key.ProtocolBps // 205 (1 byte)
	offset += 1
	data[offset] = key.AuctionMode // 206 (1 byte)
	offset += 1
	copy(data[offset:], key.Random[:]) // 207-238 (32 bytes)

	return data
}
