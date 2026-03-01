// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/base64"
	"fmt"

	"github.com/envsync/envsync/internal/crypto"
	"github.com/envsync/envsync/internal/ui"
	"github.com/spf13/cobra"
)

var serviceKeyCmd = &cobra.Command{
	Use:   "service-key",
	Short: "Manage service account keys for CI/CD",
	Long:  "Generate, export, or import service keys for use in GitHub Actions and other CI environments.",
}

var serviceKeyGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new service account key",
	RunE:  runServiceKeyGenerate,
}

var serviceKeyExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the public key for team registration",
	RunE:  runServiceKeyExport,
}

var serviceKeyOutput string

func runServiceKeyGenerate(cmd *cobra.Command, args []string) error {
	sk, err := crypto.GenerateServiceKey()
	if err != nil {
		return err
	}

	outPath := serviceKeyOutput
	if outPath == "" {
		outPath = ".envsync-service-key"
	}

	if err := sk.SaveToFile(outPath); err != nil {
		return err
	}

	pubKeyB64 := base64.StdEncoding.EncodeToString(sk.PublicKey)

	ui.Header("Service Key Generated")
	ui.Success(fmt.Sprintf("Private key saved to: %s", outPath))
	ui.Blank()
	ui.Line("Add this to your GitHub Actions secrets:")
	ui.Blank()
	ui.Code(fmt.Sprintf("  ENVSYNC_SERVICE_KEY=%s", base64.StdEncoding.EncodeToString(sk.ExportPrivateKey())))
	ui.Blank()
	ui.Line("Public key (for team registration):")
	ui.Code(fmt.Sprintf("  %s", pubKeyB64))
	ui.Blank()
	ui.Warning("Keep the private key secret. Never commit it to git.")
	ui.Line("Add '.envsync-service-key' to your .gitignore.")

	return nil
}

func runServiceKeyExport(cmd *cobra.Command, args []string) error {
	keyPath := serviceKeyOutput
	if keyPath == "" {
		keyPath = ".envsync-service-key"
	}

	sk, err := crypto.LoadServiceKeyFromFile(keyPath)
	if err != nil {
		ui.RenderError(ui.StructuredError{
			Category:   ui.ErrFile,
			Message:    "Service key not found",
			Cause:      fmt.Sprintf("Expected key at %s", keyPath),
			Suggestion: "Run 'envsync service-key generate' first",
		})
		return err
	}

	pubKeyB64 := base64.StdEncoding.EncodeToString(sk.PublicKey)
	fmt.Println(pubKeyB64)

	return nil
}

func init() {
	serviceKeyGenerateCmd.Flags().StringVarP(&serviceKeyOutput, "output", "o", "", "Output path for key file")
	serviceKeyExportCmd.Flags().StringVarP(&serviceKeyOutput, "key", "k", "", "Path to key file")

	serviceKeyCmd.AddCommand(serviceKeyGenerateCmd)
	serviceKeyCmd.AddCommand(serviceKeyExportCmd)
	rootCmd.AddCommand(serviceKeyCmd)
}
