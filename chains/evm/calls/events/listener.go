// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package events

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"

	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
)

type ChainClient interface {
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error)
	WaitAndReturnTxReceipt(h common.Hash) (*ethTypes.Receipt, error)
	LatestBlock() (*big.Int, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*ethTypes.Block, error)
}

type Listener struct {
	client   ChainClient
	abi      abi.ABI
	retryAbi abi.ABI
}

func NewListener(client ChainClient) *Listener {
	retryAbi, _ := abi.JSON(strings.NewReader(consts.RetryABI))
	abi, _ := abi.JSON(strings.NewReader(consts.BridgeABI))
	return &Listener{
		client:   client,
		abi:      abi,
		retryAbi: retryAbi,
	}
}

func (l *Listener) FetchKeygenEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]ethTypes.Log, error) {
	logs, err := l.client.FetchEventLogs(ctx, contractAddress, string(StartKeygenSig), startBlock, endBlock)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (l *Listener) FetchRefreshEvents(ctx context.Context, contractAddress common.Address, startBlock *big.Int, endBlock *big.Int) ([]*Refresh, error) {
	logs, err := l.client.FetchEventLogs(ctx, contractAddress, string(KeyRefreshSig), startBlock, endBlock)
	if err != nil {
		return nil, err
	}
	refreshEvents := make([]*Refresh, 0)

	for _, re := range logs {
		r, err := l.UnpackRefresh(l.abi, re.Data)
		if err != nil {
			log.Err(err).Msgf("failed unpacking refresh event log")
			continue
		}

		refreshEvents = append(refreshEvents, r)
	}

	return refreshEvents, nil
}

func (l *Listener) UnpackRefresh(abi abi.ABI, data []byte) (*Refresh, error) {
	var rl Refresh

	err := abi.UnpackIntoInterface(&rl, "KeyRefresh", data)
	if err != nil {
		return &Refresh{}, err
	}

	return &rl, nil
}
