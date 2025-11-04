package handlers_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sprintertech/sprinter-signing/api/handlers"
	across "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/stretchr/testify/suite"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
)

type UnlockHandlerTestSuite struct {
	suite.Suite

	chains map[uint64]struct{}
}

func TestRunUnlockHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UnlockHandlerTestSuite))
}

func (s *UnlockHandlerTestSuite) SetupTest() {
	chains := make(map[uint64]struct{})
	chains[1] = struct{}{}
	s.chains = chains
}

func (s *UnlockHandlerTestSuite) Test_HandleUnlock_InvalidRequest() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewUnlockHandler(msgChn, s.chains)

	input := handlers.UnlockBody{
		Protocol: "lifi-escrow",
		OrderID:  "id",
		Settler:  "settler",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/unlocks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.LifiUnlockData)
		ad.SigChn <- signing.EcdsaSignature{}
	}()

	handler.HandleUnlock(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *UnlockHandlerTestSuite) Test_HandleUnlock_InvalidProtocol() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewUnlockHandler(msgChn, s.chains)

	input := handlers.UnlockBody{
		Protocol: "across",
		OrderID:  "id",
		Settler:  "settler",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/unlocks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})

	recorder := httptest.NewRecorder()

	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.LifiUnlockData)
		ad.SigChn <- signing.EcdsaSignature{}
	}()

	handler.HandleUnlock(recorder, req)

	s.Equal(http.StatusBadRequest, recorder.Code)
}

func (s *UnlockHandlerTestSuite) Test_HandleUnlock_ValidRequest() {
	msgChn := make(chan []*message.Message)
	handler := handlers.NewUnlockHandler(msgChn, s.chains)

	input := handlers.UnlockBody{
		Protocol: "lifi-escrow",
		OrderID:  "id",
		Settler:  "settler",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/chains/1/unlocks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"chainId": "1",
	})

	recorder := httptest.NewRecorder()

	sigBytes, _ := hex.DecodeString("abcd")
	go func() {
		msg := <-msgChn
		ad := msg[0].Data.(*across.LifiUnlockData)
		ad.SigChn <- signing.EcdsaSignature{
			Signature: sigBytes,
			ID:        "id",
		}
	}()

	handler.HandleUnlock(recorder, req)

	s.Equal(http.StatusOK, recorder.Code)
	data, err := io.ReadAll(recorder.Body)
	s.Nil(err)

	s.Equal(string(data), "{\"signature\":\"abcd\",\"id\":\"id\"}")
}
