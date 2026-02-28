# EnvSync — Project Structure & Build Plan

---

## 8. Project Structure (Every File Justified)

```
envsync/
├── cmd/                              # CLI commands (cobra)
│   ├── root.go                       # Root command: global flags, config loading, version banner
│   ├── init.go                       # `envsync init`: SSH key detection, X25519 derivation, config creation
│   ├── invite.go                     # `envsync invite @user`: GitHub key fetch, invite token gen, relay POST
│   ├── join.go                       # `envsync join <code>`: invite retrieval, team registration, peer storage
│   ├── push.go                       # `envsync push`: orchestrates LAN→WAN→relay fallback chain
│   ├── pull.go                       # `envsync pull`: checks relay, accepts WAN, receives LAN. Diff + confirm.
│   ├── diff.go                       # `envsync diff`: three-way diff (local vs synced vs remote)
│   ├── peers.go                      # `envsync peers`: table of team members with online/offline/relay status
│   ├── revoke.go                     # `envsync revoke @user`: remove from team, re-key if needed
│   ├── status.go                     # `envsync status`: current sync state, pending incoming, last push time
│   ├── backup.go                     # `envsync backup`: encrypted local snapshot to ~/.envsync/store/
│   ├── restore.go                    # `envsync restore`: list versions, restore selected
│   ├── audit.go                      # `envsync audit`: display local sync log with timestamps and peer IDs
│   ├── config_cmd.go                 # `envsync config`: view/set config values
│   └── version.go                    # `envsync version`: semver + git commit + build date
│
├── internal/                         # Private packages (not importable by external code)
│   ├── config/
│   │   ├── config.go                 # Config struct: team settings, relay URL, preferences
│   │   │                              # Load from TOML, merge with env vars, validate
│   │   ├── paths.go                  # XDG-aware path resolution: ~/.envsync/ on all platforms
│   │   │                              # Windows: %APPDATA%\envsync\
│   │   │                              # macOS: ~/Library/Application Support/envsync/
│   │   │                              # Linux: ~/.config/envsync/ (if XDG_CONFIG_HOME set)
│   │   └── migrate.go                # Config schema migration between versions
│   │
│   ├── crypto/
│   │   ├── keys.go                   # SSH key reading (OpenSSH format, PEM, PKCS8)
│   │   │                              # Ed25519 → X25519 birational conversion
│   │   │                              # Fingerprint computation (SHA-256, base64 display)
│   │   │                              # Passphrase detection and warning
│   │   ├── noise.go                  # Noise_XX handshake over net.Conn
│   │   │                              # Config: CipherSuite(X25519, ChaCha20-Poly1305, SHA-256)
│   │   │                              # Returns: authenticated, encrypted ReadWriteCloser
│   │   ├── encrypt.go                # At-rest encryption: XChaCha20-Poly1305
│   │   │                              # HKDF key derivation from SSH key
│   │   │                              # File format: magic + nonce + ciphertext + tag
│   │   ├── envelope.go               # Relay envelope: per-recipient encryption
│   │   │                              # Ephemeral ECDH + HKDF + XChaCha20-Poly1305
│   │   │                              # Envelope struct: serialization/deserialization
│   │   └── signature.go              # Ed25519 signing for relay request authentication
│   │                                  # Canonical request format, timestamp inclusion
│   │
│   ├── discovery/
│   │   ├── mdns.go                   # mDNS service advertisement and discovery
│   │   │                              # Service: _envsync._tcp.local
│   │   │                              # TXT records: fingerprint, team, version
│   │   │                              # Handles: multiple interfaces, IPv4/IPv6
│   │   │                              # Timeout: 2s for discovery, 500ms for response
│   │   └── github.go                 # GitHub public key fetching and caching
│   │                                  # GET https://github.com/{user}.keys
│   │                                  # Parse ssh-ed25519 keys, ignore RSA/ECDSA
│   │                                  # Cache in memory (5 min) and on disk (1 hour)
│   │                                  # Handle: rate limits, 404 (user not found), multiple keys
│   │
│   ├── envfile/
│   │   ├── parser.go                 # Full .env parser (spec in 01_ARCHITECTURE.md §4)
│   │   │                              # Handles: comments, quotes, multiline, export prefix,
│   │   │                              # escape sequences, variable interpolation (optional)
│   │   │                              # Preserves: ordering, comments, blank lines (round-trip safe)
│   │   ├── writer.go                 # Write .env file preserving original formatting
│   │   │                              # Smart quoting: only quote when necessary
│   │   ├── diff.go                   # Three-way diff: local vs base vs remote
│   │   │                              # Categories: added, removed, modified, unchanged
│   │   │                              # Value masking: show first 4 + last 2 chars of secrets
│   │   ├── merge.go                  # Merge strategies:
│   │   │                              # - overwrite: remote wins entirely
│   │   │                              # - keep-local: local wins, add remote-only vars
│   │   │                              # - interactive: prompt per-variable
│   │   │                              # - three-way: auto-merge non-conflicting, prompt conflicts
│   │   └── validate.go               # Validation rules:
│   │                                  # - No duplicate keys
│   │                                  # - No binary content
│   │                                  # - Warn on suspicious patterns (e.g., base64-encoded private keys
│   │                                  #   without quotes, URLs to unknown hosts)
│   │                                  # - Max value length (64KB per variable)
│   │
│   ├── relay/
│   │   ├── client.go                 # HTTP client for relay API
│   │   │                              # Base URL from config (default: relay.envsync.dev)
│   │   │                              # Automatic retry with exponential backoff (3 attempts)
│   │   │                              # Request signing (see crypto/signature.go)
│   │   │                              # Connection pooling, keep-alive, timeout: 10s
│   │   ├── invites.go                # Invite CRUD: create, retrieve, consume
│   │   │                              # Token generation: 6 random words from curated wordlist
│   │   │                              # Token hashing before transmission (relay never sees raw token)
│   │   ├── blobs.go                  # Encrypted blob upload/download/delete
│   │   │                              # Upload: encrypt → sign → PUT
│   │   │                              # Download: GET → verify sender → decrypt
│   │   │                              # List pending: GET with recipient filter
│   │   ├── teams.go                  # Team membership management via relay
│   │   │                              # Add/remove members, list members
│   │   └── signal.go                 # WebSocket client for hole-punch signaling
│   │                                  # Connect to Durable Object room for team
│   │                                  # Exchange: public IP, port, NAT type
│   │                                  # Timeout: 5s for peer to appear
│   │
│   ├── store/
│   │   ├── store.go                  # Local encrypted version store
│   │   │                              # ~/.envsync/store/{project_hash}/
│   │   │                              # Each version: {sequence}_{timestamp}.enc
│   │   │                              # Rotate: keep last N versions (configurable, default 10)
│   │   └── history.go                # Version listing, comparison, restoration
│   │                                  # Metadata: who pushed, when, how many variables changed
│   │
│   ├── sync/
│   │   ├── orchestrator.go           # Core sync logic: manages the fallback chain
│   │   │                              # LAN (mDNS) → WAN (hole-punch) → Relay (async)
│   │   │                              # Parallel discovery: try LAN + check relay simultaneously
│   │   │                              # Timeout management per layer
│   │   ├── push.go                   # Push-specific logic:
│   │   │                              # Read .env → discover peers → connect → encrypt → send
│   │   │                              # For offline peers: encrypt for their key → upload to relay
│   │   ├── pull.go                   # Pull-specific logic:
│   │   │                              # Check relay for pending → listen on LAN → accept WAN
│   │   │                              # Diff + confirmation before applying
│   │   └── resolver.go               # Conflict resolution engine
│   │                                  # Detect: same variable modified by multiple peers since last sync
│   │                                  # Strategies: last-writer-wins, manual, three-way-merge
│   │
│   ├── transport/
│   │   ├── direct.go                 # Direct TCP connection (LAN or post-hole-punch)
│   │   │                              # Dial with timeout (2s LAN, 5s WAN)
│   │   │                              # Wrap with Noise_XX (returns encrypted conn)
│   │   ├── listener.go               # TCP listener for incoming connections
│   │   │                              # Bind to port 7733 (configurable)
│   │   │                              # Accept + Noise_XX handshake
│   │   │                              # Verify: is the connecting peer in our trust registry?
│   │   └── holepunch.go              # TCP hole-punch implementation
│   │                                  # 1. Detect local NAT type (STUN query to relay)
│   │                                  # 2. Signal endpoint to peer via WebSocket
│   │                                  # 3. Simultaneous TCP connect with SO_REUSEADDR
│   │                                  # 4. Race: first successful conn wins
│   │                                  # 5. Timeout → relay fallback
│   │
│   ├── peer/
│   │   ├── peer.go                   # Peer struct: github username, fingerprint, public key,
│   │   │                              # trust status (trusted/pending/revoked), added timestamp
│   │   ├── registry.go               # Local peer storage:
│   │   │                              # ~/.envsync/teams/{team_id}/peers/{fingerprint}.toml
│   │   │                              # CRUD operations, trust state transitions
│   │   └── team.go                   # Team struct: ID, name, members, creation time
│   │                                  # Team discovery from project directory (.envsync.toml)
│   │
│   ├── audit/
│   │   ├── logger.go                 # Append-only local audit log
│   │   │                              # Format: JSONL (one JSON object per line)
│   │   │                              # Events: push, pull, invite, join, revoke, conflict
│   │   │                              # Fields: timestamp, actor, action, file, variables_changed
│   │   └── viewer.go                 # Pretty-print audit log with filters
│   │                                  # Filter by: peer, action, date range
│   │
│   └── ui/
│       ├── theme.go                  # Color palette and style definitions
│       │                              # Brand colors: primary=#8B5CF6 (violet), accent=#06B6D4 (cyan)
│       │                              # Lipgloss styles for headers, tables, errors, success
│       ├── spinner.go                # Animated spinner with stage messages
│       │                              # "Scanning local network...", "Encrypting...", etc.
│       ├── table.go                  # Formatted tables (peers, diff, audit)
│       │                              # Auto-sizing columns, truncation, alignment
│       ├── confirm.go                # Interactive confirmation prompts
│       │                              # Y/n/d(iff)/h(elp) with keyboard navigation
│       ├── diff_view.go              # Rich diff display with sidebyside columns
│       │                              # Color-coded: green=added, red=removed, yellow=modified
│       │                              # Value masking for sensitive data
│       └── banner.go                 # "✦ EnvSync v1.0.0" branded header
│                                      # Contextual: shows team name + peer count in header
│
├── relay/                            # Cloudflare Worker (separate deployment)
│   ├── wrangler.toml                 # Worker config: name, KV bindings, DO bindings, routes
│   ├── package.json                  # Dependencies: hono (router), @cloudflare/workers-types
│   ├── tsconfig.json                 # TypeScript config
│   ├── src/
│   │   ├── index.ts                  # Worker entry: Hono app, route registration, error handler
│   │   ├── middleware/
│   │   │   ├── auth.ts               # Request signature verification middleware
│   │   │   ├── ratelimit.ts          # Per-IP, per-team rate limiting via KV
│   │   │   └── cors.ts               # CORS for future web dashboard
│   │   ├── routes/
│   │   │   ├── invites.ts            # POST/GET/DELETE /invites/:hash
│   │   │   ├── relay.ts              # PUT/GET/DELETE /relay/:team/:blob, GET /relay/:team/pending
│   │   │   ├── teams.ts              # GET/PUT/DELETE /teams/:team/members/:user
│   │   │   └── health.ts             # GET /health — uptime, KV usage stats
│   │   ├── durable/
│   │   │   └── signaling-room.ts     # Durable Object: per-team WebSocket room
│   │   │                              # Hibernation API for cost savings
│   │   │                              # Exchange: {peer_fp, public_ip, port, nat_type}
│   │   └── types.ts                  # Shared TypeScript types for all routes
│   └── test/
│       ├── invites.test.ts           # Invite flow tests (Miniflare)
│       ├── relay.test.ts             # Blob CRUD tests
│       └── auth.test.ts              # Signature verification tests
│
├── .envsync.toml                     # Per-project config (committed to git)
│                                      # team_id, default_file, sync_strategy
│
├── scripts/
│   ├── install.sh                    # curl -fsSL envsync.dev/install | sh
│   │                                  # OS/arch detection, binary download, PATH setup
│   └── install.ps1                   # PowerShell installer for Windows
│
├── .github/
│   └── workflows/
│       ├── ci.yml                    # PR checks: lint, test, build (matrix: os × arch)
│       ├── release.yml               # Tag push → GoReleaser → GitHub Release + Homebrew + Scoop
│       └── security.yml              # Weekly: gosec, govulncheck, dependency audit
│
├── .goreleaser.yml                   # GoReleaser config for cross-platform builds
├── go.mod                            # Go module: github.com/envsync/envsync
├── go.sum                            # Dependency checksums
├── main.go                           # Entry point: cmd.Execute()
├── Makefile                          # build, test, test-integration, lint, release-dry-run
├── LICENSE                           # MIT
├── SECURITY.md                       # Vulnerability reporting policy
├── CONTRIBUTING.md                   # Contribution guidelines
└── README.md                         # The product demo (terminal GIFs, quick-start)
```

