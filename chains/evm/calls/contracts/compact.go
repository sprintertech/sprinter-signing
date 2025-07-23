// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package contracts

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
	"github.com/sygmaprotocol/sygma-core/chains/evm/client"
	"github.com/sygmaprotocol/sygma-core/chains/evm/contracts"
)

type WithdrawalStatus uint8

const (
	STATUS_DISABLED WithdrawalStatus = 0
	STATUS_PENDING  WithdrawalStatus = 1
	STATUS_ENABLED  WithdrawalStatus = 2
	INVALID_STATUS  WithdrawalStatus = 3
)

type CompactContract struct {
	contracts.Contract
	client client.Client
}

func NewCompactContract(
	client client.Client,
	address common.Address,
) *CompactContract {
	return &CompactContract{
		Contract: contracts.NewContract(address, consts.CompactABI, nil, client, nil),
		client:   client,
	}
}

func (c *CompactContract) Allocator(allocatorId *big.Int) (common.Address, error) {
	res, err := c.CallContract("toRegisteredAllocator", allocatorId)
	if err != nil {
		return common.Address{}, err
	}

	out := *abi.ConvertType(res[0], new(common.Address)).(*common.Address)
	return out, nil
}

func (c *CompactContract) GetForcedWithdrawalStatus(account common.Address, id *big.Int) (WithdrawalStatus, error) {
	res, err := c.CallContract("getForcedWithdrawalStatus", account, id)
	if err != nil {
		return INVALID_STATUS, err
	}

	status := *abi.ConvertType(res[0], new(uint8)).(*uint8)
	return WithdrawalStatus(status), nil
}

func (c *CompactContract) HasConsumedAllocatorNonce(allocator common.Address, nonce *big.Int) (bool, error) {
	res, err := c.CallContract("hasConsumedAllocatorNonce", allocator, nonce)
	if err != nil {
		return true, err
	}

	hasConsumedNonce := *abi.ConvertType(res[0], new(bool)).(*bool)
	return hasConsumedNonce, nil
}
