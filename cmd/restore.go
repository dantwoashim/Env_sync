// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"

	"github.com/envsync/envsync/internal/crypto"
	"github.com/envsync/envsync/internal/store"
	"github.com/envsync/envsync/internal/ui"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a .env from encrypted backup",
	Long:  "Lists available versions and restores the selected one.",
	RunE:  runRestore,
}

var restoreVersion int

func runRestore(cmd *cobra.Command, args []string) error {
	kp, err := loadIdentity()
	if err != nil {
		return err
	}

	targetFile, _ := cmd.Flags().GetString("file")
	if targetFile == "" {
		targetFile = ".env"
	}

	key, err := crypto.DeriveAtRestKey(kp.X25519Private[:])
	if err != nil {
		return fmt.Errorf("deriving key: %w", err)
	}

	vStore, err := store.New(50)
	if err != nil {
		return err
	}

	projectHash := fmt.Sprintf("%x", key[:8])

	// List versions
	versions, err := vStore.List(projectHash)
	if err != nil || len(versions) == 0 {
		ui.Header("Restore")
		ui.Line("No backups found. Run 'envsync backup' first.")
		ui.Blank()
		return nil
	}

	ui.Header("Available Backups")

	table := ui.NewTable("Version", "Timestamp", "Size")
	for _, v := range versions {
		ts := v.Timestamp.Format("2006-01-02 15:04:05")
		table.AddRow(
			fmt.Sprintf("#%d", v.Sequence),
			ts,
			fmt.Sprintf("%d bytes", v.SizeBytes),
		)
	}
	fmt.Print(table.Render())
	ui.Blank()

	// Select version
	var target int
	if restoreVersion > 0 {
		target = restoreVersion
	} else if len(versions) > 0 {
		target = versions[0].Sequence
	}

	// Restore
	data, err := vStore.Restore(projectHash, target, key)
	if err != nil {
		return fmt.Errorf("loading version #%d: %w", target, err)
	}

	// Confirm
	if !ui.ConfirmAction(fmt.Sprintf("Restore version #%d to %s?", target, targetFile), true) {
		ui.Line("Cancelled.")
		return nil
	}

	if err := os.WriteFile(targetFile, data, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", targetFile, err)
	}

	ui.Success(fmt.Sprintf("Restored version #%d → %s (%d bytes)", target, targetFile, len(data)))
	ui.Blank()

	return nil
}

func init() {
	restoreCmd.Flags().IntVar(&restoreVersion, "version", 0, "Specific version number to restore")
	rootCmd.AddCommand(restoreCmd)
}
