package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X github.com/renaldid/logpilot/cmd.Version=x.y.z"
var Version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the logpilot version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("logpilot %s\n", Version)
		},
	}
}
