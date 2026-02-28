# EnvSync — Refined Architecture

> **Constraint:** $0 infrastructure budget | **Goal:** Million-dollar-ready project

---

## 1. Why Not WebRTC — The Critical Fix

The original bible specified WebRTC Data Channels. This is the single biggest architectural mistake.

| Issue | Impact |
|-------|--------|
| **Overkill** | WebRTC was built for browser media streaming. We send a 2KB `.env` file between two terminals. |
| **SDP Complexity** | Session Description Protocol negotiation requires a signaling server for offer/answer exchange — mandatory centralized dependency. |
| **ICE/STUN/TURN** | NAT traversal via ICE adds 3-8 seconds of connection setup. CLI users expect instant response. |
| **Dependency Bloat** | `pion/webrtc` pulls 50+ transitive deps. Binary balloons from ~8MB to ~25MB. |
| **Corporate Firewalls** | WebRTC's UDP-first approach fails in 15-20% of corporate networks. TURN fallback requires paying for a TURN server ($0 budget violated). |
| **Debugging Nightmare** | WebRTC error states are notoriously opaque. A CLI tool must have clear, actionable errors. |

### The Replacement: Three-Layer Hybrid Transport

```
┌──────────────────────────────────────────────────────────────────────┐
│                     EnvSync Transport Architecture                   │
│                                                                      │
│  LAYER 1: LAN Direct (fastest, zero config)                         │
│  ┌──────────┐   mDNS Discovery    ┌──────────┐                     │
│  │ Peer A   │◄──────────────────►│ Peer B   │                     │
│  │          │   TCP + Noise_XX    │          │                     │
│  └──────────┘   ~200ms total      └──────────┘                     │
│                                                                      │
│  LAYER 2: WAN Direct (TCP hole-punch via rendezvous)                │
│  ┌──────────┐                     ┌──────────┐                     │
│  │ Peer A   │──►┌──────────┐◄──│ Peer B   │                     │
│  │          │   │Rendezvous│   │          │                     │
│  │          │◄─────────────────►│          │                     │
│  └──────────┘   Direct TCP       └──────────┘                     │
│                  ~500ms, ~70% success rate                           │
│                                                                      │
│  LAYER 3: Relay Fallback (async, always works)                      │
│  ┌──────────┐                     ┌──────────┐                     │
│  │ Peer A   │──►┌──────────────┐──►│ Peer B   │                     │
│  │ encrypt  │   │ CF Workers   │   │ decrypt  │                     │
│  │ + upload │   │ KV Store     │   │ + download│                     │
│  └──────────┘   │ (E2E encrypted)│   └──────────┘                     │
│                  └──────────────┘                                    │
│                  72h TTL, zero knowledge                             │
│                                                                      │
│  Trust: SSH Ed25519 keys (existing)                                  │
│  Encryption: Noise_XX → ChaCha20-Poly1305 AEAD                      │
│  Auth: GitHub public key fingerprint verification                    │
└──────────────────────────────────────────────────────────────────────┘
```

**Fallback Logic (automatic, no user intervention):**
1. Try Layer 1 (mDNS LAN discovery). If peer found → direct TCP + Noise.
2. If no LAN peer → Try Layer 2 (TCP hole-punch via Cloudflare Worker rendezvous). ~70% success on non-symmetric NATs.
3. If hole-punch fails OR peer offline → Layer 3 (encrypted blob to Cloudflare KV relay). Peer pulls when online.

---

## 2. Technology Stack (Every Choice Justified)

