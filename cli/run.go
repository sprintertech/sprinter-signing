// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sprintertech/sprinter-signing/app"
)

var (
	runCMD = &cobra.Command{
		Use:   "run",
		Short: "Run app",
		Long:  "Run app",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Run(); err != nil {
				return err
			}
			return nil
		},
	}
)
