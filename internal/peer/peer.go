// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package peer

import (
	"fmt"
	"time"
)

// TrustState represents the trust level of a peer.
type TrustState string

const (
	TrustUnknown TrustState = "unknown"
	TrustPending TrustState = "pending"
	TrustTrusted TrustState = "trusted"
	TrustRevoked TrustState = "revoked"
)

// Peer represents a known peer in the registry.
type Peer struct {
	// GitHubUsername of the peer (e.g., "alice").
	GitHubUsername string `toml:"github_username"`

	// Fingerprint is the SSH key fingerprint (SHA256:...).
	Fingerprint string `toml:"fingerprint"`

	// X25519Public is the DH public key (base64).
	X25519Public string `toml:"x25519_public"`

	// Trust is the current trust state.
	Trust TrustState `toml:"trust"`

	// FirstSeen is when this peer was first encountered.
	FirstSeen time.Time `toml:"first_seen"`

	// LastSeen is the last successful sync with this peer.
	LastSeen time.Time `toml:"last_seen"`

	// TrustedAt is when the peer was explicitly trusted.
	TrustedAt time.Time `toml:"trusted_at,omitempty"`

	// RevokedAt is when the peer was revoked.
	RevokedAt time.Time `toml:"revoked_at,omitempty"`
}

// Team represents a team of peers sharing .env files.
type Team struct {
	// ID is a unique team identifier (derived from creator's fingerprint + project).
	ID string `toml:"id"`

	// Name is a human-readable team name.
	Name string `toml:"name"`

	// CreatedBy is the fingerprint of the team creator.
	CreatedBy string `toml:"created_by"`

	// CreatedAt is when the team was created.
	CreatedAt time.Time `toml:"created_at"`

	// Members is the list of peer fingerprints in this team (storage only; Peers loaded separately).
	Members []string `toml:"members"`
}

// Validate checks if a peer's data is valid.
func (p *Peer) Validate() error {
	if p.Fingerprint == "" {
		return fmt.Errorf("peer has empty fingerprint")
	}
	if p.Trust == "" {
		return fmt.Errorf("peer has empty trust state")
	}
	return nil
}

// CanSync returns true if this peer is trusted and can participate in sync.
func (p *Peer) CanSync() bool {
	return p.Trust == TrustTrusted
}

// IsRevoked returns true if this peer has been revoked.
func (p *Peer) IsRevoked() bool {
	return p.Trust == TrustRevoked
}

// PromoteToTrusted moves the peer from pending to trusted.
func (p *Peer) PromoteToTrusted() error {
	if p.Trust != TrustPending && p.Trust != TrustUnknown {
		return fmt.Errorf("cannot trust peer in state %q", p.Trust)
	}
	p.Trust = TrustTrusted
	p.TrustedAt = time.Now()
	return nil
}

// Revoke moves the peer to revoked state.
func (p *Peer) Revoke() error {
	if p.Trust == TrustRevoked {
		return fmt.Errorf("peer already revoked")
	}
	p.Trust = TrustRevoked
	p.RevokedAt = time.Now()
	return nil
}

// StatusIcon returns a visual indicator for the peer's trust state.
func (p *Peer) StatusIcon() string {
	switch p.Trust {
	case TrustTrusted:
		return "✓"
	case TrustPending:
		return "?"
	case TrustRevoked:
		return "✗"
	default:
		return "·"
	}
}
