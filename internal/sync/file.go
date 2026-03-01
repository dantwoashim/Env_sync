// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package sync

import "os"

// readFile reads a file from disk.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