| Component | Choice | Why This, Not That | Cost |
|-----------|--------|-------------------|------|
| **Language** | Go 1.22+ | Single static binary. No runtime. Cross-compiles to every OS/arch. Excellent `crypto/` stdlib. Rust was considered but Go's compilation speed and ecosystem maturity for CLI tools (cobra, bubbletea) win. | $0 |
| **CLI Framework** | `spf13/cobra` | Industry standard. Used by kubectl, gh, hugo. Automatic help generation, shell completions, subcommand nesting. | $0 |
| **Terminal UI** | `charmbracelet/bubbletea` + `lipgloss` + `bubbles` | The Elm Architecture in the terminal. Bubbletea provides the framework, Lipgloss provides CSS-like styling, Bubbles provides pre-built components (spinners, tables, progress bars). This is what makes "million-dollar" CLI UX possible. | $0 |
| **Transport (LAN)** | Raw TCP + `hashicorp/mdns` | mDNS broadcasts `_envsync._tcp.local` on port 5353. Automatic peer discovery on any local network. Zero configuration. Multicast may be blocked on some corporate WiFi — detection and clear error message built in. | $0 |
| **Transport (WAN)** | TCP hole-punch with `SO_REUSEADDR` | Recent research shows ~70% baseline TCP punch-through success rate. Both peers connect to rendezvous (Cloudflare Worker WebSocket), exchange endpoints, attempt simultaneous TCP open. No STUN/TURN needed for the direct path. | $0 |
| **Transport (Fallback)** | Cloudflare Workers + KV as encrypted relay | When direct fails or peer is offline. Client-side encryption before upload. Relay is zero-knowledge. Free tier: 100K req/day, 1GB KV storage. | $0 |
| **Encryption (Wire)** | Noise Protocol Framework via `flynn/noise`, Noise_XX pattern | Same protocol as WireGuard. Noise_XX provides mutual authentication without pre-shared keys. Three-message handshake: `→ e`, `← e, ee, s, es`, `→ s, se`. Forward secrecy via ephemeral X25519. ChaCha20-Poly1305 AEAD for post-handshake transport. | $0 |
| **Encryption (At-rest)** | XChaCha20-Poly1305 via `golang.org/x/crypto` | Extended nonce (192-bit) eliminates nonce-reuse risk for stored files. Key derived from SSH private key via HKDF-SHA256 with domain separation. | $0 |
| **Key Identity** | SSH Ed25519 keys → X25519 Noise static keys | Every developer already has `~/.ssh/id_ed25519` (GitHub requires it). We convert Ed25519 signing key to X25519 DH key using `crypto/ed25519` birational map. No new key generation. No new trust relationships. | $0 |
| **Peer Discovery (Remote)** | GitHub API: `https://github.com/{user}.keys` | Public endpoint, no auth needed, rate limit 60 req/hr (unauthenticated). Returns all SSH public keys. We filter for Ed25519. This is exactly how SSH verifies GitHub host keys. | $0 |
| **Config Format** | TOML via `pelletier/go-toml/v2` | Human-readable, well-specified, less ambiguous than YAML. Used by Rust (Cargo.toml), Go modules reference. | $0 |
| **Config Storage** | `~/.envsync/` directory | `config.toml` (settings), `peers/` (trusted peer keys), `store/` (encrypted .env versions), `audit.log` (local sync history). All encrypted at rest. | $0 |
| **CI/CD** | GitHub Actions | Free for public repos. Matrix builds across linux/darwin/windows × amd64/arm64. Automated releases on tag push. | $0 |
| **Release** | GoReleaser | Cross-compiles, creates GitHub Release, publishes Homebrew formula + Scoop manifest + Docker image automatically. Single `.goreleaser.yaml` config. | $0 |
| **Website** | Cloudflare Pages | Free static hosting. Custom domain (`envsync.dev`). Automatic HTTPS. Global CDN. | $0 |
| **Error Monitoring** | Sentry free tier | 5,000 events/month. Opt-in from CLI with `envsync telemetry on`. Crash reports + breadcrumbs. | $0 |
| **Analytics** | Cloudflare Workers Analytics Engine | Built into the relay Worker. Track: sync count, peer count, error rate. No third-party dependency. Privacy-first (no PII). | $0 |

**Total infrastructure cost: $0.00/month** (until ~2,000 active teams)

---

## 3. Security Architecture (Deep)

### 3.1 Cryptographic Identity Chain

```
Developer's Machine
  └── ~/.ssh/id_ed25519 (private key, passphrase-protected)
       │
       ├── Ed25519 Signing Key (used for SSH/git)
       │
       └── Birational Map (RFC 7748 / draft-ietf-openpgp-crypto-refresh)
            │
            └── X25519 Diffie-Hellman Key (used for Noise Protocol)
                 │
                 ├── Noise Static Public Key = peer identity
                 │    (announced to other peers, stored in their trust registry)
                 │
                 └── SHA-256(Noise Static Public Key) = fingerprint
                      (displayed to users for verification, e.g., SHA256:a3B7x...kQ9f)
```

**Why Ed25519 → X25519 conversion?**
- Ed25519 keys are signing keys (not suitable for DH key exchange)
- X25519 keys are DH keys (not suitable for signing)
- The birational map is a mathematically proven, lossless conversion between the two curve representations (Edwards ↔ Montgomery)
- Go's `crypto/ed25519` and `golang.org/x/crypto/curve25519` support this natively

