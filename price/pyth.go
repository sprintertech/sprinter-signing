package price

import (
	"fmt"
	"strings"

	"github.com/sprintertech/solver-sdk/pkg/tokenpricing"
)

type PricerProxy struct {
	pricer tokenpricing.USDPricer
}

func NewPricerProxy(pricer tokenpricing.USDPricer) *PricerProxy {
	return &PricerProxy{
		pricer: pricer,
	}
}

func (p *PricerProxy) TokenPrice(symbol string) (float64, error) {
	data, err := p.pricer.PriceUSD(strings.ToUpper(symbol))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch pyth price for %s: %w", symbol, err)
	}
	return data.Value, nil
}
