package message

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type LighterData struct {
	ErrChn chan error `json:"-"`

	OrderHash     string
	Coordinator   peer.ID
	LiquidityPool common.Address
	DepositTxHash string
	Calldata      string
	Nonce         *big.Int
	Deadline      uint64
	Source        uint64
	Destination   uint64
}

func NewLighterMessage(source, destination uint64, lighterData *LighterData) *message.Message {
	return &message.Message{
		Source:      source,
		Destination: destination,
		Data:        lighterData,
		Type:        message.MessageType(comm.LighterMsg.String()),
		Timestamp:   time.Now(),
	}
}
