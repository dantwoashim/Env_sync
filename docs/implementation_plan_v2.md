# EnvSync — Implementation Plan V2: 100% Completion

> **Status:** The codebase has ~87 files. The build plan spec calls for ~90.
> Many files exist but are stubs or lack integration with each other.
> This plan fills every gap, organized into 4 phases by dependency order.

---

## Phase A — Missing Core Files & Plumbing

> **Goal:** Create every file the build plan specifies that doesn't exist yet.
> Wire the existing relay client into push/pull. Make the CLI actually functional end-to-end.

### A1: Missing config/peer files

#### [NEW] `internal/config/paths.go`
XDG-aware path resolution:
- Windows: `%APPDATA%\envsync\`
- macOS: `~/Library/Application Support/envsync/`
- Linux: `~/.config/envsync/` (if `XDG_CONFIG_HOME` set), else `~/.envsync/`
- Must replace all hardcoded `~/.envsync` references in `config.go`

#### [NEW] `internal/config/migrate.go`
Config schema migration:
- Read `config_version` from TOML
- Run migrations sequentially (v1→v2→v3)
- Backup old config before migrating

#### [NEW] `internal/peer/team.go`
Team struct and operations:
- `Team{ID, Name, Members, CreatedAt}`
- `LoadTeamFromProject(dir)` → reads `.envsync.toml`
- `CreateTeam(name)` → generates team ID (hash of creator fingerprint + name)

#### [NEW] `.envsync.toml` (template/example)
Per-project config template:
```toml
team_id = ""
default_file = ".env"
sync_strategy = "three-way"
```

---

### A2: Missing envfile validation

#### [NEW] `internal/envfile/validate.go`
Validation rules:
- No duplicate keys (warn, not error)
- No binary content (reject `\x00`)
- Suspicious pattern warnings (base64 private keys without quotes, localhost URLs)
- Max value length enforcement (64KB per variable)
- Max file size (1MB total)

---

### A3: Relay client decomposition

The current `internal/relay/client.go` is a monolith (213 lines) with invite/blob/team methods all inlined. Split per the spec:

#### [NEW] `internal/relay/invites.go`
Move `CreateInvite`, `GetInvite`, `ConsumeInvite` out of `client.go`.

#### [NEW] `internal/relay/blobs.go`
Move `UploadBlob`, `GetPendingBlobs`, `DownloadBlob`, `DeleteBlob` out of `client.go`.

#### [NEW] `internal/relay/teams.go`
Move `AddTeamMember`, `ListTeamMembers`, `RemoveTeamMember` out of `client.go`.

#### [NEW] `internal/relay/signal.go`
WebSocket client for hole-punch signaling:
- Connect to signaling Durable Object room for team
- Exchange: public IP, port, NAT type
- Listen for peer join/leave events
- Timeout: 5s for peer to appear

#### [MODIFY] `internal/relay/client.go`
- Fix `loadPrivateKey()` — currently a stub that returns a hardcoded error
- Make it actually read from the configured key path
- Keep only the core `doRequest`, `signRequest`, `readError` methods
- Import and delegate to the split files

---

### A4: Hole-punch transport

#### [NEW] `internal/transport/holepunch.go`
TCP hole-punch implementation:
1. Detect local NAT type (STUN query)
2. Signal endpoint to peer via WebSocket (via `relay/signal.go`)
3. Simultaneous TCP connect with `SO_REUSEADDR` / `SO_REUSEPORT`
4. Race: first successful connection wins
5. Wrap with Noise_XX handshake
6. Timeout → fall through to relay

---

### A5: Store and audit completion

#### [NEW] `internal/store/history.go`
Version listing with metadata:
- Who pushed (fingerprint)
- When (timestamp)
- How many variables changed (diff vs previous)
- File name and size

#### [NEW] `internal/audit/viewer.go`
Pretty-print audit log with filters (separate from logger):
- `ViewEntries(entries []Entry)` → formatted table output
- `FilterEntries(entries, FilterOptions{Peer, Event, DateRange})`
- Used by `cmd/audit.go`

---

### A6: Missing CLI commands

#### [NEW] `cmd/status.go`
`envsync status`:
- Current team info
- Last push/pull time (from audit log)
- Pending incoming blobs (check relay)
- Peer count (online/offline/relay)
- Version store summary

#### [NEW] `cmd/config_cmd.go`
`envsync config [key] [value]`:
- `envsync config` → print all config
- `envsync config relay.url` → print specific value
- `envsync config relay.url https://custom.relay.dev` → set value

---

## Phase B — Integration Wiring

> **Goal:** Wire everything together. Push/pull must use the full fallback chain,
> audit log must be written on every event, diff must use version store.

### B1: Push with relay fallback + audit

#### [MODIFY] `internal/sync/push.go`
After LAN push completes or fails:
- If zero peers found → encrypt with relay envelope → `client.UploadBlob()` for each trusted peer
- Always call `audit.Logger.Log(EventPush, ...)` on success

#### [MODIFY] `cmd/push.go`
- Use UI spinner instead of raw `fmt.Println`
- Integrate orchestrator fallback: LAN → hole-punch → relay
- Write audit entry on completion
- Use structured errors from `ui/errors.go`

### B2: Pull with relay download + diff confirm + audit

#### [MODIFY] `internal/sync/pull.go`
Before listening for LAN:
- Check relay for pending blobs: `client.GetPendingBlobs()`
- Download + decrypt each blob
- Compute diff against current `.env`
- Use `ui.ConfirmAction()` before applying