### 3.2 Noise_XX Handshake (Step-by-Step)

The XX pattern is chosen because **neither peer knows the other's static key in advance** (first connection). After initial connection, keys are cached (TOFU — Trust On First Use, identical to SSH).

```
INITIATOR (Alice)                              RESPONDER (Bob)
─────────────────                              ────────────────
Generate ephemeral keypair (e_A)
                    ── Message 1: e_A ──►
                                               Generate ephemeral keypair (e_B)
                                               Compute: ee = DH(e_B, e_A)
                                               Encrypt static key s_B with ee
                                               Compute: es = DH(e_A, s_B)
                    ◄── Message 2: e_B, encrypted(s_B), auth_tag ──
Compute: ee = DH(e_A, e_B)
Decrypt s_B, verify
Compute: es = DH(e_A, s_B)
Encrypt static key s_A with derived key
Compute: se = DH(s_A, e_B)
                    ── Message 3: encrypted(s_A), auth_tag ──►
                                               Decrypt s_A, verify
                                               Compute: se = DH(s_A, e_B)

Both sides now have:
  - Shared symmetric keys for send/receive (split from chaining key)
  - Mutual authentication (both static keys verified)
  - Forward secrecy (ephemeral keys discarded)
  - Post-handshake: ChaCha20-Poly1305 AEAD transport
```

**Total handshake: 3 messages, 1.5 round-trips, ~50ms on LAN, ~150ms over WAN.**

### 3.3 At-Rest Encryption

```
~/.envsync/store/{project_hash}/{version}.enc

File format:
┌──────────────────────────────────────────┐
│ Magic bytes: "ENVSYNC\x01" (8 bytes)     │
│ Nonce: 24 bytes (XChaCha20 extended)     │
│ Ciphertext: variable length              │
│ Auth tag: 16 bytes (Poly1305)            │
└──────────────────────────────────────────┘

Key derivation:
  master_key = HKDF-SHA256(
    ikm = SSH_private_key_bytes,
    salt = "envsync-at-rest-v1",
    info = "local-storage-encryption"
  )
```

### 3.4 Relay Encryption (Zero-Knowledge)

When pushing to the async relay, the payload is encrypted **for each recipient** before leaving the sender's machine:

```
For each authorized peer:
  1. sender_ephemeral = X25519_keygen()
  2. shared_secret = X25519(sender_ephemeral.private, recipient.noise_static_public)
  3. encryption_key = HKDF-SHA256(shared_secret, "envsync-relay-v1", recipient_fingerprint)
  4. ciphertext = XChaCha20-Poly1305.Seal(encryption_key, nonce, plaintext_env_file)
  5. envelope = {
       sender_fingerprint,
       sender_ephemeral_public,  // so recipient can derive same shared_secret
       recipient_fingerprint,
       nonce,
       ciphertext,
       timestamp,
       version: 1
     }
  6. Upload envelope to relay: PUT /relay/{team_id}/{blob_id}
```

**The relay never sees:** the plaintext, the decryption key, or the SSH private keys. Even with full relay compromise + database dump, secrets are safe.

### 3.5 Complete Threat Model

