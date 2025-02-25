// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package message_test

import (
	"math/big"
	"testing"

	"github.com/sprintertech/sprinter-signing/tss/message"
	"github.com/stretchr/testify/suite"
)

type TssMessageTestSuite struct {
	suite.Suite
}

func TestRunTssMessageTestSuite(t *testing.T) {
	suite.Run(t, new(TssMessageTestSuite))
}

func (s *TssMessageTestSuite) Test_UnmarshaledMessageShouldBeEqual() {
	originalMsg := &message.TssMessage{
		MsgBytes:    []byte{1},
		IsBroadcast: true,
	}
	msgBytes, err := message.MarshalTssMessage(originalMsg.MsgBytes, originalMsg.IsBroadcast)
	s.Nil(err)

	unmarshaledMsg, err := message.UnmarshalTssMessage(msgBytes)
	s.Nil(err)

	s.Equal(originalMsg, unmarshaledMsg)
}

type StartMessageTestSuite struct {
	suite.Suite
}

func TestRunStartMessageTestSuite(t *testing.T) {
	suite.Run(t, new(StartMessageTestSuite))
}

func (s *StartMessageTestSuite) Test_UnmarshaledMessageShouldBeEqual() {
	originalMsg := &message.StartMessage{
		Params: []byte("test"),
	}
	msgBytes, err := message.MarshalStartMessage(originalMsg.Params)
	s.Nil(err)

	unmarshaledMsg, err := message.UnmarshalStartMessage(msgBytes)
	s.Nil(err)

	s.Equal(originalMsg, unmarshaledMsg)
}

type SignatureMessageTestSuite struct {
	suite.Suite
}

func TestRunSignatureMessageTestSuite(t *testing.T) {
	suite.Run(t, new(SignatureMessageTestSuite))
}

func (s *SignatureMessageTestSuite) Test_UnmarshaledMessageShouldBeEqual() {
	originalMsg := &message.SignatureMessage{
		ID:        "id",
		Signature: []byte("test"),
	}
	msgBytes, err := message.MarshalSignatureMessage(originalMsg.ID, originalMsg.Signature)
	s.Nil(err)

	unmarshaledMsg, err := message.UnmarshalSignatureMessage(msgBytes)
	s.Nil(err)

	s.Equal(originalMsg, unmarshaledMsg)
}

type AcrossMessageTestSuite struct {
	suite.Suite
}

func TestRunAcrossMessageTestSuite(t *testing.T) {
	suite.Run(t, new(AcrossMessageTestSuite))
}

func (s *AcrossMessageTestSuite) Test_UnmarshaledMessageShouldBeEqual() {
	originalMsg := &message.AcrossMessage{
		DepositId:     big.NewInt(100),
		SourceChainId: big.NewInt(101),
		Source:        1,
		Destination:   2,
	}
	msgBytes, err := message.MarshalAcrossMessage(
		originalMsg.DepositId,
		originalMsg.SourceChainId,
		originalMsg.Source,
		originalMsg.Destination)
	s.Nil(err)

	unmarshaledMsg, err := message.UnmarshalAcrossMessage(msgBytes)
	s.Nil(err)

	s.Equal(originalMsg, unmarshaledMsg)
}
