package message_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/contracts"
	"github.com/sprintertech/sprinter-signing/chains/evm/message"
	mock_message "github.com/sprintertech/sprinter-signing/chains/evm/message/mock"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/keyshare"
	"github.com/sprintertech/sprinter-signing/protocol/mayan"
	mock_tss "github.com/sprintertech/sprinter-signing/tss/ecdsa/common/mock"
	"github.com/stretchr/testify/suite"
	coreMessage "github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

var (
	mayanCalldata = "488c35910000000000000000000000000000000000000000000000000022a234ce522eb100000000000000000000000000000000000000000000000000000000000000800000000000000000000000006ffc5848c46319e7c6d48f56ca2152b213d4535f0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000047201000000040d000881698215359f2238fc4a826edb226cdb16341fd05312ff489d7b1a6802532b48442301a935850009067ca2cceef73ba79e6c99ec42e3c458fb7549be2dcc9d00025f7076ea20ea888ceb6ec245583131597cf53eb9ebb501b6e78d8c0237afe60a45dbd6264085f06c66ff4c507ce80f5594065c6580c557d57035cc19acd34cdd0103c667ce54fbd7aa89a4c6e6d96b6d96a67f14294119deba644b719ca7bde5ba617f97feefb528549eed7a11ddfed160ce3f4b9f2c1448e27d92af2d42edc0520e0004f9cf8fb86762d87f26d46e13489bf36c4441b416dfe05779f57ab0356021d67505cffd6decd0c2b3903d9a7d2ed92b9c5cbf276c9310176fd9e1d8c0bd5e5ffe00069ca2f7e36c13d58c700e8a33d18bbec53070072b8d60e8dac41f24615caa9bec7491f889a1fa7bc9b2b387a44597d149576b5ca3f2fb4f44afe73afd32744a6d0107f70c20d919f0c3dabec14adad9cad9c3b9a0f74fd71296509f733b232c31d76b6e297068214b8b68ef30b499d1d7739641e849741f4b16000b714c901708e666000835370a611742837d40c0d7fbb2d90bf3809d664ec86972d9e5a78b647facb8113d5c6f649cb482466e04f6d0b4036ce5eed0e0f42f38330f288985d0893dd36b010aeec9bbe877140e75fbfdc6bdd2d1ef2736d4ae01811f30dceebf7eb440cb1ec17d2dfa894ba4c8a6dbccf91da5182de86cf2142a049de515b666a051853ed858000b89963305c356b251c5d4402ec7af7adc182d385d5327dd0c43f344f7593382b4083998f702371dca1b507d83f254b12b2fbe4c4cb3d6a905a7c7f2acbd3d5f3c010d4c3655823a39abaf93a7cc0c4a73c67e63b79535452b2dd8d6e27713e48326ed7b4b28fbb690595d0545c6a2f3e42165520eba0311d24ffa1f6bfeb67b14428b000f5689a1d5550361c616b03b875e02d9de688047a0cb59e6da104fe3589712232c7384a1a653e28a22dffc8d484fd1c40b133622d99a9be586c928eb9d681e976f01106a96bd0c9f66811485a6cc017ddb31a4e38335479b9f75fda3259275b640698b4a308ca3d93df6f4cbb5f62ee3b3e815fb74267e10d1cae23372c92a250a697d0011ffdeb06d9dc35e134b5e40c9cc4db2cb1d06b46ce5b4717c6355fb9319f0ee275bcaa57246984aa694c717583d4b4d35f198f1c105025376275bceab861163780167f7c47900000000000134cdc6b2623f36d60ae820e95b60f764e81ec2cd3b57b77e3f8e25ddd43ac37300000000000e158d01015ab756fa13dcf3b79c9d4636c5d241018ee1a797582e979b095872e13b01e2e000180000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f24f2160676ae01f5a8754e544e132298868a367001e000000000000000000000000000000000000000000000000000000000000000000000000000eded400000000000000000000000067f7c90c000000000000000000000000a5aa6e2171b416e1d27ec53ca8c13db3f91a89cd00030000000000000000000000006ffc5848c46319e7c6d48f56ca2152b213d4535f0000000000000000000000000000"
)

