package message

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/contracts"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/protocol/rhinestone"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type BundleFetcher interface {
	GetBundle(bundleID string) (*rhinestone.Bundle, error)
}

type RhinestoneMessageHandler struct {
	chainID uint64

	bundleFetcher      BundleFetcher
	tokenStore         config.TokenStore
	rhinestoneContract contracts.RhinestoneContract

	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan any
}

func NewRhinestoneMessageHandler(
	chainID uint64,
	coordinator Coordinator,
	host host.Host,
	comm comm.Communication,
	fetcher signing.SaveDataFetcher,
	tokenStore config.TokenStore,
	rhinestoneContract contracts.RhinestoneContract,
	bundleFetcher BundleFetcher,
	sigChn chan any,
) *RhinestoneMessageHandler {
	return &RhinestoneMessageHandler{
		chainID:            chainID,
		coordinator:        coordinator,
		host:               host,
		comm:               comm,
		fetcher:            fetcher,
		sigChn:             sigChn,
		rhinestoneContract: rhinestoneContract,
		bundleFetcher:      bundleFetcher,
		tokenStore:         tokenStore,
	}
}

// HandleMessage verifies the bundle data and signs the unlock hash to use liquidity
// for the Rhinestone protocol
func (h *RhinestoneMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*RhinestoneData)
	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	bundle, err := h.bundleFetcher.GetBundle(data.BundleID)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	calldata, err := hex.DecodeString(bundle.BundleEvent.FillPayload.Data[2:])
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	deadline, err := strconv.ParseUint(bundle.BundleData.Expires, 10, 64)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	err = h.verifyOrder(bundle, data, calldata)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	borrowToken, err := h.outputToken(
		bundle.BundleEvent.AcrossDepositEvents[0].OriginClaimPayload.ChainID,
		bundle.TargetChainId,
		common.HexToAddress(bundle.BundleEvent.AcrossDepositEvents[0].InputToken),
	)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}
	data.ErrChn <- nil

	unlockHash, err := unlockHash(
		calldata,
		data.BorrowAmount,
		borrowToken,
		new(big.Int).SetUint64(bundle.TargetChainId),
		common.HexToAddress(bundle.BundleEvent.FillPayload.To),
		deadline,
		data.Caller,
		data.LiquidityPool,
		data.Nonce,
	)
	if err != nil {
		return nil, err
	}

	sessionID := fmt.Sprintf("%d-%s", bundle.TargetChainId, bundle.BundleEvent.BundleId)
	signing, err := signing.NewSigning(
		new(big.Int).SetBytes(unlockHash),
		sessionID,
		sessionID,
		h.host,
		h.comm,
		h.fetcher)
	if err != nil {
		return nil, err
	}

	err = h.coordinator.Execute(context.Background(), []tss.TssProcess{signing}, h.sigChn, data.Coordinator)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// outputToken fetches the matching output token to the
// input token of the across deposit
func (h *RhinestoneMessageHandler) outputToken(
	srcChainID uint64,
	dstChainID uint64,
	token common.Address,
) (common.Address, error) {
	symbol, _, err := h.tokenStore.ConfigByAddress(srcChainID, token)
	if err != nil {
		return common.Address{}, err
	}

	cfg, err := h.tokenStore.ConfigBySymbol(dstChainID, symbol)
	if err != nil {
		return common.Address{}, err
	}

	return cfg.Address, nil
}

// verifyOrder checks that the order is an existing valid order on-chain
// and the fill calldata is corresponds to the expected stash fill
func (h *RhinestoneMessageHandler) verifyOrder(
	bundle *rhinestone.Bundle,
	data *RhinestoneData,
	calldata []byte,
) error {
	if bundle.Status == rhinestone.StatusCompleted {
		return fmt.Errorf("invalid order status %s", bundle.Status)
	}

	inputToken := bundle.BundleEvent.AcrossDepositEvents[0].InputToken
	outputToken := bundle.BundleEvent.AcrossDepositEvents[0].OutputToken
	inputAmount := big.NewInt(0)
	for _, d := range bundle.BundleEvent.AcrossDepositEvents {
		if d.InputToken != inputToken || d.OutputToken != outputToken {
			return fmt.Errorf("order has different tokens in the bundle")
		}

		bigInputAmount, _ := new(big.Int).SetString(d.InputAmount, 10)
		inputAmount.Add(inputAmount, bigInputAmount)
	}

	if data.BorrowAmount.Cmp(inputAmount) == 1 {
		return fmt.Errorf(
			"requested borrow amount %s larger than input amount %s",
			data.BorrowAmount, inputAmount)
	}

	fillInput, err := h.rhinestoneContract.DecodeFillCall(calldata)
	if err != nil {
		return err
	}

	for _, address := range fillInput.RepaymentAddresses {
		if address != data.LiquidityPool {
			return fmt.Errorf(
				"repayment address %s different from liquidity pool address %s",
				address,
				data.LiquidityPool,
			)
		}
	}
	for _, chainID := range fillInput.RepaymentChainIds {
		if chainID.Uint64() != h.chainID {
			return fmt.Errorf(
				"repayment chainID %d different than expected chainID %d",
				chainID.Uint64(),
				h.chainID,
			)
		}
	}

	return nil
}

func (h *RhinestoneMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(comm.RhinestoneSessionID, comm.RhinestoneMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				d := &RhinestoneData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling rhinestone message: %s", err)
					continue
				}

				d.ErrChn = make(chan error, 1)
				msg := NewRhinestoneMessage(d.Source, d.Destination, d)
				_, err = h.HandleMessage(msg)
				if err != nil {
					log.Err(err).Msgf("Failed handling rhinestone message %+v because of: %s", msg, err)
				}
			}
		case <-ctx.Done():
			{
				h.comm.UnSubscribe(subID)
				return
			}
		}
	}
}

func (h *RhinestoneMessageHandler) notify(data *RhinestoneData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.RhinestoneMsg, fmt.Sprintf("%d-%s", h.chainID, comm.RhinestoneSessionID))
}