**File count: ~70 Go files, ~10 TypeScript files, ~10 config/doc files = ~90 files total.**  
**Estimated lines of code: ~8,000-10,000 Go + ~1,500 TypeScript = ~10,000-12,000 total.**

---

## 9. Build Plan (10-Week Sprint, Day-by-Day)

### Phase 0: Core Cryptographic Engine (Week 1)

**Goal:** Two processes on localhost can exchange an encrypted `.env` file.

| Day | Focus | Deliverables | Tests |
|-----|-------|-------------|-------|
| D1 | **Project scaffold** | `go mod init`, cobra CLI skeleton, `main.go`, Makefile (`build`, `test`, `lint`), GitHub repo, `.github/workflows/ci.yml`, `.gitignore`, LICENSE (MIT), `README.md` stub | CI pipeline passing ✅ |
| D2 | **SSH key reading** | `internal/crypto/keys.go`: Read OpenSSH Ed25519 private/public key. Handle: passphrase-protected (detect + warn), PEM format, missing key. Ed25519 → X25519 birational conversion. Fingerprint computation. | Unit tests: read valid key, detect passphrase, handle missing file, correct X25519 output, fingerprint matches OpenSSH format |
| D3 | **.env parser** | `internal/envfile/parser.go`, `writer.go`: Full parser per spec §4. Round-trip: `parse(write(parse(file))) == parse(file)`. Preserves comments, ordering, blank lines. | 30+ test cases covering every parser rule in §4.2 matrix. Edge cases: empty file, BOM, CRLF, no trailing newline |
| D4 | **Noise protocol** | `internal/crypto/noise.go`: Noise_XX handshake over `net.Conn`. Uses `flynn/noise` with `X25519 + ChaCha20-Poly1305 + SHA-256`. Returns authenticated `io.ReadWriteCloser`. Verifies remote static key fingerprint. | Integration test: two goroutines, TCP loopback, handshake, bidirectional encrypted messaging. Negative test: reject unknown peer |
| D5 | **At-rest encryption** | `internal/crypto/encrypt.go`: XChaCha20-Poly1305 file encryption. HKDF key derivation. File format with magic bytes. `internal/store/store.go`: version store in `~/.envsync/store/`. | Round-trip: encrypt(decrypt(plaintext)) == plaintext. Corrupt file detection. Version rotation (keep last N) |

