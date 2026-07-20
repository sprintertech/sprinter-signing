package handlers_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/sprintertech/sprinter-signing/api/handlers"
	mock_handlers "github.com/sprintertech/sprinter-signing/api/handlers/mock"
	across "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sprintertech/sprinter-signing/chains/evm/signature"
	lighter "github.com/sprintertech/sprinter-signing/chains/lighter/message"
	"github.com/stretchr/testify/suite"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

type SigningHandlerTestSuite struct {
	suite.Suite

	chains map[uint64]struct{}
}

func TestRunSigningHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SigningHandlerTestSuite))
}

func (s *SigningHandlerTestSuite) SetupTest() {
	chains := make(map[uint64]struct{})
	chains[1] = struct{}{}
	s.chains = chains
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_MissingDepositID() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		Protocol:      "across",
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Nonce:         &handlers.BigInt{big.NewInt(1001)},
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_MissingCaller() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		Protocol:      "across",
		DepositId:     "1000",
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Nonce:         &handlers.BigInt{big.NewInt(1001)},
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_MissingLiquidityPool() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		Protocol:  "across",
		DepositId: "1000",
		Caller:    "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Nonce:     &handlers.BigInt{big.NewInt(1001)},
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_InvalidChainID() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		DepositId:     "1000",
		Protocol:      "across",
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Nonce:         &handlers.BigInt{big.NewInt(1001)},
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/invalid/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "invalid",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_ChainNotSupported() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		DepositId:     "1000",
		Protocol:      "across",
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Nonce:         &handlers.BigInt{big.NewInt(1001)},
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/111/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "3",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_InvalidProtocol() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		DepositId:     "1000",
		Protocol:      "invalid",
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Nonce:         &handlers.BigInt{big.NewInt(1001)},
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_ErrorHandlingMessage() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		DepositId:     "1000",
		Protocol:      "across",
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Nonce:         &handlers.BigInt{big.NewInt(1001)},
		BorrowAmount:  &handlers.BigInt{big.NewInt(1000)},
		//nolint:gosec
		Deadline: uint64(time.Now().Unix()),
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusInternalServerError, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_AcrossSuccess() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		DepositId:        "1000",
		Protocol:         "across",
		LiquidityPool:    "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:           "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		BorrowAmount:     &handlers.BigInt{big.NewInt(1000)},
		Nonce:            &handlers.BigInt{big.NewInt(1001)},
		RepaymentChainId: 5,
		//nolint:gosec
		Deadline: uint64(time.Now().Unix()),
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.AcrossData)
		s.Equal(ad.RepaymentChainID, uint64(5))
		ad.ErrChn <- nil
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusAccepted, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_LifiSuccess() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		DepositId:     "depositID",
		Protocol:      "lifi-escrow",
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Calldata:      "0xbe5",
		Nonce:         &handlers.BigInt{big.NewInt(1001)},
		BorrowAmount:  &handlers.BigInt{big.NewInt(1000)},
		//nolint:gosec
		Deadline: uint64(time.Now().Unix()),
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.LifiEscrowData)
		ad.ErrChn <- nil
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusAccepted, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_LighterSuccess() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		DepositId:     "depositID",
		Protocol:      "lighter",
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Calldata:      "0xbe5",
		Nonce:         &handlers.BigInt{big.NewInt(1001)},
		BorrowAmount:  &handlers.BigInt{big.NewInt(1000)},
		//nolint:gosec
		Deadline: uint64(time.Now().Unix()),
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*lighter.LighterData)
		ad.ErrChn <- nil
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusAccepted, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_SprinterSuccess() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewSigningHandler(msgChn, s.chains)

	input := handlers.SigningBody{
		DepositId:     "depositID",
		Protocol:      "sprinter-credit",
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Calldata:      "0xbe5",
		Nonce:         &handlers.BigInt{big.NewInt(1001)},
		BorrowAmount:  &handlers.BigInt{big.NewInt(1000)},
		//nolint:gosec
		Deadline: uint64(time.Now().Unix()),
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.SprinterCreditData)
		ad.ErrChn <- nil
	}()

	handler.HandleSigning(recorder, req)

	s.Equal(http.StatusAccepted, recorder.Code)
}

