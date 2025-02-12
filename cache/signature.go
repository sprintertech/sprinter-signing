package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sprintertech/sprinter-signing/tss/message"
)

const (
	SIGNATURE_TTL = time.Minute * 10
)

type SignatureCache struct {
	sigCache *ttlcache.Cache[string, []byte]

	comm  comm.Communication
	subID comm.SubscriptionID
}

func NewSignatureCache(ctx context.Context, c comm.Communication, sigChn chan interface{}) *SignatureCache {
	cache := ttlcache.New(
		ttlcache.WithTTL[string, []byte](SIGNATURE_TTL),
	)

	msgChn := make(chan *comm.WrappedMessage)
	subID := c.Subscribe(comm.SignatureSessionID, comm.SignatureMsg, msgChn)

	sc := &SignatureCache{
		sigCache: cache,
		subID:    subID,
		comm:     c,
	}

	go cache.Start()
	go sc.watch(ctx, sigChn, msgChn)
	return sc
}

func (s *SignatureCache) Signature(id string) ([]byte, error) {
	sig := s.sigCache.Get(id)
	if sig == nil {
		return []byte{}, fmt.Errorf("no signature found with id %s", id)
	}

	return sig.Value(), nil
}

func (s *SignatureCache) watch(ctx context.Context, sigChn chan interface{}, msgChn chan *comm.WrappedMessage) {
	for {
		select {
		case sig := <-sigChn:
			{
				sig := sig.(signing.EcdsaSignature)
				s.sigCache.Set(sig.ID, sig.Signature, ttlcache.DefaultTTL)
			}
		case msg := <-msgChn:
			{
				msg, err := message.UnmarshalSignatureMessage(msg.Payload)
				if err != nil {
					log.Warn().Msgf("Failed to unmarshal signature message: %s", err)
					continue
				}

				log.Debug().Msgf("Received signature for ID: %s", msg.ID)
				s.sigCache.Set(msg.ID, msg.Signature, ttlcache.DefaultTTL)
			}
		case <-ctx.Done():
			{
				s.sigCache.Stop()
				s.comm.UnSubscribe(s.subID)
				return
			}
		}
	}
}
