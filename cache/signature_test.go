package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sprintertech/sprinter-signing/cache"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sprintertech/sprinter-signing/tss/message"
	"github.com/stretchr/testify/suite"
)

type SignatureCacheTestSuite struct {
	suite.Suite

	sc                *cache.SignatureCache
	mockCommunication *mock_communication.MockCommunication
	cancel            context.CancelFunc
	sigChn            chan interface{}
	msgChn            chan *comm.WrappedMessage
}

func TestRunSignatureCacheTestSuite(t *testing.T) {
	suite.Run(t, new(SignatureCacheTestSuite))
}

func (s *SignatureCacheTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.sigChn = make(chan interface{})

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockCommunication.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(sessionID string, msgType comm.MessageType, channel chan *comm.WrappedMessage) comm.SubscriptionID {
		s.msgChn = channel
		return comm.NewSubscriptionID("ID", comm.SignatureMsg)
	})
	s.mockCommunication.EXPECT().UnSubscribe(gomock.Any()).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.sc = cache.NewSignatureCache(ctx, s.mockCommunication, s.sigChn)
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
