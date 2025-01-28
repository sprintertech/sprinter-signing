// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package events

import (
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
)

// Refresh struct holds key refresh event data
type Refresh struct {
	// SHA1 hash of topology file
	Hash string
}
