package price

import (
	"fmt"
	"strings"

	"github.com/sprintertech/solver-sdk/pkg/tokenpricing/pyth"
)

type PythPricer struct {
	client *pyth.PythClient
}

func NewPythPricer(client *pyth.PythClient) *PythPricer {
	return &PythPricer{
		client: client,
	}
}

func (p *PythPricer) TokenPrice(symbol string) (float64, error) {
	data, err := p.client.PriceUSD(strings.ToUpper(symbol))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch pyth price for %s: %w", symbol, err)
	}
	return data.Value, nil
}
