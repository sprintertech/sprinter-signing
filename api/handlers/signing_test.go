package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sprintertech/sprinter-signing/api/handlers"
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
		DepositId: big.NewInt(1000),
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
		DepositId: big.NewInt(1000),
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
		DepositId: big.NewInt(1000),
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
		DepositId: big.NewInt(1000),
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
		DepositId: big.NewInt(1000),
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
