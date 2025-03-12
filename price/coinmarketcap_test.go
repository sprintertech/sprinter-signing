package price_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprintertech/sprinter-signing/price"
	"github.com/stretchr/testify/suite"
)

type CoinmarketcapAPITestSuite struct {
	suite.Suite
	api        *price.CoinmarketcapAPI
	testServer *httptest.Server
}

func TestRunCoinmarketcapAPITestSuite(t *testing.T) {
	suite.Run(t, new(CoinmarketcapAPITestSuite))
}

func (s *CoinmarketcapAPITestSuite) SetupTest() {
	// Mock server to simulate CoinMarketCap API responses
	s.testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/cryptocurrency/quotes/latest" && r.URL.Query().Get("symbol") == "BTC" {
			w.WriteHeader(http.StatusOK)
			response := price.CoinmarketcapResponse{
				Status: struct {
					ErrorCode    int    `json:"error_code"`
					ErrorMessage string `json:"error_message"`
				}{
					ErrorCode: 0,
				},
				Data: map[string]struct {
					Quote struct {
						USD struct {
							Price float64 `json:"price"`
						} `json:"USD"`
					} `json:"quote"`
				}{
					"BTC": {
						Quote: struct {
							USD struct {
								Price float64 `json:"price"`
							} `json:"USD"`
						}{
							USD: struct {
								Price float64 `json:"price"`
							}{
								Price: 45000.00,
							},
						},
					},
				},
			}
			respBytes, _ := json.Marshal(response)
			w.Write(respBytes)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	}))

	s.api = price.NewCoinmarketcapAPI(s.testServer.URL, "test-api-key")
}

func (s *CoinmarketcapAPITestSuite) TearDownTest() {
	s.testServer.Close()
}

func (s *CoinmarketcapAPITestSuite) TestTokenPrice_Success() {
	price, err := s.api.TokenPrice("BTC")

	s.Nil(err)
	s.Equal(45000.00, price)
}

func (s *CoinmarketcapAPITestSuite) TestTokenPrice_InvalidSymbol() {
	price, err := s.api.TokenPrice("INVALID")
	s.NotNil(err)
	s.Equal(float64(0), price)
}

func (s *CoinmarketcapAPITestSuite) TestTokenPrice_APIError() {
	s.testServer.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"status": {"error_code": 500, "error_message": "Internal Server Error"}}`)
	})

	price, err := s.api.TokenPrice("BTC")
	s.NotNil(err)
	s.Contains(err.Error(), "HTTP request failed with status code 500")
	s.Equal(float64(0), price)
}
