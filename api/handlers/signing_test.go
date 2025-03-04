package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

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
		ChainId: 1,
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/signing", bytes.NewReader(body))
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

func (s *SigningHandlerTestSuite) Test_HandleSigning_MissingChainID() {
	input := handlers.SigningBody{
		DepositId: big.NewInt(1000),
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/signing", bytes.NewReader(body))
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
		ChainId:   2,
		DepositId: big.NewInt(1000),
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/signing", bytes.NewReader(body))
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
		ChainId:   1,
		DepositId: big.NewInt(1000),
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/signing", bytes.NewReader(body))
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
		ChainId:   1,
		DepositId: big.NewInt(1000),
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/signing", bytes.NewReader(body))
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
