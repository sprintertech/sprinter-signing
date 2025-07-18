
.PHONY: help run build install license example e2e-test
all: help

export GOLANG_PROTOBUF_REGISTRATION_CONFLICT=ignore

## license: Adds license header to missing files.
license:
	@echo "  >  \033[32mAdding license headers...\033[0m "
	GO111MODULE=off go get -u github.com/google/addlicense
	addlicense -v -c "Sygma" -f ./scripts/header.txt -y 2021 -ignore ".idea/**"  .

## license-check: Checks for missing license headers
license-check:
	@echo "  >  \033[Checking for license headers...\033[0m "
	GO111MODULE=off go get -u github.com/google/addlicense
	addlicense -check -c "Sygma" -f ./scripts/header.txt -y 2021 -ignore ".idea/**" .


coverage:
	go tool cover -func cover.out | grep total | awk '{print $3}'

test:
	./scripts/tests.sh

genmocks:
	mockgen -destination=./tss/ecdsa/common/mock/tss.go github.com/binance-chain/tss-lib/tss Message
	mockgen -destination=./tss/ecdsa/common/mock/communication.go -source=./tss/ecdsa/common/base.go -package mock_tss
	mockgen -destination=./tss/ecdsa/common/mock/fetcher.go -source=./tss/ecdsa/signing/signing.go -package mock_tss
	mockgen --package mock_tss -destination=./tss/mock/ecdsa.go -source=./tss/ecdsa/keygen/keygen.go
	mockgen -source=./tss/coordinator.go -destination=./tss/mock/coordinator.go
	mockgen -source=./comm/communication.go -destination=./comm/mock/communication.go
	mockgen -source=./chains/evm/calls/events/listener.go -destination=./chains/evm/calls/events/mock/listener.go
	mockgen -source=./topology/topology.go -destination=./topology/mock/topology.go
	mockgen -destination=./comm/p2p/mock/host/host.go github.com/libp2p/go-libp2p/core/host Host
	mockgen -destination=./comm/p2p/mock/conn/conn.go github.com/libp2p/go-libp2p/core/network Conn
	mockgen -destination=./comm/p2p/mock/stream/stream.go github.com/libp2p/go-libp2p/core/network Stream,Conn
	mockgen -source=./chains/evm/message/across.go -destination=./chains/evm/message/mock/across.go
	mockgen -source=./chains/evm/message/rhinestone.go -destination=./chains/evm/message/mock/rhinestone.go
	mockgen -source=./chains/evm/message/mayan.go -destination=./chains/evm/message/mock/mayan.go
	mockgen -source=./chains/evm/message/confirmations.go -destination=./chains/evm/message/mock/confirmations.go
	mockgen -source=./api/handlers/signing.go -destination=./api/handlers/mock/signing.go



e2e-test:
	./scripts/e2e_tests.sh

example:
	docker-compose --file=./example/docker-compose.yml up --build

PLATFORMS := linux/amd64 darwin/amd64 darwin/arm64 linux/arm

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -ldflags "-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=ignore" -o 'build/${os}-${arch}/relayer'; \

build-all: $(PLATFORMS)