| # | Threat | Attack Vector | Mitigation | Residual Risk |
|---|--------|--------------|------------|---------------|
| T1 | **Man-in-the-middle (LAN)** | ARP spoofing, rogue WiFi | Noise_XX mutual auth. Both peers verify static key fingerprints. TOFU on first connect (prompt user). Cached fingerprints reject impersonation on subsequent connects. | Same as SSH: first-connect TOFU window. Mitigated by fetching expected key from GitHub first. |
| T2 | **Man-in-the-middle (WAN)** | DNS poisoning of relay, BGP hijack | TLS to relay (Cloudflare handles cert). Noise_XX over the direct TCP connection. Even if relay is MITMed, payload is E2E encrypted. | None for data confidentiality. Availability could be impacted. |
| T3 | **Compromised relay** | Cloudflare account hack, insider threat, legal compulsion | All blobs E2E encrypted client-side. Relay stores only: `(blob_id, encrypted_bytes, TTL, sender_fp, recipient_fp)`. No decryption keys on server. | Metadata exposure (who syncs with whom, when). Acceptable for free tier. Enterprise tier: on-premise relay eliminates this. |
| T4 | **Stolen laptop** | Physical theft, unattended machine | `.env` store encrypted at rest with key derived from SSH private key. SSH private key should be passphrase-protected. `envsync init` warns if no passphrase detected. | If SSH key has no passphrase AND attacker has disk access → compromise. Out of scope (same as SSH threat model boundary). |
| T5 | **GitHub account compromise** | Phisher adds their SSH key to victim's GitHub | On `envsync invite @user`, we fetch GitHub keys. If attacker added a key, we'd fetch it. Mitigation: on first connect, display full fingerprint + require interactive confirmation. Warn if GitHub has multiple Ed25519 keys. | Reduced to social engineering window. Same risk as SSH known_hosts on first connect. |
| T6 | **Replay attack on relay** | Attacker re-uploads a previously captured blob | Each blob has: unique nonce, monotonic sequence number per team, timestamp. Recipients reject: duplicate nonce, out-of-order sequence, timestamp > 72h old. | None. |
| T7 | **Denial of service** | Flood relay with requests | Cloudflare's built-in DDoS protection (free). Worker-level rate limiting: 10 req/min per IP per team. KV TTLs auto-clean old data. | Targeted DoS against specific team's relay namespace. Low impact (fallback to LAN sync). |
| T8 | **Supply chain attack** | Compromised Go dependency | Only 8 direct dependencies, all widely audited. `go.sum` integrity verification. Dependabot alerts. GoReleaser reproducible builds. | Standard supply chain risk. Mitigated by minimal dependency surface. |
| T9 | **Malicious peer** | Authorized peer sends crafted .env to inject malicious values | `.env` files are parsed and validated before writing. Dangerous patterns warned: URLs to unknown hosts, abnormally long values, binary content. `envsync diff` always shown before applying changes (opt-out, not opt-in). | Authorized peer is trusted by definition. Social trust boundary. |
| T10 | **Timing side-channel** | Attacker measures relay response times to infer sync patterns | All relay operations are constant-time where possible. Blob sizes padded to nearest 1KB boundary. | Minimal metadata leakage. Acceptable. |

---

## 4. The .env Parser (Specification-Grade)

There is no formal `.env` specification. EnvSync's parser becomes the reference implementation. It must handle every real-world `.env` file correctly.

### 4.1 Parsing Rules

```
# BASIC KEY=VALUE
DATABASE_URL=postgres://localhost:5432/mydb

# SPACES AROUND = ARE TRIMMED
API_KEY = sk_test_12345

# EXPORT PREFIX (IGNORED)
export SECRET_KEY=my_secret

# COMMENTS
# This is a comment
DATABASE_URL=postgres://localhost:5432/mydb  # inline comment

# SINGLE QUOTES: VERBATIM (no interpolation, no escapes except \')
PASSWORD='p@$$w0rd!#with"special'

# DOUBLE QUOTES: ESCAPE SEQUENCES + INTERPOLATION
GREETING="Hello\nWorld"           # \n becomes actual newline
FULL_URL="https://${HOST}:${PORT}/api"  # variable interpolation

# MULTILINE (DOUBLE QUOTES)
PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"

# EMPTY VALUES
EMPTY_VAR=
ALSO_EMPTY=""

# UNQUOTED VALUES: TRIMMED, NO ESCAPES
SIMPLE=hello world   # value is "hello world" (trailing comment stripped)
```

### 4.2 Parser Behavior Matrix

| Input | Parsed Value | Notes |
|-------|-------------|-------|
| `KEY=value` | `value` | Basic, trimmed |
| `KEY = value ` | `value` | Spaces around `=` and trailing trimmed |
| `KEY="value"` | `value` | Quotes stripped |
| `KEY='value'` | `value` | Quotes stripped, literal content |
| `KEY="hello\nworld"` | `hello↵world` | Escape sequences processed |
| `KEY='hello\nworld'` | `hello\nworld` | Literal `\n`, not newline |
| `KEY=` | `` (empty) | Empty value |
| `KEY=""` | `` (empty) | Explicit empty |
| `KEY="has \"quotes\""` | `has "quotes"` | Escaped quotes |
| `export KEY=value` | `value` | `export` prefix stripped |
| `# comment` | (skipped) | Full-line comment |
| `KEY=value # comment` | `value` | Inline comment stripped (unquoted) |
| `KEY="value # not comment"` | `value # not comment` | `#` inside quotes preserved |
| `KEY="${OTHER}_suffix"` | `${OTHER}_suffix` | Variable interpolation (optional, configurable) |
