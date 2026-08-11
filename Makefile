
.PHONY: help run build install license example e2e-test \
        build-ffi build-ffi-debug clean-ffi
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
	mockgen -source=./chains/evm/message/lifiEscrow.go -destination=./chains/evm/message/mock/lifiEscrow.go
	mockgen -source=./chains/evm/message/unlock.go -destination=./chains/evm/message/mock/unlock.go
	mockgen -source=./chains/evm/message/confirmations.go -destination=./chains/evm/message/mock/confirmations.go
	mockgen -source=./api/handlers/signing.go -destination=./api/handlers/mock/signing.go
	mockgen -package mock_message -destination=./chains/evm/message/mock/pricing.go github.com/sprintertech/solver-sdk/pkg/pricing OrderPricer
	mockgen -source=./chains/lighter/message/lighter.go -destination=./chains/lighter/message/mock/lighter.go
	mockgen -source=./protocol/lifi/event.go -destination=./protocol/lifi/mock/event.go
	mockgen -source=./protocol/across/deposit.go -destination=./protocol/across/mock/deposit.go



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

# ── Rust FFI (cggmp21) ────────────────────────────────────────────────────────
# Builds the cggmp21-ffi Rust crate as a static library for CGo consumption.
# Targets linux/amd64 only (we build inside the Docker image).
# Outputs:  rust/lib/libcggmp21_ffi.a  and  rust/include/cggmp21.h
FFI_MANIFEST    := rust/Cargo.toml
FFI_CRATE       := cggmp21-ffi
FFI_OUT_LIB     := rust/lib
FFI_OUT_INCLUDE := rust/include
FFI_HEADER_SRC  := rust/cggmp21-ffi/include/cggmp21.h

## build-ffi: Build cggmp21-ffi as a release static library and stage it for CGo.
build-ffi:
	cargo build --release --manifest-path $(FFI_MANIFEST) -p $(FFI_CRATE)
	mkdir -p $(FFI_OUT_LIB) $(FFI_OUT_INCLUDE)
	cp rust/target/release/libcggmp21_ffi.a $(FFI_OUT_LIB)/
	cp $(FFI_HEADER_SRC) $(FFI_OUT_INCLUDE)/

## build-ffi-debug: Same as build-ffi but unoptimised (faster to build).
build-ffi-debug:
	cargo build --manifest-path $(FFI_MANIFEST) -p $(FFI_CRATE)
	mkdir -p $(FFI_OUT_LIB) $(FFI_OUT_INCLUDE)
	cp rust/target/debug/libcggmp21_ffi.a $(FFI_OUT_LIB)/
	cp $(FFI_HEADER_SRC) $(FFI_OUT_INCLUDE)/

## clean-ffi: Remove staged FFI artifacts and the Rust target directory.
clean-ffi:
	cargo clean --manifest-path $(FFI_MANIFEST) -p $(FFI_CRATE)
	rm -rf $(FFI_OUT_LIB) $(FFI_OUT_INCLUDE)