type MayanMessageHandlerTestSuite struct {
	suite.Suite

	mockCommunication *mock_communication.MockCommunication
	mockCoordinator   *mock_message.MockCoordinator
	mockEventFilterer *mock_message.MockEventFilterer
	mockHost          *mock_host.MockHost
	mockFetcher       *mock_tss.MockSaveDataFetcher
	mockWatcher       *mock_message.MockConfirmationWatcher
	mockSwapFetcher   *mock_message.MockSwapFetcher
	mockContract      *mock_message.MockMayanContract

	handler *message.MayanMessageHandler
	sigChn  chan interface{}

	validLog []byte
}

func TestRunMayanMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(MayanMessageHandlerTestSuite))
}

func (s *MayanMessageHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockCoordinator = mock_message.NewMockCoordinator(ctrl)
	s.mockEventFilterer = mock_message.NewMockEventFilterer(ctrl)

	s.mockHost = mock_host.NewMockHost(ctrl)
	s.mockHost.EXPECT().ID().Return(peer.ID("")).AnyTimes()

	s.mockFetcher = mock_tss.NewMockSaveDataFetcher(ctrl)
	s.mockFetcher.EXPECT().UnlockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().LockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().GetKeyshare().AnyTimes().Return(keyshare.ECDSAKeyshare{}, nil)

	s.mockWatcher = mock_message.NewMockConfirmationWatcher(ctrl)
	s.mockContract = mock_message.NewMockMayanContract(ctrl)
	s.mockSwapFetcher = mock_message.NewMockSwapFetcher(ctrl)

	pools := make(map[uint64]common.Address)
	pools[8453] = common.HexToAddress("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C5")

	liquidityPools := make(map[uint64]common.Address)
	liquidityPools[10] = common.HexToAddress("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C6")

	s.sigChn = make(chan interface{}, 1)

	// Ethereum: 0x93a9d5e32f5c81cbd17ceb842edc65002e3a79da4efbdc9f1e1f7e97fbcd669b
	s.validLog, _ = hex.DecodeString("000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc200000000000000000000000082af49447d8a07e3bd95bd0d56f35241523fbab100000000000000000000000000000000000000000000000000119baee0ab0400000000000000000000000000000000000000000000000000001199073ea3008d0000000000000000000000000000000000000000000000000000000067bc6e3f0000000000000000000000000000000000000000000000000000000067bc927b00000000000000000000000000000000000000000000000000000000000000000000000000000000000000001886a1eb051c10f20c7386576a6a0716b20b2734000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000000")

	tokens := make(map[uint64]map[string]config.TokenConfig)
	tokens[10] = make(map[string]config.TokenConfig)
	tokens[10]["ETH"] = config.TokenConfig{
		Address:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Decimals: 18,
	}
	tokens[8453] = make(map[string]config.TokenConfig)
	tokens[8453]["ETH"] = config.TokenConfig{
		Address:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Decimals: 18,
	}
	tokenStore := config.TokenStore{
		Tokens: tokens,
	}
	confirmations := make(map[uint64]uint64)
	confirmations[1000] = 100
	confirmations[2000] = 200

	s.handler = message.NewMayanMessageHandler(
		10,
		s.mockEventFilterer,
		liquidityPools,
		pools,
		s.mockCoordinator,
		s.mockHost,
		s.mockCommunication,
		s.mockFetcher,
		s.mockWatcher,
		tokenStore,
		s.mockContract,
		s.mockSwapFetcher,
		s.sigChn,
	)
}