### Phase 0 continued: Transport Layer (Week 2)

| Day | Focus | Deliverables | Tests |
|-----|-------|-------------|-------|
| D6 | **mDNS discovery** | `internal/discovery/mdns.go`: Advertise `_envsync._tcp.local` with TXT records (fingerprint, team, version). Discover peers on LAN with 2s timeout. Handle multiple network interfaces. | Integration test: two processes, mDNS discovery on localhost. Verify TXT records parsed correctly |
| D7 | **TCP transport + push** | `internal/transport/direct.go`, `listener.go`. `cmd/push.go`: Read .env → discover peers (mDNS) → TCP connect → Noise handshake → send encrypted payload. `internal/sync/push.go` for orchestration. | End-to-end: Process A pushes .env, Process B receives. Verify content matches. Test on localhost (two terminals) |
| D8 | **Pull command + config** | `cmd/pull.go`, `cmd/init.go`. `internal/config/config.go`, `paths.go`. `envsync init` creates config. `envsync pull` listens for incoming push or checks relay. | `envsync init` creates `~/.envsync/config.toml`. `envsync pull` receives from `envsync push` on localhost |
| D9 | **Peer registry + trust** | `internal/peer/peer.go`, `registry.go`, `team.go`. TOFU model: first connect prompts for fingerprint verification. Subsequent connects auto-verify cached fingerprint. Reject unknown peers. | Trust state machine: unknown→pending→trusted→revoked. Storage persistence across restarts |
| D10 | **Cross-platform testing** | Test on Windows (PowerShell), macOS (zsh), Linux (bash). Fix path handling (`filepath.Join`, not hardcoded `/`). Fix mDNS on Windows (may need `dns-sd` fallback). Fix terminal colors on Windows (enable VT100). | All existing tests pass on Windows, macOS, Linux. Manual test: two-machine LAN push/pull |

