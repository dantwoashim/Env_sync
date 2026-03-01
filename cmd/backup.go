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

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create an encrypted backup of your .env",
	Long:  "Encrypts the current .env file and saves it to the version store.",
	RunE:  runBackup,
}

func runBackup(cmd *cobra.Command, args []string) error {
	kp, err := loadIdentity()
	if err != nil {
		return err
	}

	targetFile, _ := cmd.Flags().GetString("file")
	if targetFile == "" {
		targetFile = ".env"
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		ui.RenderError(ui.ErrEnvFileNotFound(targetFile))
		return fmt.Errorf("file not found: %s", targetFile)
	}

	// Derive encryption key
	key, err := crypto.DeriveAtRestKey(kp.X25519Private[:])
	if err != nil {
		return fmt.Errorf("deriving key: %w", err)
	}


	vStore, err := store.New(50)
	if err != nil {
		return err
	}

	projectHash := fmt.Sprintf("%x", key[:8])
	latestVer, _ := vStore.Latest(projectHash)
	seq := 1
	if latestVer != nil {
		seq = latestVer.Sequence + 1
	}

	if err := vStore.Save(projectHash, data, seq, key); err != nil {
		return fmt.Errorf("saving backup: %w", err)
	}

	ui.Header("Backup Created")
	ui.Success(fmt.Sprintf("Encrypted %s (%d bytes) → version #%d", targetFile, len(data), seq))
	ui.Blank()

	return nil
}

func init() {
	rootCmd.AddCommand(backupCmd)
}
