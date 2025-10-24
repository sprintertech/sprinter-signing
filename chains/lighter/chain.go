package lighter

import (
	"context"

	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

const (
	LIGHTER_DOMAIN_ID uint64 = 1513889025
)

type MessageHandler interface {
	HandleMessage(m *message.Message) (*proposal.Proposal, error)
}

type LighterChain struct {
	messageHandler MessageHandler
	domainID       uint64
}

func NewLighterChain(messageHandler MessageHandler) *LighterChain {
	return &LighterChain{
		messageHandler: messageHandler,
		domainID:       LIGHTER_DOMAIN_ID,
	}
}

func (c *LighterChain) PollEvents(_ context.Context) {}

func (c *LighterChain) ReceiveMessage(m *message.Message) (*proposal.Proposal, error) {
	return c.messageHandler.HandleMessage(m)
}

func (c *LighterChain) Write(_ []*proposal.Proposal) error {
	return nil
}

func (c *LighterChain) DomainID() uint64 {
	return c.domainID
}