---

### Phase 1: Relay + Remote Sync (Weeks 3-4)

**Goal:** Two developers on different networks can sync `.env` files, even asynchronously.

| Day | Focus | Deliverables | Tests |
|-----|-------|-------------|-------|
| D11 | **CF Worker scaffold** | `relay/` directory. Wrangler project. Hono router. Health endpoint. KV binding. Deploy to `relay.envsync.dev` (or temporary subdomain). | `wrangler dev` running locally. `GET /health` returns 200 |
| D12 | **Invite system (server)** | `relay/src/routes/invites.ts`: POST (create invite in KV, 24h TTL), GET (retrieve), DELETE (consume). Token hash as key (relay never stores raw token). | Miniflare tests: CRUD lifecycle, TTL expiry, duplicate prevention |
| D13 | **Invite system (client)** | `cmd/invite.go`, `cmd/join.go`. `internal/discovery/github.go`: Fetch keys from `github.com/{user}.keys`. `internal/relay/invites.go`: HTTP client for invite endpoints. Full flow: invite → share code → join → team created. | End-to-end: `envsync invite @testuser` creates invite on relay. `envsync join <code>` consumes invite. Both processes have each other in peer registry |
| D14 | **Relay auth middleware** | `relay/src/middleware/auth.ts`: Ed25519 signature verification per §6.2. `internal/crypto/signature.go`: Sign requests. `internal/relay/client.go`: Auto-sign all requests. | Auth: valid signature accepted, expired timestamp rejected, wrong key rejected, tampered body rejected |
| D15 | **Blob storage (server)** | `relay/src/routes/relay.ts`: PUT (store encrypted blob, 72h TTL), GET (download), DELETE (cleanup), GET pending (list for recipient). Rate limiting via KV. | Miniflare: full blob lifecycle. Rate limit enforcement. TTL expiry. Pending list filters correctly |
| D16 | **Blob storage (client)** | `internal/crypto/envelope.go`: Per-recipient encryption (ephemeral ECDH). `internal/relay/blobs.go`: Upload/download encrypted blobs. Integrate into push/pull commands. | Round-trip: push encrypts for specific recipient, upload succeeds, pull downloads + decrypts correctly. Wrong recipient cannot decrypt |
| D17 | **Hole-punch signaling** | `relay/src/durable/signaling-room.ts`: Durable Object with hibernating WebSockets. Per-team room. Exchange endpoint info. `internal/relay/signal.go`: WebSocket client. `internal/transport/holepunch.go`: TCP simultaneous open with `SO_REUSEADDR`. | Signaling: two clients connect to room, receive each other's endpoints. Hole-punch: test on two machines on different networks (may need manual verification) |
| D18 | **Sync orchestrator** | `internal/sync/orchestrator.go`: Full fallback chain. Try LAN mDNS (2s) → try WAN hole-punch (5s) → relay upload. Parallel: check relay for pending while doing LAN discovery. | Integration test: Mock each transport layer. Verify fallback chain executes in correct order with correct timeouts |
| D19 | **Team management** | `relay/src/routes/teams.ts`: Member CRUD. `cmd/peers.go`: Display team members. `cmd/revoke.go`: Remove peer. `internal/peer/team.go`: Team-level operations. | Add member, list members, revoke member. Revoked member rejected on next sync attempt |
| D20 | **End-to-end testing** | Full scenario test across two real machines on different networks: init → invite → join → push (LAN) → push (relay) → pull (relay). Document any issues. Fix. Retest. | All scenarios pass. Connection times within targets (LAN <500ms, relay <3s) |

