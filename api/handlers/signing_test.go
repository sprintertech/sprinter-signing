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

	"github.com/gorilla/mux"
	"github.com/sprintertech/sprinter-signing/api/handlers"
	mock_handlers "github.com/sprintertech/sprinter-signing/api/handlers/mock"
	across "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/stretchr/testify/suite"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

type SigningHandlerTestSuite struct {
	suite.Suite

	handler *handlers.SigningHandler
	msgChn  chan []*message.Message
}

func TestRunSigningHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SigningHandlerTestSuite))
}

func (s *SigningHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	chains := make(map[uint64]struct{})
	chains[1] = struct{}{}

	s.msgChn = make(chan []*message.Message, 1)
	s.handler = handlers.NewSigningHandler(s.msgChn, chains)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_MissingDepositID() {
	input := handlers.SigningBody{
		Protocol: "across",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-s.msgChn
		ad := msg[0].Data.(across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	s.handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_InvalidChainID() {
	input := handlers.SigningBody{
		DepositId: &handlers.BigInt{big.NewInt(1000)},
		Protocol:  "across",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/invalid/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "invalid",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-s.msgChn
		ad := msg[0].Data.(across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	s.handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_ChainNotSupported() {
	input := handlers.SigningBody{
		DepositId: &handlers.BigInt{big.NewInt(1000)},
		Protocol:  "across",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/111/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "3",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-s.msgChn
		ad := msg[0].Data.(across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	s.handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_InvalidProtocol() {
	input := handlers.SigningBody{
		DepositId: &handlers.BigInt{big.NewInt(1000)},
		Protocol:  "invalid",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-s.msgChn
		ad := msg[0].Data.(across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	s.handler.HandleSigning(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_ErrorHandlingMessage() {
	input := handlers.SigningBody{
		DepositId: &handlers.BigInt{big.NewInt(1000)},
		Protocol:  "across",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-s.msgChn
		ad := msg[0].Data.(across.AcrossData)
		ad.ErrChn <- fmt.Errorf("error handling message")
	}()

	s.handler.HandleSigning(recorder, req)

	s.Equal(http.StatusInternalServerError, recorder.Code)
}

func (s *SigningHandlerTestSuite) Test_HandleSigning_Success() {
	input := handlers.SigningBody{
		DepositId: &handlers.BigInt{big.NewInt(1000)},
		Protocol:  "across",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/signatures", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-s.msgChn
		ad := msg[0].Data.(across.AcrossData)
		ad.ErrChn <- nil
	}()

	s.handler.HandleSigning(recorder, req)

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

func (s *StatusHandlerTestSuite) Test_HandleRequest_MissingDepositID() {
	req := httptest.NewRequest(http.MethodGet, "/v1/chains/1/signatures", nil)
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})

	recorder := httptest.NewRecorder()

	s.handler.HandleRequest(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *StatusHandlerTestSuite) Test_HandleRequest_InvalidChainID() {
	req := httptest.NewRequest(http.MethodGet, "/v1/chains/invalid/signatures", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chainId":   "invalid",
		"depositId": "id",
	})

	recorder := httptest.NewRecorder()

	s.handler.HandleRequest(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *StatusHandlerTestSuite) Test_HandleRequest_ChainNotSupported() {
	req := httptest.NewRequest(http.MethodGet, "/v1/chains/3/signatures", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chainId":   "3",
		"depositId": "id",
	})

	recorder := httptest.NewRecorder()

	s.handler.HandleRequest(recorder, req)

	s.Equal(http.StatusNotFound, recorder.Code)
}

func (s *StatusHandlerTestSuite) Test_HandleRequest_ValidSignature() {
	req := httptest.NewRequest(http.MethodGet, "/v1/chains/1/signatures/id", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chainId":   "1",
		"depositId": "id",
	})

	recorder := httptest.NewRecorder()

	expectedSignature := []byte{0x01, 0x02, 0x03}
	s.mockSignatureCacher.EXPECT().
		Subscribe(gomock.Any(), "1-id", gomock.Any()).
		Do(func(ctx context.Context, id string, sigChannel chan []byte) {
			go func() {
				sigChannel <- expectedSignature
			}()
		})

	go s.handler.HandleRequest(recorder, req)

	time.Sleep(100 * time.Millisecond) // Give some time for the goroutine to execute

	s.Equal(http.StatusOK, recorder.Code)
	s.Equal("text/event-stream", recorder.Header().Get("Content-Type"))
	s.Equal("no-cache", recorder.Header().Get("Cache-Control"))
	s.Equal("keep-alive", recorder.Header().Get("Connection"))
	s.Equal("*", recorder.Header().Get("Access-Control-Allow-Origin"))
	s.Equal("data: "+hex.EncodeToString(expectedSignature)+"\n\n", recorder.Body.String())
}
