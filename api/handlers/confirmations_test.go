package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sprintertech/sprinter-signing/api/handlers"
	"github.com/stretchr/testify/suite"
)

type ConfirmationsHandlerTestSuite struct {
	suite.Suite
}

func TestRunConfirmationsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ConfirmationsHandlerTestSuite))
}

func (s *ConfirmationsHandlerTestSuite) Test_HandleRequest_InvalidChainID() {
	handler := handlers.NewConfirmationsHandler(map[uint64]map[uint64]uint64{})

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/confirmations", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "invalid",
	})

	recorder := httptest.NewRecorder()

	handler.HandleRequest(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *ConfirmationsHandlerTestSuite) Test_HandleRequest_ChainNotFound() {
	handler := handlers.NewConfirmationsHandler(map[uint64]map[uint64]uint64{})

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/confirmations", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})

	recorder := httptest.NewRecorder()

	handler.HandleRequest(recorder, req)

	s.Equal(http.StatusNotFound, recorder.Code)
}

func (s *ConfirmationsHandlerTestSuite) Test_HandleRequest_ValidConfirmations() {
	expectedConfirmations := map[uint64]uint64{
		1000:  0,
		5000:  2,
		10000: 3,
	}

	confirmations := make(map[uint64]map[uint64]uint64)
	confirmations[1] = expectedConfirmations
	handler := handlers.NewConfirmationsHandler(confirmations)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/confirmations", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})

	recorder := httptest.NewRecorder()

	handler.HandleRequest(recorder, req)

	s.Equal(http.StatusOK, recorder.Code)

	data, err := io.ReadAll(recorder.Body)
	s.Nil(err)

	s.Equal(string(data), "{\"1000\":0,\"10000\":3,\"5000\":2}")
}
