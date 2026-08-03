package price

import (
	"context"
	"fmt"
	"strings"

	"github.com/sprintertech/solver-sdk/pkg/tokenpricing/pyth"
)

type PythPricer struct {
	client *pyth.PythClient
}

func NewPythPricer(ctx context.Context, opts ...pyth.PythClientOption) (*PythPricer, error) {
	client := pyth.NewClient(ctx, opts...)
	err := client.Start(ctx)
	if err != nil {
		return nil, err
	}

	return &PythPricer{
		client: pyth.NewClient(ctx, opts...),
	}, nil
}

func (p *PythPricer) TokenPrice(symbol string) (float64, error) {
	data, err := p.client.PriceUSD(strings.ToUpper(symbol))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch pyth price for %s: %w", symbol, err)
	}
	return data.Value, nil
}
