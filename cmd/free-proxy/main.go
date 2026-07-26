// Command free-proxy is the single static binary entry point: a cobra CLI that
// hosts the web/API control plane, the local proxy gateway, the OpenVPN tunnel
// manager, and dependency tooling.
package main

import (
	"fmt"
	"os"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/spf13/cobra"
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "free-proxy",
		Short:         "Self-hosted free proxy discovery and SOCKS5/HTTP gateway",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		serveCmd(),
		discoverCmd(),
		credentialsCmd(),
		statusCmd(),
		preflightCmd(),
		logsCmd(),
		adminConfigCmd(),
		databaseUpgradeCmd(),
		doctorCmd(),
		installDepsCmd(),
		installCmd(),
		uninstallCmd(),
		versionCmd(),
	)
	return root
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the binary version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return nil
		},
	}
}

// loadConfig is the shared config entry for every subcommand.
func loadConfig() (*config.Config, error) {
	return config.Load()
}