---

### Phase 2: Polish & Launch (Weeks 5-6)

**Goal:** Million-dollar-feel CLI. Ready for public launch.

| Day | Focus | Deliverables | Tests |
|-----|-------|-------------|-------|
| D21 | **Terminal UI: theme + spinner** | `internal/ui/theme.go`, `spinner.go`, `banner.go`. Brand colors. Lipgloss styles for all output types. Animated spinner with stage messages. Detect terminal capabilities (color support, width). | Visual inspection on multiple terminals (iTerm2, Windows Terminal, VS Code integrated terminal). Fallback for no-color terminals |
| D22 | **Terminal UI: tables + diff** | `internal/ui/table.go`, `diff_view.go`. Auto-sizing columns. Color-coded diff. Value masking for secrets (-show only first 4 + last 2 chars). Sidebyside layout for narrow terminals. | Diff display matches mockups in §7.3. Tables render correctly at 80, 120, 200 column widths |
| D23 | **Terminal UI: confirm + interactive merge** | `internal/ui/confirm.go`. Interactive per-variable merge: accept/reject/edit each change. Keyboard navigation. Preview before apply. | Interactive merge flow: present 5 changes, accept 3, reject 1, edit 1. Verify .env output is correct |
| D24 | **Multiple .env files** | `--file` flag on push/pull/diff. Support: `.env`, `.env.local`, `.env.development`, `.env.production`, `.env.test`, custom names. Per-file version history. | Push/pull `.env.production` without affecting `.env`. Diff works on non-default files |
| D25 | **Conflict resolution** | `internal/sync/resolver.go`, `internal/envfile/merge.go`. Detect: same variable modified by two peers since last sync. Three-way merge with base version. Interactive conflict resolution UI. | Create conflict: Alice changes `API_KEY`, Bob changes `API_KEY` differently. Merge tool presents both values + base. Resolution applied correctly |
| D26 | **Error UX overhaul** | Every error path reviewed. Each error: message + cause + suggestion + URL. No stack traces. No Go-internal errors exposed. Wrap all errors with `fmt.Errorf("descriptive message: %w", err)`. | Trigger every known error condition. Verify output matches error UX design in §7.3 |
| D27 | **Backup + restore** | `cmd/backup.go`, `cmd/restore.go`. `envsync backup` → encrypted snapshot. `envsync restore` → list versions with metadata, restore selected. | Backup creates encrypted file. Restore recovers exact content. List shows correct metadata |
| D28 | **Audit log** | `cmd/audit.go`, `internal/audit/logger.go`, `viewer.go`. JSONL append-only log. Events: push, pull, invite, join, revoke, conflict-resolved. Pretty-print with filters. | Every sync event logged. `envsync audit --last 5` shows correct events. Filter by peer works |
| D29 | **GoReleaser + distribution** | `.goreleaser.yml`: cross-compile (linux/darwin/windows × amd64/arm64 = 6 targets). `.github/workflows/release.yml`: tag push triggers release. Homebrew tap repo. Scoop bucket. `scripts/install.sh` + `install.ps1`. | `goreleaser --snapshot` produces all 6 binaries. Install script works on fresh Ubuntu, macOS, Windows |
| D30 | **README + docs site** | `README.md`: terminal GIFs (recorded with `vhs`), quick-start (3 steps), security summary, badge wall, FAQ. `docs/` deployed to Cloudflare Pages at `envsync.dev`. | README renders correctly on GitHub. All links work. GIFs play. Quick-start instructions verified on clean machine |

