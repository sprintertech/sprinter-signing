package message

import (
	"context"
	"math/big"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	evmMessage "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sprintertech/sprinter-signing/chains/evm/signature"
	"github.com/sprintertech/sprinter-signing/chains/tron"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type Coordinator interface {
	Execute(ctx context.Context, tssProcesses []tss.TssProcess, resultChn chan interface{}, coordinator peer.ID) error
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
	req := m.Data.(*evmMessage.SignRequest)

	borrowToken, err := tron.ToCommonAddress(req.BorrowToken)
	if err != nil {
		return nil, err
	}
	target, err := tron.ToCommonAddress(req.Target)
	if err != nil {
		return nil, err
	}
	caller, err := tron.ToCommonAddress(req.Caller)
	if err != nil {
		return nil, err
	}
	liquidityPool, err := tron.ToCommonAddress(req.LiquidityPool)
	if err != nil {
		return nil, err
	}

	unlockHash, err := signature.BorrowUnlockHash(
		req.Calldata,
		req.BorrowAmount,
		borrowToken,
		new(big.Int).SetUint64(m.Destination),
		target,
		req.Deadline,
		caller,
		liquidityPool,
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
