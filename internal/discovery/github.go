// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// GitHubKeyCache caches GitHub SSH key lookups.
type GitHubKeyCache struct {
	mu       sync.RWMutex
	memory   map[string]cachedKeys
	cacheDir string
}

type cachedKeys struct {
	Keys      []string  `json:"keys"`
	FetchedAt time.Time `json:"fetched_at"`
}

const (
	memCacheTTL  = 5 * time.Minute
	diskCacheTTL = 1 * time.Hour
	githubAPITimeout = 10 * time.Second
)

// NewGitHubKeyCache creates a new cache with the given disk cache directory.
func NewGitHubKeyCache(cacheDir string) *GitHubKeyCache {
	return &GitHubKeyCache{
		memory:   make(map[string]cachedKeys),
		cacheDir: cacheDir,
	}
}

// FetchEd25519Keys fetches Ed25519 SSH public keys for a GitHub user.
// Uses memory cache (5min) → disk cache (1hr) → network.
func (c *GitHubKeyCache) FetchEd25519Keys(username string) ([]ssh.PublicKey, error) {
	// Strip @ prefix if present
	username = strings.TrimPrefix(username, "@")

	// Check memory cache
	c.mu.RLock()
	if cached, ok := c.memory[username]; ok && time.Since(cached.FetchedAt) < memCacheTTL {
		c.mu.RUnlock()
		return parseSSHKeys(cached.Keys)
	}
	c.mu.RUnlock()

	// Check disk cache
	diskKeys, err := c.readDiskCache(username)
	if err == nil {
		c.mu.Lock()
		c.memory[username] = *diskKeys
		c.mu.Unlock()
		return parseSSHKeys(diskKeys.Keys)
	}

	// Fetch from GitHub
	keys, err := fetchGitHubKeys(username)
	if err != nil {
		return nil, err
	}

	// Filter Ed25519 only
	var ed25519Keys []string
	for _, key := range keys {
		if strings.HasPrefix(key, "ssh-ed25519 ") {
			ed25519Keys = append(ed25519Keys, key)
		}
	}

	if len(ed25519Keys) == 0 {
		return nil, fmt.Errorf("GitHub user @%s has no Ed25519 SSH keys. They need to add one:\n"+
			"  ssh-keygen -t ed25519\n"+
			"  Then add to GitHub: https://github.com/settings/keys", username)
	}

	// Cache
	cached := cachedKeys{
		Keys:      ed25519Keys,
		FetchedAt: time.Now(),
	}
	c.mu.Lock()
	c.memory[username] = cached
	c.mu.Unlock()
	_ = c.writeDiskCache(username, &cached)

	return parseSSHKeys(ed25519Keys)
}

// fetchGitHubKeys fetches raw SSH keys from GitHub's public API.
func fetchGitHubKeys(username string) ([]string, error) {
	url := fmt.Sprintf("https://github.com/%s.keys", username)

	client := &http.Client{Timeout: githubAPITimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching GitHub keys for @%s: %w", username, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("GitHub user @%s not found", username)
	}
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("GitHub rate limit exceeded. Wait a minute and try again")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub returned HTTP %d for @%s", resp.StatusCode, username)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, fmt.Errorf("reading GitHub response: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	var keys []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
}

// parseSSHKeys parses raw SSH public key strings into ssh.PublicKey objects.
func parseSSHKeys(rawKeys []string) ([]ssh.PublicKey, error) {
	var keys []ssh.PublicKey
	for _, raw := range rawKeys {
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(raw))
		if err != nil {
			continue
		}
		keys = append(keys, pubKey)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no valid SSH keys could be parsed")
	}
	return keys, nil
}

func (c *GitHubKeyCache) readDiskCache(username string) (*cachedKeys, error) {
	path := filepath.Join(c.cacheDir, "github_keys", username+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cached cachedKeys
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	if time.Since(cached.FetchedAt) > diskCacheTTL {
		return nil, fmt.Errorf("disk cache expired")
	}

	return &cached, nil
}

func (c *GitHubKeyCache) writeDiskCache(username string, cached *cachedKeys) error {
	dir := filepath.Join(c.cacheDir, "github_keys")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, username+".json"), data, 0600)
}