---

### Phase 3: Growth Features (Weeks 7-10)

| Week | Focus | Key Deliverables |
|------|-------|-----------------|
| **W7** | **GitHub Actions integration** | `envsync/action` GitHub Action. Injects env vars from trust graph into CI workflow. Uses service account key. `.github/marketplace.yml`. Listed on GitHub Marketplace (free). |
| **W8** | **Paid tier + Stripe** | Stripe Checkout integration. Self-serve upgrade in CLI: `envsync upgrade`. License key stored in relay KV per team. Feature gating: check tier on relay responses. Free tier limits enforced: 3 peers, 10 relay blobs/day. Paid tier unlocks: unlimited peers, unlimited relay, 30-day cloud history. |
| **W9** | **VS Code extension** | Extension: status bar indicator (synced ✓ / out of date ⚠). Command palette: "EnvSync: Push", "EnvSync: Pull", "EnvSync: Diff". Sidebar panel: peer list, sync history. Uses CLI binary under the hood (no reimplementation). |
| **W10** | **Hardening + launch** | Security audit (self + community review). Performance benchmarks. Load test relay (k6 scripts). Fix all P0/P1 issues. Prepare HN post. Prepare dev.to article. Prepare Twitter thread with GIFs. Set up Discord server. Configure Sentry. **Ship it.** |

