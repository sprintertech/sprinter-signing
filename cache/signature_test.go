package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/sprintertech/sprinter-signing/cache"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sprintertech/sprinter-signing/tss/message"
	mock_tss "github.com/sprintertech/sprinter-signing/tss/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type SignatureCacheTestSuite struct {
	suite.Suite

	sc  *cache.SignatureCache
	ctx context.Context

	mockCommunication *mock_communication.MockCommunication
	mockMetrics       *mock_tss.MockMetrics
	cancel            context.CancelFunc
	sigChn            chan interface{}
	msgChn            chan *comm.WrappedMessage
}

func TestRunSignatureCacheTestSuite(t *testing.T) {
	suite.Run(t, new(SignatureCacheTestSuite))
}

func (s *SignatureCacheTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.sigChn = make(chan interface{}, 1)

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockCommunication.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(sessionID string, msgType comm.MessageType, channel chan *comm.WrappedMessage) comm.SubscriptionID {
		s.msgChn = channel
		return comm.NewSubscriptionID("ID", comm.SignatureMsg)
	})
	s.mockCommunication.EXPECT().UnSubscribe(gomock.Any()).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.ctx = ctx

	s.mockMetrics = mock_tss.NewMockMetrics(gomock.NewController(s.T()))
	s.sc = cache.NewSignatureCache(s.mockCommunication, s.mockMetrics)
	go s.sc.Watch(s.ctx, s.sigChn)
	time.Sleep(time.Millisecond * 100)
}
func (s *SignatureCacheTestSuite) TearDownTest() {
	s.cancel()
}

func (s *SignatureCacheTestSuite) Test_Signature_MissingSignature() {
	_, err := s.sc.Signature("invalid")

	s.NotNil(err)
}

func (s *SignatureCacheTestSuite) Test_Signature_ValidSignatureResult() {
	expectedSig := signing.EcdsaSignature{
		Signature: []byte("signature"),
		ID:        "signatureID",
	}
	s.mockMetrics.EXPECT().EndProcess(expectedSig.ID)
	s.sigChn <- expectedSig
	time.Sleep(time.Millisecond * 100)

	sig, err := s.sc.Signature(expectedSig.ID)

	s.Nil(err)
	s.Equal(sig, expectedSig.Signature)
}

func (s *SignatureCacheTestSuite) Test_Signature_ValidMessage() {
	expectedSig := signing.EcdsaSignature{
		Signature: []byte("signature"),
		ID:        "signatureID",
	}
	s.mockMetrics.EXPECT().EndProcess(expectedSig.ID)
	wMsgBytes, _ := message.MarshalSignatureMessage(expectedSig.ID, expectedSig.Signature)
	wMsg := &comm.WrappedMessage{
		Payload: wMsgBytes,
	}

	s.msgChn <- wMsg
	time.Sleep(time.Millisecond * 100)

	sig, err := s.sc.Signature(expectedSig.ID)

	s.Nil(err)
	s.Equal(sig, expectedSig.Signature)
}

func (s *SignatureCacheTestSuite) Test_Subscribe_ValidMessage_EarlyExit() {
	expectedSig := signing.EcdsaSignature{
		Signature: []byte("signature"),
		ID:        "signatureID",
	}
	s.mockMetrics.EXPECT().EndProcess(expectedSig.ID)
	wMsgBytes, _ := message.MarshalSignatureMessage(expectedSig.ID, expectedSig.Signature)
	wMsg := &comm.WrappedMessage{
		Payload: wMsgBytes,
	}

	s.msgChn <- wMsg
	time.Sleep(time.Millisecond * 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChn := make(chan []byte, 1)

	go s.sc.Subscribe(ctx, expectedSig.ID, sigChn)

	sig := <-sigChn
	s.Equal(sig, expectedSig.Signature)
}

func (s *SignatureCacheTestSuite) Test_Subscribe_ValidMessage() {
	expectedSig := signing.EcdsaSignature{
		Signature: []byte("signature"),
		ID:        "signatureID",
	}
	s.mockMetrics.EXPECT().EndProcess(expectedSig.ID)
	wMsgBytes, _ := message.MarshalSignatureMessage(expectedSig.ID, expectedSig.Signature)
	wMsg := &comm.WrappedMessage{
		Payload: wMsgBytes,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChn := make(chan []byte, 1)

	go s.sc.Subscribe(ctx, expectedSig.ID, sigChn)

	time.Sleep(time.Millisecond * 100)
	s.msgChn <- wMsg

	sig := <-sigChn
	s.Equal(sig, expectedSig.Signature)
}