func (s *MayanMessageHandlerTestSuite) Test_HandleMessage_ValidMessage() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.MayanMsg,
		fmt.Sprintf("%d-%s", 10, comm.MayanSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.MayanData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Calldata:      mayanCalldata,
		BorrowAmount:  big.NewInt(9000000000000000),
		DepositTxHash: "0x6cd3de31d0085c8318a19eb1299b00e1d0636838cb6359da6199adcd6d142952",
	}

	calldataBytes, _ := hex.DecodeString(ad.Calldata)
	txHash := common.HexToHash(ad.DepositTxHash)
	orderHash := common.HexToHash(common.Bytes2Hex([]byte("orderHash")))

	s.mockContract.EXPECT().DecodeFulfillCall(calldataBytes).Return(
		&contracts.MayanFulfillParams{
			Recipient:     common.HexToHash("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C6"),
			FulfillAmount: big.NewInt(9900000000000000),
		},
		&contracts.MayanFulfillMsg{
			OrderHash:      orderHash,
			SrcChainId:     24,
			DestChainId:    30,
			ReferrerAddr:   common.HexToHash(ad.Caller.Hex()),
			ReferrerBps:    1,
			ProtocolBps:    3,
			PromisedAmount: 989000,
		},
		nil)
	s.mockSwapFetcher.EXPECT().GetSwap(txHash.Hex()).Return(&mayan.MayanSwap{
		OrderHash: orderHash.Hex(),
	}, nil)
	s.mockContract.EXPECT().GetOrder(gomock.Any(), gomock.Any(), uint8(18)).Return(&contracts.MayanOrder{
		Status:   contracts.OrderCreated,
		AmountIn: 1000000,
	}, nil)
	s.mockWatcher.EXPECT().WaitForConfirmations(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	s.mockCoordinator.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.Nil(err)

	err = <-errChn
	s.Nil(err)
}

func (s *MayanMessageHandlerTestSuite) Test_HandleMessage_BorrowAmountTooHigh() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.MayanMsg,
		fmt.Sprintf("%d-%s", 10, comm.MayanSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.MayanData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Calldata:      mayanCalldata,
		BorrowAmount:  big.NewInt(900000000000000000),
		DepositTxHash: "0x6cd3de31d0085c8318a19eb1299b00e1d0636838cb6359da6199adcd6d142952",
	}

	calldataBytes, _ := hex.DecodeString(ad.Calldata)
	txHash := common.HexToHash(ad.DepositTxHash)
	orderHash := common.HexToHash(common.Bytes2Hex([]byte("orderHash")))

	s.mockContract.EXPECT().DecodeFulfillCall(calldataBytes).Return(
		&contracts.MayanFulfillParams{
			Recipient:     common.HexToHash("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C6"),
			FulfillAmount: big.NewInt(99000000000000000),
		},
		&contracts.MayanFulfillMsg{
			OrderHash:      orderHash,
			SrcChainId:     24,
			DestChainId:    30,
			ReferrerAddr:   common.HexToHash(ad.Caller.Hex()),
			ReferrerBps:    1,
			ProtocolBps:    3,
			PromisedAmount: 989000,
		},
		nil)
	s.mockSwapFetcher.EXPECT().GetSwap(txHash.Hex()).Return(&mayan.MayanSwap{
		OrderHash: orderHash.Hex(),
	}, nil)
	s.mockContract.EXPECT().GetOrder(gomock.Any(), gomock.Any(), uint8(18)).Return(&contracts.MayanOrder{
		Status:   contracts.OrderCreated,
		AmountIn: 1000000,
	}, nil)

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.NotNil(err)

	err = <-errChn
	s.NotNil(err)
}

func (s *MayanMessageHandlerTestSuite) Test_HandleMessage_InvalidOrderHash() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.MayanMsg,
		fmt.Sprintf("%d-%s", 10, comm.MayanSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.MayanData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Calldata:      mayanCalldata,
		BorrowAmount:  big.NewInt(900000000000000000),
		DepositTxHash: "0x6cd3de31d0085c8318a19eb1299b00e1d0636838cb6359da6199adcd6d142952",
	}

	calldataBytes, _ := hex.DecodeString(ad.Calldata)
	txHash := common.HexToHash(ad.DepositTxHash)
	orderHash := common.HexToHash(common.Bytes2Hex([]byte("orderHash")))

	s.mockContract.EXPECT().DecodeFulfillCall(calldataBytes).Return(
		&contracts.MayanFulfillParams{
			Recipient:     common.HexToHash("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C6"),
			FulfillAmount: big.NewInt(99000000000000000),
		},
		&contracts.MayanFulfillMsg{
			OrderHash:      orderHash,
			SrcChainId:     24,
			DestChainId:    30,
			ReferrerAddr:   common.HexToHash(ad.Caller.Hex()),
			ReferrerBps:    1,
			ProtocolBps:    3,
			PromisedAmount: 989000,
		},
		nil)
	s.mockSwapFetcher.EXPECT().GetSwap(txHash.Hex()).Return(&mayan.MayanSwap{
		OrderHash: "invalid",
	}, nil)
	s.mockContract.EXPECT().GetOrder(gomock.Any(), gomock.Any(), uint8(18)).Return(&contracts.MayanOrder{
		Status:   contracts.OrderCreated,
		AmountIn: 100000000000,
	}, nil)

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.NotNil(err)

	err = <-errChn
	s.NotNil(err)
}