---

## 10. Go Dependencies (Final, Locked)

| Package | Version | Purpose | License | Weekly Downloads | Why Not Alternatives |
|---------|---------|---------|---------|-----------------|---------------------|
| `github.com/spf13/cobra` | v1.8+ | CLI framework | Apache-2.0 | 1M+ | Industry standard. urfave/cli is viable but cobra has better completion generation and documentation features. |
| `github.com/charmbracelet/bubbletea` | v1.2+ | Terminal UI framework | MIT | 300K+ | Only production-grade TUI framework in Go with the Elm Architecture. survey/promptui are prompt-only, not full TUI. |
| `github.com/charmbracelet/lipgloss` | v1.0+ | Terminal styling | MIT | 500K+ | Companion to bubbletea. CSS-like API. Auto-downsample colors for terminal capability. |
| `github.com/charmbracelet/bubbles` | v0.20+ | Prebuilt TUI components | MIT | 200K+ | Spinners, tables, text input, progress bars. Saves ~2,000 lines of custom code. |
| `github.com/flynn/noise` | latest | Noise Protocol Framework | BSD-3 | 50K+ | Only maintained Go Noise library. Used by Perlin Network in production. Alternative: hand-roll (insane). |
| `github.com/hashicorp/mdns` | v1.0+ | mDNS discovery | MIT | 100K+ | HashiCorp quality. Used in Consul. Alternative: grandcat/zeroconf (less maintained). |
| `github.com/pelletier/go-toml/v2` | v2.2+ | TOML parsing | Apache-2.0 | 500K+ | Best TOML v1.0 support in Go. Alternative: BurntSushi/toml (slower, less complete). |
| `golang.org/x/crypto` | latest | SSH key reading, XChaCha20-Poly1305, HKDF, curve25519 | BSD-3 | Official | Official Go crypto extension. No alternative — this IS the standard. |

