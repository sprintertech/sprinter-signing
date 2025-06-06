// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package relayer

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

type RelayerConfig struct {
	OpenTelemetryCollectorURL string
	LogLevel                  zerolog.Level
	LogFile                   string
	HealthPort                uint16
	Env                       string
	Id                        string
	MpcConfig                 MpcRelayerConfig
	BullyConfig               BullyConfig
	CoinmarketcapConfig       CoinmarketcapConfig
	SolverConfig              SolverConfig
	ApiAddr                   string
}

type CoinmarketcapConfig struct {
	Url    string `default:"https://pro-api.coinmarketcap.com"`
	ApiKey string
}

type SolverConfig struct {
	SecretKey string
	AccessKey string
}

type MpcRelayerConfig struct {
	TopologyConfiguration   TopologyConfiguration
	Port                    uint16
	KeysharePath            string
	FrostKeysharePath       string
	Key                     string
	CommHealthCheckInterval time.Duration
}

type BullyConfig struct {
	PingWaitTime     time.Duration
	PingBackOff      time.Duration
	PingInterval     time.Duration
	ElectionWaitTime time.Duration
	BullyWaitTime    time.Duration
}

type TopologyConfiguration struct {
	EncryptionKey string `mapstructure:"EncryptionKey" json:"encryptionKey"`
	Url           string `mapstructure:"Url" json:"url"`
	Path          string `mapstructure:"Path" json:"path"`
}

type RawRelayerConfig struct {
	OpenTelemetryCollectorURL string              `mapstructure:"OpenTelemetryCollectorURL" json:"opentelemetryCollectorURL"`
	LogLevel                  string              `mapstructure:"LogLevel" json:"logLevel" default:"info"`
	LogFile                   string              `mapstructure:"LogFile" json:"logFile" default:"out.log"`
	HealthPort                string              `mapstructure:"HealthPort" json:"healthPort" default:"9001"`
	Env                       string              `mapstructure:"Env" json:"env"`
	Id                        string              `mapstructure:"Id" json:"id"`
	MpcConfig                 RawMpcRelayerConfig `mapstructure:"MpcConfig" json:"mpcConfig"`
	BullyConfig               RawBullyConfig      `mapstructure:"BullyConfig" json:"bullyConfig"`
	CoinmarketcapConfig       CoinmarketcapConfig `mapstructure:"CoinmarketcapConfig" json:"coinmarketcapConfig"`
	SolverConfig              SolverConfig        `mapstructure:"SolverConfig" json:"solverConfig"`
	ApiAddr                   string              `mapstructure:"apiAddr" default:"0.0.0.0:3000"`
}

type RawMpcRelayerConfig struct {
	KeysharePath            string                `mapstructure:"KeysharePath" json:"keysharePath"`
	FrostKeysharePath       string                `mapstructure:"FrostKeysharePath" json:"frostKeysharePath"`
	Key                     string                `mapstructure:"Key" json:"key"`
	Port                    string                `mapstructure:"Port" json:"port" default:"9000"`
	TopologyConfiguration   TopologyConfiguration `mapstructure:"TopologyConfiguration" json:"topologyConfiguration"`
	CommHealthCheckInterval string                `mapstructure:"CommHealthCheckInterval" json:"commHealthCheckInterval" default:"5m"`
}

type RawBullyConfig struct {
	PingWaitTime     string `mapstructure:"PingWaitTime" json:"pingWaitTime" default:"1s"`
	PingBackOff      string `mapstructure:"PingBackOff" json:"pingBackOff" default:"1s"`
	PingInterval     string `mapstructure:"PingInterval" json:"pingInterval" default:"1s"`
	ElectionWaitTime string `mapstructure:"ElectionWaitTime" json:"electionWaitTime" default:"2s"`
	BullyWaitTime    string `mapstructure:"BullyWaitTime" json:"bullyWaitTime" default:"3m"`
}

func (c *RawRelayerConfig) Validate() error {
	if c.MpcConfig.TopologyConfiguration.EncryptionKey == "" {
		return errors.New("topology configuration encryption key not provided")
	}
	if c.MpcConfig.TopologyConfiguration.Url == "" {
		return errors.New("topology configuration url not provided")
	}
	if c.MpcConfig.TopologyConfiguration.Path == "" {
		return errors.New("topology configuration path not provided")
	}
	return nil
}

