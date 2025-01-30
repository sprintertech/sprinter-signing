// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package cli

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sprintertech/sprinter-signing/cli/keygen"
	"github.com/sprintertech/sprinter-signing/cli/peer"
	"github.com/sprintertech/sprinter-signing/cli/topology"
	"github.com/sprintertech/sprinter-signing/cli/utils"
	"github.com/sprintertech/sprinter-signing/config"
)

var (
	rootCMD = &cobra.Command{
		Use: "",
	}
)

func init() {
	config.BindFlags(rootCMD)
	rootCMD.PersistentFlags().String("name", "", "relayer name")
	_ = viper.BindPFlag("name", rootCMD.PersistentFlags().Lookup("name"))

	rootCMD.PersistentFlags().String("config-url", "", "URL of shared configuration")
	_ = viper.BindPFlag("config-url", rootCMD.PersistentFlags().Lookup("config-url"))
}

func Execute() {
	rootCMD.AddCommand(runCMD, peer.PeerCLI, topology.TopologyCLI, utils.UtilsCLI, keygen.KeygenCLI)
	if err := rootCMD.Execute(); err != nil {
		log.Fatal().Err(err).Msg("failed to execute root cmd")
	}
}
