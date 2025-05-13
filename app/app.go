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
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	solverConfig "github.com/sprintertech/solver-config/go/config"
	"github.com/sprintertech/sprinter-signing/api"
	"github.com/sprintertech/sprinter-signing/api/handlers"
	"github.com/sprintertech/sprinter-signing/cache"
	"github.com/sprintertech/sprinter-signing/chains/evm"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/contracts"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
	evmListener "github.com/sprintertech/sprinter-signing/chains/evm/listener"
	evmMessage "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sprintertech/sprinter-signing/comm/elector"
	"github.com/sprintertech/sprinter-signing/comm/p2p"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/health"
	"github.com/sprintertech/sprinter-signing/jobs"
	"github.com/sprintertech/sprinter-signing/keyshare"
	"github.com/sprintertech/sprinter-signing/metrics"
	"github.com/sprintertech/sprinter-signing/price"
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
	configURL := viper.GetString("config-url")

	var configuration *config.Config
	if configURL != "" {
		configuration, err = config.GetSharedConfigFromNetwork(configURL)
		panicOnError(err)
	}

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

	go health.StartHealthEndpoint(configuration.RelayerConfig.HealthPort)

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

	solverConfig, err := solverConfig.FetchSolverConfig(ctx)
	panicOnError(err)

	var hubPoolContract evmMessage.TokenMatcher
	var mayanSwiftContract *contracts.MayanSwiftContract
	acrossPools := make(map[uint64]common.Address)
	mayanPools := make(map[uint64]common.Address)
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
					mayanSwiftContract = contracts.NewMayanSwiftContract(client, common.HexToAddress(c.MayanSwift))
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
				c, err := evm.NewEVMConfig(chainConfig)
				panicOnError(err)

				client, err := evmClient.NewEVMClient(c.GeneralChainConfig.Endpoint, nil)
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
					acrossMh := evmMessage.NewAcrossMessageHandler(
						*c.GeneralChainConfig.Id,
						client,
						acrossPools,
						coordinator,
						host,
						communication,
						keyshareStore,
						hubPoolContract,
						tokenStore,
						watcher,
						sigChn)
					go acrossMh.Listen(ctx)

					mh.RegisterMessageHandler(evmMessage.AcrossMessage, acrossMh)
					supportedChains[*c.GeneralChainConfig.Id] = struct{}{}
					confirmationsPerChain[*c.GeneralChainConfig.Id] = c.ConfirmationsByValue
				}

				if c.MayanSwift != "" {
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

					mh.RegisterMessageHandler(evmMessage.MayanMessage, mayanMh)
					supportedChains[*c.GeneralChainConfig.Id] = struct{}{}
					confirmationsPerChain[*c.GeneralChainConfig.Id] = c.ConfirmationsByValue
				}

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
	go api.Serve(ctx, configuration.RelayerConfig.ApiAddr, signingHandler, statusHandler, confirmationsHandler)

	sig := <-sysErr
	log.Info().Msgf("terminating got ` [%v] signal", sig)
	return nil
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