**Total: 8 direct dependencies.** Transitive dependency count: ~15 (most from charmbracelet ecosystem).

**Binary size estimate:** ~12MB (stripped, without UPX). With UPX: ~5MB. Acceptable for a CLI tool.

---

## 11. Distribution Strategy ($0)

| Channel | Command | Setup | Maintenance |
|---------|---------|-------|-------------|
| **Homebrew (macOS/Linux)** | `brew install envsync/tap/envsync` | Create `homebrew-envsync` GitHub repo. GoReleaser auto-pushes formula on release. | Zero — GoReleaser handles updates |
| **Scoop (Windows)** | `scoop bucket add envsync https://github.com/envsync/scoop-envsync && scoop install envsync` | Create `scoop-envsync` GitHub repo. GoReleaser auto-pushes manifest. | Zero — GoReleaser handles updates |
| **Go install** | `go install github.com/envsync/envsync@latest` | Free — comes with being a Go module | Zero |
| **curl install (any)** | `curl -fsSL https://envsync.dev/install \| sh` | `scripts/install.sh`: detect OS/arch, download from GitHub Releases, place in `/usr/local/bin/`, verify checksum | Update script on new platform support |
| **PowerShell (Windows)** | `irm https://envsync.dev/install.ps1 \| iex` | `scripts/install.ps1`: similar to above for Windows | Mirrors install.sh logic |
| **GitHub Releases** | Direct download | GoReleaser creates releases on tag push | Zero — fully automated |
| **Docker** | `docker run envsync/envsync push` | Multi-arch Dockerfile in repo. GitHub Actions builds + pushes to Docker Hub (free for OSS) | Auto-built on release |

### The README Embed (Core Growth Mechanic)

Every repo that uses EnvSync should have this in their README:

```markdown
## 🔧 Environment Setup

1. Install EnvSync: `brew install envsync/tap/envsync` (or [other methods](https://envsync.dev/install))
2. Get your environment variables:
   ```bash
   envsync join <ask-your-team-lead-for-the-code>
   envsync pull
   ```
3. Start developing: `npm run dev`
```

**Distribution math:** If 10 repos with 50 contributors each embed this → 500 potential installs. If 10% install → 50 new users. If each user's next project embeds it → exponential growth with zero marketing spend.

---

## 12. Revenue Architecture

### 12.1 Tier Enforcement

Enforcement happens at the relay layer, not the CLI. The CLI is open-source and unmodifiable. The relay checks team tier on every request:

```
Request comes in → Check team_id → Look up tier in KV:

Free tier:
  team:{team_id}:tier = "free"
  Limits: 3 members, 10 blob writes/day, 72h TTL, 5 version history

Team tier ($29/mo):
  team:{team_id}:tier = "team"
  team:{team_id}:stripe_sub = "sub_xxxxx"
  Limits: unlimited members, unlimited blobs, 30-day TTL, 30-day history

Enterprise ($199/mo):
  team:{team_id}:tier = "enterprise"
  team:{team_id}:stripe_sub = "sub_xxxxx"
  Limits: dedicated namespace, 365-day history, SIEM export, SSO
```

### 12.2 Stripe Integration Flow

```
$ envsync upgrade

  ┌──────────────────────────────────────────────────────────┐
  │  ✦ Upgrade to EnvSync Team ($29/month)                   │
  │                                                          │
  │  What you get:                                           │
  │  • Unlimited team members (currently limited to 3)       │
  │  • Unlimited async relay syncs                           │
  │  • 30-day version history (currently 5 versions)         │
  │  • Cloud audit log                                       │
  │  • Email support (48h response)                          │
  │                                                          │
  │  Opening checkout in your browser...                     │
  └──────────────────────────────────────────────────────────┘

Browser opens → Stripe Checkout session (pre-filled with team_id)
→ Payment successful
→ Stripe webhook → Cloudflare Worker → Update team tier in KV
→ CLI polls relay → Detects upgrade → "✓ Upgraded to Team tier!"
```
