package price

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

const (
	PRICE_TTL = time.Minute * 60
)

type CoinmarketcapResponse struct {
	Status struct {
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	} `json:"status"`
	Data map[string]struct {
		Quote struct {
			USD struct {
				Price float64 `json:"price"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"data"`
}

type CoinmarketcapAPI struct {
	url    string
	apiKey string
	cache  *ttlcache.Cache[string, float64]
}

func NewCoinmarketcapAPI(url string, apiKey string) *CoinmarketcapAPI {
	cache := ttlcache.New(
		ttlcache.WithTTL[string, float64](PRICE_TTL),
	)
	go cache.Start()

	return &CoinmarketcapAPI{
		url:    url,
		apiKey: apiKey,
		cache:  cache,
	}
}

func (c *CoinmarketcapAPI) TokenPrice(symbol string) (float64, error) {
	price := c.cache.Get(symbol)
	if price != nil {
		return price.Value(), nil
	}

	url := fmt.Sprintf("%s/v1/cryptocurrency/quotes/latest?symbol=%s", c.url, symbol)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accepts", "application/json")
	req.Header.Set("X-CMC_PRO_API_KEY", c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
	}

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var cmcResponse CoinmarketcapResponse
	err = json.Unmarshal(response, &cmcResponse)
	if err != nil {
		return 0, err
	}

	if cmcResponse.Status.ErrorCode != 0 {
		return 0, fmt.Errorf("API Error: %d - %s", cmcResponse.Status.ErrorCode, cmcResponse.Status.ErrorMessage)
	}

	qoutedPrice := cmcResponse.Data[symbol].Quote.USD.Price
	c.cache.Set(symbol, qoutedPrice, ttlcache.DefaultTTL)
	return qoutedPrice, nil
}
