// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package message

import (
	"encoding/json"
	"math/big"
)

type TssMessage struct {
	MsgBytes    []byte `json:"msgBytes"`
	IsBroadcast bool   `json:"isBroadcast"`
}

func MarshalTssMessage(msgBytes []byte, isBroadcast bool) ([]byte, error) {
	tssMsg := &TssMessage{
		IsBroadcast: isBroadcast,
		MsgBytes:    msgBytes,
	}

	msgBytes, err := json.Marshal(tssMsg)
	if err != nil {
		return []byte{}, err
	}

	return msgBytes, nil
}

func UnmarshalTssMessage(msgBytes []byte) (*TssMessage, error) {
	msg := &TssMessage{}
	err := json.Unmarshal(msgBytes, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

type StartMessage struct {
	Params []byte `json:"params"`
}

func MarshalStartMessage(params []byte) ([]byte, error) {
	startSignMessage := &StartMessage{
		Params: params,
	}

	msgBytes, err := json.Marshal(startSignMessage)
	if err != nil {
		return []byte{}, err
	}

	return msgBytes, nil
}

func UnmarshalStartMessage(msgBytes []byte) (*StartMessage, error) {
	msg := &StartMessage{}
	err := json.Unmarshal(msgBytes, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

type SignatureMessage struct {
	Signature []byte `json:"signature"`
	ID        string `json:"id"`
}

func MarshalSignatureMessage(id string, signature []byte) ([]byte, error) {
	signatureMessage := &SignatureMessage{
		Signature: signature,
		ID:        id,
	}

	msgBytes, err := json.Marshal(signatureMessage)
	if err != nil {
		return []byte{}, err
	}

	return msgBytes, nil
}

func UnmarshalSignatureMessage(msgBytes []byte) (*SignatureMessage, error) {
	msg := &SignatureMessage{}
	err := json.Unmarshal(msgBytes, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

type AcrossMessage struct {
	DepositId   *big.Int `json:"depositId"`
	Source      uint64   `json:"source"`
	Destination uint64   `json:"destination"`
}

func MarshalAcrossMessage(depositId *big.Int, source, destination uint64) ([]byte, error) {
	signatureMessage := &AcrossMessage{
		DepositId:   depositId,
		Source:      source,
		Destination: destination,
	}

	msgBytes, err := json.Marshal(signatureMessage)
	if err != nil {
		return []byte{}, err
	}

	return msgBytes, nil
}

func UnmarshalAcrossMessage(msgBytes []byte) (*AcrossMessage, error) {
	msg := &AcrossMessage{}
	err := json.Unmarshal(msgBytes, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
