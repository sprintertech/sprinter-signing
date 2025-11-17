// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package app

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethereumCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/sprintertech/lifi-solver/pkg/pricing"
	"github.com/sprintertech/lifi-solver/pkg/protocols"
	"github.com/sprintertech/lifi-solver/pkg/protocols/lifi/validation"
	"github.com/sprintertech/lifi-solver/pkg/router"
	"github.com/sprintertech/lifi-solver/pkg/token"
	"github.com/sprintertech/lifi-solver/pkg/tokenpricing/pyth"
	solverConfig "github.com/sprintertech/solver-config/go/config"
	"github.com/sprintertech/sprinter-signing/api"
	"github.com/sprintertech/sprinter-signing/api/handlers"
	"github.com/sprintertech/sprinter-signing/cache"
	"github.com/sprintertech/sprinter-signing/chains/evm"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/contracts"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
	evmListener "github.com/sprintertech/sprinter-signing/chains/evm/listener"
	evmMessage "github.com/sprintertech/sprinter-signing/chains/evm/message"

	lifiConfig "github.com/sprintertech/lifi-solver/pkg/config"
	"github.com/sprintertech/sprinter-signing/chains/lighter"
	lighterMessage "github.com/sprintertech/sprinter-signing/chains/lighter/message"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/comm/elector"
	"github.com/sprintertech/sprinter-signing/comm/p2p"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/jobs"
	"github.com/sprintertech/sprinter-signing/keyshare"
	"github.com/sprintertech/sprinter-signing/metrics"
	"github.com/sprintertech/sprinter-signing/price"
	"github.com/sprintertech/sprinter-signing/protocol/across"
	"github.com/sprintertech/sprinter-signing/protocol/lifi"
	lighterAPI "github.com/sprintertech/sprinter-signing/protocol/lighter"
	"github.com/sprintertech/sprinter-signing/protocol/mayan"
	"github.com/sprintertech/sprinter-signing/topology"
	"github.com/sprintertech/sprinter-signing/tss"
	coreEvm "github.com/sygmaprotocol/sygma-core/chains/evm"
	evmClient "github.com/sygmaprotocol/sygma-core/chains/evm/client"
	coreListener "github.com/sygmaprotocol/sygma-core/chains/evm/listener"
	"github.com/sygmaprotocol/sygma-core/crypto/secp256k1"

	"github.com/sygmaprotocol/sygma-core/observability"
	"github.com/sygmaprotocol/sygma-core/relayer"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/store"
	"github.com/sygmaprotocol/sygma-core/store/lvldb"
)

var Version string

