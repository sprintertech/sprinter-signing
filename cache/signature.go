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
	comm     comm.Communication
}

func NewSignatureCache(c comm.Communication) *SignatureCache {
	cache := ttlcache.New(
		ttlcache.WithTTL[string, []byte](SIGNATURE_TTL),
	)

	sc := &SignatureCache{
		sigCache: cache,
		comm:     c,
	}

	go cache.Start()
	return sc
}

// Subscribe watches for new signatures for given id and returns it through the channel
func (s *SignatureCache) Subscribe(ctx context.Context, id string, sigChannel chan []byte) {
	sig := s.sigCache.Get(id)
	if sig != nil {
		sigChannel <- sig.Value()
		return
	}

	ping := time.Tick(time.Millisecond * 250)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ping:
			{
				sig := s.sigCache.Get(id)
				if sig == nil {
					continue
				}

				sigChannel <- sig.Value()
				return
			}
		}
	}
}

func (s *SignatureCache) Signature(id string) ([]byte, error) {
	sig := s.sigCache.Get(id)
	if sig == nil {
		return []byte{}, fmt.Errorf("no signature found with id %s", id)
	}

	return sig.Value(), nil
}

func (s *SignatureCache) Watch(ctx context.Context, sigChn chan interface{}) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := s.comm.Subscribe(comm.SignatureSessionID, comm.SignatureMsg, msgChn)

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
				s.comm.UnSubscribe(subID)
				return
			}
		}
	}
}
