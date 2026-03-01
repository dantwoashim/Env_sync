// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose  bool
	quiet    bool
	noColor  bool
	cfgFile  string
	envFile  string

	// Build info (set via ldflags)
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "envsync",
	Short: "P2P environment variable synchronization",
	Long: `EnvSync — Securely sync .env files between developers.

Zero accounts. Zero servers. End-to-end encrypted.
Uses your existing SSH keys for identity and trust.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "  ✗ %s\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all output except errors")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Use alternate config file")
	rootCmd.PersistentFlags().StringVar(&envFile, "file", ".env", "Target specific .env file")
}
