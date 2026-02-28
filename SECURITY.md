# Security Policy

## Reporting Vulnerabilities

Report security vulnerabilities to: **security@envsync.dev**

We will respond within 48 hours and aim to provide a patch within 7 days for critical issues.

**Do not** open public GitHub issues for security vulnerabilities.

## Cryptographic Design

| Layer | Primitive | Purpose |
|-------|-----------|---------|
| Identity | Ed25519 (SSH keys) | User identity, no accounts needed |
| Key Exchange | X25519 (Curve25519 ECDH) | Derive shared secrets |
| Channel Encryption | Noise_XX (XChaCha20-Poly1305) | LAN peer-to-peer transport |
| At-Rest Encryption | XChaCha20-Poly1305 + HKDF-SHA256 | Local encrypted backups |
| Relay Encryption | Ephemeral X25519 + XChaCha20-Poly1305 | Per-recipient relay blobs |
| Request Auth | Ed25519 signatures (ES-SIG) | Relay API authentication |
| Key Derivation | HKDF-SHA256 | Derive encryption keys from shared secrets |

## Trust Model

- **TOFU (Trust On First Use)**: First connection prompts fingerprint verification
- **Peer Registry**: Trusted peers stored locally in TOML
- **Trust States**: Unknown → Pending → Trusted → Revoked
- **Team Scoping**: Peers are scoped to teams, preventing cross-team access

## Zero-Knowledge Relay

The relay server **never** has access to plaintext secrets:

1. All blobs are encrypted **client-side** before upload
2. Each blob uses an **ephemeral ECDH key** unique to the recipient
3. The relay stores only opaque ciphertext with a 72-hour TTL
4. Blob metadata (sender, recipient, size) is visible to the relay for routing

## What We Don't Protect Against

- **Compromised SSH keys**: If an attacker has your private SSH key, they can impersonate you
- **Compromised endpoint**: If your machine is compromised, the .env file is readable in memory
- **Relay metadata**: The relay can see who is syncing with whom (but not what)
- **Denial of service**: The relay can refuse to relay blobs

## Dependencies

All cryptographic operations use well-audited libraries:

- `golang.org/x/crypto` — Official Go crypto extensions (Ed25519, X25519, XChaCha20-Poly1305, HKDF)
- `github.com/flynn/noise` — Noise Protocol Framework implementation
- No custom cryptographic primitives are used
