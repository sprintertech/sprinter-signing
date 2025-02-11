package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
)

const (
	SIGNATURE_TTL = time.Minute * 10
)

type SignatureCache struct {
	sigChn   chan interface{}
	sigCache *ttlcache.Cache[string, []byte]
}

func New(ctx context.Context) *SignatureCache {
	cache := ttlcache.New[string, []byte](
		ttlcache.WithTTL[string, []byte](SIGNATURE_TTL),
	)
	sc := &SignatureCache{
		sigCache: cache,
	}

	go cache.Start()
	go sc.watch(ctx)
	return sc
}

func (s *SignatureCache) Signature(id string) ([]byte, error) {
	sig := s.sigCache.Get(id)
	if sig == nil {
		return []byte{}, fmt.Errorf("no signature found with id %s", id)
	}

	return sig.Value(), nil
}

func (s *SignatureCache) watch(ctx context.Context) {
	for {
		select {
		case sig := <-s.sigChn:
			{
				sig := sig.(signing.EcdsaSignature)
				s.sigCache.Set(sig.ID, sig.Signature, ttlcache.DefaultTTL)
			}

		case <-ctx.Done():
			{
				s.sigCache.Stop()
				return
			}
		}
	}
}