func (s *MayanMessageHandlerTestSuite) Test_HandleMessage_InvalidCaller() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.MayanMsg,
		fmt.Sprintf("%d-%s", 10, comm.MayanSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.MayanData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Calldata:      mayanCalldata,
		BorrowAmount:  big.NewInt(900000000000000000),
		DepositTxHash: "0x6cd3de31d0085c8318a19eb1299b00e1d0636838cb6359da6199adcd6d142952",
	}

	calldataBytes, _ := hex.DecodeString(ad.Calldata)
	txHash := common.HexToHash(ad.DepositTxHash)
	orderHash := common.HexToHash(common.Bytes2Hex([]byte("orderHash")))

	s.mockContract.EXPECT().DecodeFulfillCall(calldataBytes).Return(
		&contracts.MayanFulfillParams{
			Recipient:     common.HexToHash("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C6"),
			FulfillAmount: big.NewInt(99000000000000000),
		},
		&contracts.MayanFulfillMsg{
			OrderHash:      orderHash,
			SrcChainId:     24,
			DestChainId:    30,
			ReferrerAddr:   common.HexToHash("0xde526bA5d1ad94cC59D7A79d99A59F607d31A658"),
			ReferrerBps:    1,
			ProtocolBps:    3,
			PromisedAmount: 989000,
		},
		nil)
	s.mockSwapFetcher.EXPECT().GetSwap(txHash.Hex()).Return(&mayan.MayanSwap{
		OrderHash: orderHash.Hex(),
	}, nil)
	s.mockContract.EXPECT().GetOrder(gomock.Any(), gomock.Any(), uint8(18)).Return(&contracts.MayanOrder{
		Status:   contracts.OrderCreated,
		AmountIn: 100000000000,
	}, nil)

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.NotNil(err)

	err = <-errChn
	s.NotNil(err)
}

func (s *MayanMessageHandlerTestSuite) Test_HandleMessage_InvalidFulfillAmount() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.MayanMsg,
		fmt.Sprintf("%d-%s", 10, comm.MayanSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.MayanData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Calldata:      mayanCalldata,
		BorrowAmount:  big.NewInt(900000000000000000),
		DepositTxHash: "0x6cd3de31d0085c8318a19eb1299b00e1d0636838cb6359da6199adcd6d142952",
	}

	calldataBytes, _ := hex.DecodeString(ad.Calldata)
	txHash := common.HexToHash(ad.DepositTxHash)
	orderHash := common.HexToHash(common.Bytes2Hex([]byte("orderHash")))

	s.mockContract.EXPECT().DecodeFulfillCall(calldataBytes).Return(
		&contracts.MayanFulfillParams{
			Recipient:     common.HexToHash("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C6"),
			FulfillAmount: big.NewInt(99000000000000000),
		},
		&contracts.MayanFulfillMsg{
			OrderHash:      orderHash,
			SrcChainId:     24,
			DestChainId:    30,
			ReferrerAddr:   common.HexToHash(ad.Caller.Hex()),
			ReferrerBps:    1,
			ProtocolBps:    3,
			PromisedAmount: 9900000,
		},
		nil)
	s.mockSwapFetcher.EXPECT().GetSwap(txHash.Hex()).Return(&mayan.MayanSwap{
		OrderHash: orderHash.Hex(),
	}, nil)
	s.mockContract.EXPECT().GetOrder(gomock.Any(), gomock.Any(), uint8(18)).Return(&contracts.MayanOrder{
		Status:   contracts.OrderCreated,
		AmountIn: 100000000000,
	}, nil)

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.NotNil(err)

	err = <-errChn
	s.NotNil(err)
}
