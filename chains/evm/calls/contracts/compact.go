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