// NewRelayerConfig parses RawRelayerConfig into RelayerConfig
func NewRelayerConfig(rawConfig RawRelayerConfig) (RelayerConfig, error) {
	config := RelayerConfig{}
	err := rawConfig.Validate()
	if err != nil {
		return config, err
	}

	logLevel, err := zerolog.ParseLevel(rawConfig.LogLevel)
	if err != nil {
		return config, fmt.Errorf("unknown log level: %s", rawConfig.LogLevel)
	}
	config.LogLevel = logLevel

	config.LogFile = rawConfig.LogFile
	config.OpenTelemetryCollectorURL = rawConfig.OpenTelemetryCollectorURL

	healthPort, err := strconv.ParseInt(rawConfig.HealthPort, 0, 16)
	if err != nil {
		return RelayerConfig{}, fmt.Errorf("unable to parse health port %v", err)
	}
	if healthPort < 0 || healthPort > math.MaxUint16 {
		return RelayerConfig{}, fmt.Errorf("mpc port %d is out of valid range for uint16", healthPort)
	}
	config.HealthPort = uint16(healthPort)

	mpcConfig, err := parseMpcConfig(rawConfig)
	if err != nil {
		return RelayerConfig{}, err
	}
	config.MpcConfig = mpcConfig

	bullyConfig, err := parseBullyConfig(rawConfig)
	if err != nil {
		return RelayerConfig{}, err
	}

	config.CoinmarketcapConfig = rawConfig.CoinmarketcapConfig
	config.BullyConfig = bullyConfig
	config.Env = rawConfig.Env
	config.Id = rawConfig.Id
	config.ApiAddr = rawConfig.ApiAddr
	config.SolverConfig = rawConfig.SolverConfig
	return config, nil
}

func parseMpcConfig(rawConfig RawRelayerConfig) (MpcRelayerConfig, error) {
	var mpcConfig MpcRelayerConfig

	port, err := strconv.ParseInt(rawConfig.MpcConfig.Port, 0, 16)
	if err != nil {
		return MpcRelayerConfig{}, fmt.Errorf("unable to parse mpc port from config %v", err)
	}
	if port < 0 || port > math.MaxUint16 {
		return MpcRelayerConfig{}, fmt.Errorf("mpc port %d is out of valid range for uint16", port)
	}
	mpcConfig.Port = uint16(port)

	mpcConfig.TopologyConfiguration = rawConfig.MpcConfig.TopologyConfiguration
	mpcConfig.KeysharePath = rawConfig.MpcConfig.KeysharePath
	mpcConfig.FrostKeysharePath = rawConfig.MpcConfig.FrostKeysharePath
	mpcConfig.Key = rawConfig.MpcConfig.Key

	duration, err := time.ParseDuration(rawConfig.MpcConfig.CommHealthCheckInterval)
	if err != nil {
		return MpcRelayerConfig{}, fmt.Errorf("unable to parse communication health check interval time: %w", err)
	}
	mpcConfig.CommHealthCheckInterval = duration

	return mpcConfig, nil
}

func parseBullyConfig(rawConfig RawRelayerConfig) (BullyConfig, error) {
	electionWaitTime, err := time.ParseDuration(rawConfig.BullyConfig.ElectionWaitTime)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully election wait time: %w", err)
	}

	pingWaitTime, err := time.ParseDuration(rawConfig.BullyConfig.PingWaitTime)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully ping wait time: %w", err)
	}

	pingInterval, err := time.ParseDuration(rawConfig.BullyConfig.PingInterval)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully ping interval time: %w", err)
	}

	pingBackOff, err := time.ParseDuration(rawConfig.BullyConfig.PingBackOff)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully ping back off time: %w", err)
	}

	bullyWaitTime, err := time.ParseDuration(rawConfig.BullyConfig.BullyWaitTime)
	if err != nil {
		return BullyConfig{}, fmt.Errorf("unable to parse bully wait time: %w", err)
	}

	return BullyConfig{
		PingWaitTime:     pingWaitTime,
		PingBackOff:      pingBackOff,
		PingInterval:     pingInterval,
		ElectionWaitTime: electionWaitTime,
		BullyWaitTime:    bullyWaitTime,
	}, nil
}