After receiving any push (LAN or relay):
- Run three-way merge via `resolver.go` if conflicts exist
- Write to `.env` only after confirmation
- Save previous version to store (backup before overwrite)
- Call `audit.Logger.Log(EventPull, ...)`

#### [MODIFY] `cmd/pull.go`
- Use UI spinner
- Integrate relay pull path
- Show diff via `ui.RenderDiff()` before applying
- Write audit entry

### B3: Diff with version store

#### [MODIFY] `cmd/diff.go`
- Compare current `.env` against latest version in store (not `.env.bak`)
- Support `--against` flag for comparing two files
- Use three-way diff when base version available

### B4: Invite/Join/Revoke audit integration

#### [MODIFY] `cmd/invite.go`
- Write `audit.EventInvite` on successful invite creation

#### [MODIFY] `cmd/invite.go` (join section)
- Write `audit.EventJoin` on successful join

#### [MODIFY] `cmd/revoke.go`
- Write `audit.EventRevoke` on successful revocation

### B5: Orchestrator real implementation

#### [MODIFY] `internal/sync/orchestrator.go`
- Replace the TODO hole-punch step with actual `transport.HolePunch()` call
- Replace the placeholder relay step with actual `relay.UploadBlob()` encrypted envelope
- Wire signal.go for signaling coordination

---

## Phase C — Tests & Infrastructure

> **Goal:** Every critical path has a test. CI pipeline is real.

### C1: Missing unit tests

#### [NEW] `internal/crypto/noise_test.go`
Integration test:
- Two goroutines, TCP loopback
- Noise_XX handshake
- Bidirectional encrypted messaging
- Negative test: reject unknown peer

#### [NEW] `internal/crypto/service_key_test.go`
- Generate → export → import → compare
- Verify PEM format correctness
- Invalid PEM rejection

#### [NEW] `internal/audit/logger_test.go`
- Write entries → read back → verify order
- Filter by peer
- Filter by event type
- Empty log handling

#### [NEW] `internal/store/store_test.go`
- Save → list → restore round-trip
- Rotation (save N+1, verify oldest deleted)
- Encrypted content verification
- Missing project handling

#### [NEW] `internal/envfile/validate_test.go`
- Duplicate key detection
- Binary content rejection
- Max value length enforcement
- Suspicious pattern warnings

### C2: Worker tests (Miniflare)

#### [NEW] `relay/test/invites.test.ts`
- CRUD lifecycle
- TTL expiry
- Duplicate prevention
- Fingerprint verification on consume

#### [NEW] `relay/test/relay.test.ts`
- Blob upload/download/delete
- Rate limit enforcement
- Pending list filtering
- TTL expiry

#### [NEW] `relay/test/auth.test.ts`
- Valid signature accepted
- Expired timestamp rejected
- Wrong key rejected
- Tampered body rejected

### C3: CI/CD workflows

#### [NEW] `.github/workflows/ci.yml`
PR checks:
- `go vet ./...`
- `go test ./... -race -count=1`
- `go build ./...`
- Matrix: `{ubuntu-latest, macos-latest, windows-latest}`

#### [NEW] `.github/workflows/security.yml`
Weekly schedule:
- `govulncheck ./...`
- `go list -m all | nancy sleuth`
- Dependency audit

### C4: Project files

#### [NEW] `Makefile`
```makefile
build, test, test-race, lint, vet, cover, release-snapshot, clean
```

#### [NEW] `LICENSE`
MIT license with current year + "EnvSync Contributors"

#### [NEW] `CONTRIBUTING.md`
- How to set up dev environment
- Code style (gofmt, golangci-lint)
- PR process
- Test requirements

---

## Phase D — Polish & Final Verification

> **Goal:** Every output the user sees is beautiful. Every edge case handled.

### D1: Error UX overhaul across all commands

#### [MODIFY] All `cmd/*.go` files
- Replace raw `fmt.Errorf` returns with `ui.RenderError(ui.StructuredError{...})`
- Every error path: message + cause + suggestion
- No stack traces, no Go-internal errors exposed
- Test: trigger every known error condition

### D2: UI integration across all commands

#### [MODIFY] `cmd/push.go`, `cmd/pull.go`, `cmd/init.go`, `cmd/peers.go`
- Use `ui.Header()`, `ui.Success()`, `ui.Spinner()` instead of raw `fmt.Println`
- Use `ui.PrintTable()` for peers
- Use `ui.RenderDiff()` for pull confirmation

### D3: Command help text quality

- Every command: meaningful `Long` description
- Every flag: documented with examples
- `--help` output should be self-sufficient

### D4: End-to-end verification

Run the full flow on two machines (or two terminals):
1. `envsync init` (both sides)
2. `envsync invite @teammate` → share code
3. `envsync join <code>` (other side)
4. `envsync push` (LAN direct)
5. `envsync pull` (LAN receive + diff + confirm)
6. `envsync push` (relay fallback — kill other side first)
7. `envsync pull` (relay download)
8. `envsync diff`
9. `envsync backup` → `envsync restore`
10. `envsync audit --last 10`
11. `envsync status`
12. `envsync revoke @teammate`

Verify each step produces correct styled output and audit entries.

---

## Summary

| Phase | Files | Type |
|-------|-------|------|
| **A** | 14 new + 1 modify | Missing core files |
| **B** | 7 modify | Integration wiring |
| **C** | 12 new | Tests + infrastructure |
| **D** | 6 modify | Polish + verification |
| **Total** | **26 new + 14 modify = 40 changes** | |
