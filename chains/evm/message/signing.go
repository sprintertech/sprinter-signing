package message

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sprintertech/sprinter-signing/chains/evm/signature"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

const SignMessageType = message.MessageType("BorrowSignature")

type SignRequest struct {
	Calldata      []byte
	BorrowAmount  *big.Int
	BorrowToken   string
	Target        string
	Deadline      uint64
	Caller        string
	LiquidityPool string
	Nonce         *big.Int
	SessionID     string
	Coordinator   peer.ID
	ResultChn     chan any
}

func NewSignMessage(source, destination uint64, req *SignRequest) *message.Message {
	return &message.Message{
		Source:      source,
		Destination: destination,
		Data:        req,
		Type:        SignMessageType,
		Timestamp:   time.Now(),
	}
}

type SignHandler struct {
	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher
}

func NewSignHandler(
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
) *SignHandler {
	return &SignHandler{
		coordinator: coordinator,
		host:        host,
		comm:        comm,
		fetcher:     fetcher,
	}
}

func (h *SignHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	req := m.Data.(*SignRequest)

	unlockHash, err := signature.BorrowUnlockHash(
		req.Calldata,
		req.BorrowAmount,
		common.HexToAddress(req.BorrowToken),
		new(big.Int).SetUint64(m.Destination),
		common.HexToAddress(req.Target),
		req.Deadline,
		common.HexToAddress(req.Caller),
		common.HexToAddress(req.LiquidityPool),
		req.Nonce,
	)
	if err != nil {
		return nil, err
	}

	signingProcess, err := signing.NewSigning(
		new(big.Int).SetBytes(unlockHash),
		req.SessionID,
		req.SessionID,
		h.host,
		h.comm,
		h.fetcher)
	if err != nil {
		return nil, err
	}

	return nil, h.coordinator.Execute(context.Background(), []tss.TssProcess{signingProcess}, req.ResultChn, req.Coordinator)
}