//nolint:gocognit
func Run() error {
	var err error

	configFlag := viper.GetString(config.ConfigFlagName)
	var configuration *config.Config
	if strings.ToLower(configFlag) == "env" {
		configuration, err = config.GetConfigFromENV(configuration)
		panicOnError(err)
	} else {
		configuration, err = config.GetConfigFromFile(configFlag, configuration)
		panicOnError(err)
	}

	observability.ConfigureLogger(configuration.RelayerConfig.LogLevel, os.Stdout)

	log.Info().Msg("Successfully loaded configuration")

	topologyProvider, err := topology.NewNetworkTopologyProvider(configuration.RelayerConfig.MpcConfig.TopologyConfiguration, http.DefaultClient)
	panicOnError(err)
	topologyStore := topology.NewTopologyStore(configuration.RelayerConfig.MpcConfig.TopologyConfiguration.Path)
	networkTopology, err := topologyStore.Topology()
	// if topology is not already in file, read from provider
	if err != nil {
		networkTopology, err = topologyProvider.NetworkTopology("")
		panicOnError(err)

		err = topologyStore.StoreTopology(networkTopology)
		panicOnError(err)
	}
	log.Info().Msgf("Successfully loaded topology")

	privBytes, err := crypto.ConfigDecodeKey(configuration.RelayerConfig.MpcConfig.Key)
	panicOnError(err)

	priv, err := crypto.UnmarshalPrivateKey(privBytes)
	panicOnError(err)

	connectionGate := p2p.NewConnectionGate(networkTopology)
	host, err := p2p.NewHost(priv, networkTopology, connectionGate, configuration.RelayerConfig.MpcConfig.Port)
	panicOnError(err)
	log.Info().Str("peerID", host.ID().String()).Msg("Successfully created libp2p host")

	communication := p2p.NewCommunication(host, "p2p/sprinter")
	electorFactory := elector.NewCoordinatorElectorFactory(host, configuration.RelayerConfig.BullyConfig)
	coordinator := tss.NewCoordinator(host, communication, electorFactory)

	db, err := lvldb.NewLvlDB(viper.GetString(config.BlockstoreFlagName))
	if err != nil {
		panicOnError(err)
	}
	blockstore := store.NewBlockStore(db)
	keyshareStore := keyshare.NewECDSAKeyshareStore(configuration.RelayerConfig.MpcConfig.KeysharePath)

	mp, err := observability.InitMetricProvider(context.Background(), configuration.RelayerConfig.OpenTelemetryCollectorURL)
	panicOnError(err)
	defer func() {
		if err := mp.Shutdown(context.Background()); err != nil {
			log.Error().Msgf("Error shutting down meter provider: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sygmaMetrics, err := metrics.NewSygmaMetrics(ctx, mp.Meter("relayer-metric-provider"), configuration.RelayerConfig.Env, configuration.RelayerConfig.Id, Version)
	if err != nil {
		panic(err)
	}
	msgChan := make(chan []*message.Message)
	sigChn := make(chan interface{})

	priceAPI := price.NewCoinmarketcapAPI(
		configuration.RelayerConfig.CoinmarketcapConfig.Url,
		configuration.RelayerConfig.CoinmarketcapConfig.ApiKey)

	signatureCache := cache.NewSignatureCache(communication)
	go signatureCache.Watch(ctx, sigChn)

	supportedChains := make(map[uint64]struct{})
	confirmationsPerChain := make(map[uint64]map[uint64]uint64)
	domains := make(map[uint64]relayer.RelayedChain)

	solverConfigOpts := []solverConfig.Option{
		solverConfig.WithCredentials(
			configuration.RelayerConfig.SolverConfig.AccessKey,
			configuration.RelayerConfig.SolverConfig.SecretKey),
	}
	staging := viper.GetBool(config.StagingFlagName)
	if staging {
		solverConfigOpts = append(solverConfigOpts, solverConfig.WithStaging())
	}
	solverConfig, err := solverConfig.FetchSolverConfig(ctx, solverConfigOpts...)
	panicOnError(err)

	keyshare, err := keyshareStore.GetKeyshare()
	var mpcAddress common.Address
	if err == nil {
		mpcAddress = ethereumCrypto.PubkeyToAddress(*keyshare.Key.ECDSAPub.ToBtcecPubKey().ToECDSA())
	} else {
		mpcAddress = common.HexToAddress(solverConfig.ProtocolsMetadata.Sprinter.MpcAddress)
	}

	var hubPoolContract across.TokenMatcher
	acrossPools := make(map[uint64]common.Address)
	mayanPools := make(map[uint64]common.Address)
	lifiOutputSettlers := make(map[uint64]common.Address)
	repayerAddresses := make(map[uint64]common.Address)
	tokens := make(map[uint64]map[string]config.TokenConfig)
	for _, chainConfig := range configuration.ChainConfigs {
		switch chainConfig["type"] {
		case "evm":
			{
				c, err := evm.NewEVMConfig(chainConfig, *solverConfig)
				panicOnError(err)
				kp, _ := secp256k1.GenerateKeypair()
				client, err := evmClient.NewEVMClient(c.GeneralChainConfig.Endpoint, kp)
				panicOnError(err)

				if c.AcrossPool != "" {
					poolAddress := common.HexToAddress(c.AcrossPool)
					acrossPools[*c.GeneralChainConfig.Id] = poolAddress
				}

				if c.MayanSwift != "" {
					poolAddress := common.HexToAddress(c.MayanSwift)
					mayanPools[*c.GeneralChainConfig.Id] = poolAddress
				}

				if c.LifiOutputSettler != "" {
					settlerAddress := common.HexToAddress(c.LifiOutputSettler)
					lifiOutputSettlers[*c.GeneralChainConfig.Id] = settlerAddress
				}

				if c.AcrossHubPool != "" {
					hubPoolAddress := common.HexToAddress(c.AcrossHubPool)
					hubPoolContract = contracts.NewHubPoolContract(client, hubPoolAddress, c.Tokens)
				}

				if c.Repayer != "" {
					repayerAddress := common.HexToAddress(c.Repayer)
					repayerAddresses[*c.GeneralChainConfig.Id] = repayerAddress
				}

				tokens[*c.GeneralChainConfig.Id] = c.Tokens
			}
		default:
			panic(fmt.Errorf("type '%s' not recognized", chainConfig["type"]))
		}
	}
	tokenStore := config.TokenStore{
		Tokens: tokens,
	}

	for _, chainConfig := range configuration.ChainConfigs {
		switch chainConfig["type"] {
		case "evm":
			{
				c, err := evm.NewEVMConfig(chainConfig, *solverConfig)
				panicOnError(err)

				kp, _ := secp256k1.GenerateKeypair()
				client, err := evmClient.NewEVMClient(c.GeneralChainConfig.Endpoint, kp)
				panicOnError(err)

				log.Info().Uint64("chain", *c.GeneralChainConfig.Id).Msgf("Registering EVM domain")

				l := log.With().Str("chain", fmt.Sprintf("%v", c.GeneralChainConfig.Name)).Uint64("domainID", *c.GeneralChainConfig.Id)

				watcher := evmMessage.NewWatcher(
					client,
					priceAPI,
					tokenStore,
					c.ConfirmationsByValue,
					// nolint:gosec
					time.Duration(c.GeneralChainConfig.Blocktime)*time.Second,
				)

				mh := message.NewMessageHandler()
				if c.AcrossPool != "" {
					acrossDepositFetcher := across.NewAcrossDepositFetcher(
						*c.GeneralChainConfig.Id,
						tokenStore,
						client,
						hubPoolContract,
					)
					acrossMh := evmMessage.NewAcrossMessageHandler(
						*c.GeneralChainConfig.Id,
						acrossPools,
						repayerAddresses,
						coordinator,
						host,
						communication,
						keyshareStore,
						acrossDepositFetcher,
						watcher,
						sigChn)
					go acrossMh.Listen(ctx)

					mh.RegisterMessageHandler(message.MessageType(comm.AcrossMsg.String()), acrossMh)
					supportedChains[*c.GeneralChainConfig.Id] = struct{}{}
					confirmationsPerChain[*c.GeneralChainConfig.Id] = c.ConfirmationsByValue
				}

				if c.MayanSwift != "" {
					mayanSwiftContract := contracts.NewMayanSwiftContract(client, common.HexToAddress(c.MayanSwift))
					mayanApi := mayan.NewMayanExplorer()
					mayanMh := evmMessage.NewMayanMessageHandler(
						*c.GeneralChainConfig.Id,
						client,
						repayerAddresses,
						mayanPools,
						coordinator,
						host,
						communication,
						keyshareStore,
						watcher,
						tokenStore,
						mayanSwiftContract,
						mayanApi,
						sigChn)
					go mayanMh.Listen(ctx)

					mh.RegisterMessageHandler(message.MessageType(comm.MayanMsg.String()), mayanMh)
					supportedChains[*c.GeneralChainConfig.Id] = struct{}{}
					confirmationsPerChain[*c.GeneralChainConfig.Id] = c.ConfirmationsByValue
				}

				if c.LifiOutputSettler != "" {
					usdPricer := pyth.NewClient(ctx)
					err = usdPricer.Start(ctx)
					panicOnError(err)

					lifiConfig, err := lifiConfig.GetSolverConfig(solverConfig, protocols.LifiEscrow, lifiConfig.PulsarSolver)
					panicOnError(err)

					resolver := token.NewTokenResolver(solverConfig, usdPricer)
					orderPricer := pricing.NewStandardPricer(resolver)
					lifiApi := lifi.NewLifiAPI()
					lifiValidator := validation.NewLifiEscrowOrderValidator(solverConfig, resolver)

					lifiMh := evmMessage.NewLifiEscrowMessageHandler(
						*c.GeneralChainConfig.Id,
						mpcAddress,
						lifiOutputSettlers,
						coordinator,
						host,
						communication,
						keyshareStore,
						watcher,
						tokenStore,
						lifiApi,
						orderPricer,
						router.NewRouter(resolver, nil, nil, lifiConfig.Routes),
						lifiValidator,
						sigChn,
					)
					go lifiMh.Listen(ctx)
					mh.RegisterMessageHandler(message.MessageType(comm.LifiEscrowMsg.String()), lifiMh)
					supportedChains[*c.GeneralChainConfig.Id] = struct{}{}
					confirmationsPerChain[*c.GeneralChainConfig.Id] = c.ConfirmationsByValue
				}

				lifiUnlockMh := evmMessage.NewLifiUnlockHandler(
					*c.GeneralChainConfig.Id,
					repayerAddresses,
					coordinator,
					host,
					communication,
					keyshareStore,
				)
				go lifiUnlockMh.Listen(ctx)
				mh.RegisterMessageHandler(message.MessageType(comm.LifiUnlockMsg.String()), lifiUnlockMh)

				var startBlock *big.Int
				var listener *coreListener.EVMListener
				eventHandlers := make([]coreListener.EventHandler, 0)
				if c.Admin != "" {
					head, err := client.LatestBlock()
					panicOnError(err)

					startBlock = head

					tssListener := events.NewListener(client)
					adminAddress := common.HexToAddress(c.Admin)
					eventHandlers = append(eventHandlers, evmListener.NewKeygenEventHandler(l, tssListener, coordinator, host, communication, keyshareStore, adminAddress, networkTopology.Threshold))
					eventHandlers = append(eventHandlers, evmListener.NewRefreshEventHandler(l, topologyProvider, topologyStore, tssListener, coordinator, host, communication, connectionGate, keyshareStore, adminAddress))
					listener = coreListener.NewEVMListener(client, eventHandlers, blockstore, sygmaMetrics, *c.GeneralChainConfig.Id, c.BlockRetryInterval, new(big.Int).SetUint64(c.GeneralChainConfig.BlockConfirmations), c.BlockInterval)
				}

				chain := coreEvm.NewEVMChain(listener, mh, nil, *c.GeneralChainConfig.Id, startBlock)
				domains[*c.GeneralChainConfig.Id] = chain
			}
		default:
			panic(fmt.Errorf("type '%s' not recognized", chainConfig["type"]))
		}
	}

	lighterConfig, err := lighter.NewLighterConfig(*solverConfig)
	panicOnError(err)
	lighterAPI := lighterAPI.NewLighterAPI()
	lighterMessageHandler := lighterMessage.NewLighterMessageHandler(
		lighterConfig.WithdrawalAddress,
		lighterConfig.UsdcAddress,
		lighterConfig.RepaymentAddress,
		lighterAPI,
		coordinator,
		host,
		communication,
		keyshareStore,
		sigChn,
	)
	go lighterMessageHandler.Listen(ctx)
	lighterChain := lighter.NewLighterChain(lighterMessageHandler)
	domains[lighter.LIGHTER_DOMAIN_ID] = lighterChain
	supportedChains[lighter.LIGHTER_DOMAIN_ID] = struct{}{}

	go jobs.StartCommunicationHealthCheckJob(host, configuration.RelayerConfig.MpcConfig.CommHealthCheckInterval, sygmaMetrics)

	r := relayer.NewRelayer(domains, sygmaMetrics)
	go r.Start(ctx, msgChan)

	sysErr := make(chan os.Signal, 1)
	signal.Notify(sysErr,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT)

	relayerName := viper.GetString("name")
	log.Info().Msgf("Started relayer: %s with PID: %s. Version: v%s", relayerName, host.ID().String(), Version)

	_, err = keyshareStore.GetKeyshare()
	if err != nil {
		log.Info().Msg("Relayer not part of MPC. Waiting for refresh event...")
	}

	signingHandler := handlers.NewSigningHandler(msgChan, supportedChains)
	statusHandler := handlers.NewStatusHandler(signatureCache, supportedChains)
	confirmationsHandler := handlers.NewConfirmationsHandler(confirmationsPerChain)
	unlockHandler := handlers.NewUnlockHandler(msgChan, supportedChains)
	go api.Serve(
		ctx,
		configuration.RelayerConfig.ApiAddr,
		signingHandler,
		unlockHandler,
		statusHandler,
		confirmationsHandler)

	sig := <-sysErr
	log.Info().Msgf("terminating got ` [%v] signal", sig)
	return nil
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