type StatusHandlerTestSuite struct {
	suite.Suite

	mockSignatureCacher *mock_handlers.MockSignatureCacher
	handler             *handlers.StatusHandler
}

func TestRunStatusHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(StatusHandlerTestSuite))
}

func (s *StatusHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	chains := make(map[uint64]struct{})
	chains[1] = struct{}{}

	s.mockSignatureCacher = mock_handlers.NewMockSignatureCacher(ctrl)
	s.handler = handlers.NewStatusHandler(s.mockSignatureCacher, chains)
}

func (s *StatusHandlerTestSuite) Test_HandleRequest_Errors() {
	tests := []struct {
		name     string
		vars     map[string]string
		query    string
		wantCode int
	}{
		{
			name:     "missing deposit id",
			vars:     map[string]string{"chainId": "1"},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid chain id",
			vars:     map[string]string{"chainId": "invalid", "depositId": "id"},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "chain not supported",
			vars:     map[string]string{"chainId": "3", "depositId": "id"},
			wantCode: http.StatusNotFound,
		},
		{
			// A partial composite param set must fail fast, not fall back to the legacy key and hang.
			name:     "partial composite params",
			vars:     map[string]string{"chainId": "1", "depositId": "id"},
			query:    "?deadline=1000&caller=0x1111111111111111111111111111111111111111",
			wantCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/v1/chains/1/signatures/id"+tt.query, nil)
			req = mux.SetURLVars(req, tt.vars)
			recorder := httptest.NewRecorder()

			s.handler.HandleRequest(recorder, req)

			s.Equal(tt.wantCode, recorder.Code)
		})
	}
}

func (s *StatusHandlerTestSuite) Test_HandleRequest_SubscribeKey() {
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	pool := common.HexToAddress("0x3333333333333333333333333333333333333333")
	lowerCaller := common.HexToAddress("0xabcdef0123456789abcdef0123456789abcdef01")
	lowerPool := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	tests := []struct {
		name      string
		depositId string
		query     string
		wantKey   string
	}{
		{
			name:      "legacy key without params",
			depositId: "id",
			wantKey:   "1-id",
		},
		{
			name:      "composite key from params",
			depositId: "id",
			query:     "?deadline=1000&caller=" + caller.Hex() + "&borrowAmount=500&liquidityPool=" + pool.Hex() + "&repaymentChainId=10",
			wantKey:   signature.BorrowSessionID(1, "id", 1000, caller, big.NewInt(500), pool, 10),
		},
		{
			// Lowercase addresses and a leading-zero deposit id must still map to the canonical publish key.
			name:      "composite key normalizes address casing and deposit id",
			depositId: "0042",
			query:     "?deadline=1000&caller=0xabcdef0123456789abcdef0123456789abcdef01&borrowAmount=500&liquidityPool=0x1234567890abcdef1234567890abcdef12345678&repaymentChainId=10",
			wantKey:   signature.BorrowSessionID(1, "42", 1000, lowerCaller, big.NewInt(500), lowerPool, 10),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/v1/chains/1/signatures/"+tt.depositId+tt.query, nil)
			req = mux.SetURLVars(req, map[string]string{"chainId": "1", "depositId": tt.depositId})
			recorder := httptest.NewRecorder()

			expectedSignature := []byte{0x01, 0x02, 0x03}
			s.mockSignatureCacher.EXPECT().
				Subscribe(gomock.Any(), tt.wantKey, gomock.Any()).
				Do(func(ctx context.Context, id string, sigChannel chan []byte) {
					go func() {
						sigChannel <- expectedSignature
					}()
				})

			go s.handler.HandleRequest(recorder, req)

			time.Sleep(50 * time.Millisecond)

			s.Equal(http.StatusOK, recorder.Code)
			s.Equal("text/event-stream", recorder.Header().Get("Content-Type"))
			s.Equal("no-cache", recorder.Header().Get("Cache-Control"))
			s.Equal("keep-alive", recorder.Header().Get("Connection"))
			s.Equal("*", recorder.Header().Get("Access-Control-Allow-Origin"))
			s.Equal("data: "+hex.EncodeToString(expectedSignature)+"\n\n", recorder.Body.String())
		})
	}
}
