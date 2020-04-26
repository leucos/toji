package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "show program version",
	Example: "toji version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("toji version %s (built %s)\n", Version, BuildDate)
	},
	SilenceUsage: true,
}
