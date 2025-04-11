// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package contracts

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sygmaprotocol/sygma-core/chains/evm/client"
	"github.com/sygmaprotocol/sygma-core/chains/evm/contracts"
)

type HubPoolContract struct {
	contracts.Contract
	client client.Client
	tokens map[string]config.TokenConfig
}

func NewHubPoolContract(
	client client.Client,
	address common.Address,
	l1Tokens map[string]config.TokenConfig,
) *HubPoolContract {
	return &HubPoolContract{
		Contract: contracts.NewContract(address, consts.HubPoolABI, nil, client, nil),
		client:   client,
		tokens:   l1Tokens,
	}
}

func (c *HubPoolContract) DestinationToken(destinationChainId *big.Int, symbol string) (common.Address, error) {
	tokenConfig, ok := c.tokens[symbol]
	if !ok {
		return common.Address{}, fmt.Errorf("no hub pool token configured for symbol %s", symbol)
	}

	res, err := c.CallContract("poolRebalanceRoute", destinationChainId, tokenConfig.Address)
	if err != nil {
		return common.Address{}, err
	}

	out := *abi.ConvertType(res[0], new(common.Address)).(*common.Address)
	if out.Hex() == (common.Address{}).Hex() {
		return common.Address{}, fmt.Errorf("rebalance route not configured for %s", symbol)
	}

	return out, nil
}
