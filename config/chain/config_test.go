// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package chain

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	var id uint64 = 1
	valid := GeneralChainConfig{
		Name:     "chain",
		Id:       &id,
		Endpoint: "endpoint",
	}

	missingEndpoint := GeneralChainConfig{
		Name:     "chain",
		Id:       &id,
		Endpoint: "",
	}

	missingName := GeneralChainConfig{
		Name:     "",
		Id:       &id,
		Endpoint: "endpoint",
	}

	missingId := GeneralChainConfig{
		Name:     "chain",
		Endpoint: "endpoint",
	}

	err := valid.Validate()
	if err != nil {
		t.Fatal(err)
	}

	err = missingEndpoint.Validate()
	if err == nil {
		t.Fatalf("must require endpoint field, %v", err)
	}

	err = missingName.Validate()
	if err == nil {
		t.Fatal("must require name field")
	}

	err = missingId.Validate()
	if err == nil {
		t.Fatalf("must require domain id field, %v", err)
	}
}
