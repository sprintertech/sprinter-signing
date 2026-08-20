package tron

import (
	"context"

	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type MessageHandler interface {
	HandleMessage(m *message.Message) (*proposal.Proposal, error)
}

type TronChain struct {
	messageHandler MessageHandler
}

func NewTronChain(messageHandler MessageHandler) *TronChain {
	return &TronChain{messageHandler: messageHandler}
}

func (c *TronChain) PollEvents(_ context.Context) {}

func (c *TronChain) ReceiveMessage(m *message.Message) (*proposal.Proposal, error) {
	return c.messageHandler.HandleMessage(m)
}

func (c *TronChain) Write(_ []*proposal.Proposal) error {
	return nil
}

func (c *TronChain) DomainID() uint64 {
	return ChainID
}
