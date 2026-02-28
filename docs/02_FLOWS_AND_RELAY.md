# EnvSync — Connection Flows & Relay Design

---

## 5. Connection Flows (Every Byte Accounted For)

### 5.1 Flow 1: LAN Sync (Same Network — The Magic Moment)

This is the flow that makes people say "holy shit" the first time they see it.

```
ALICE'S MACHINE                                    BOB'S MACHINE
────────────────                                    ──────────────
$ envsync push                                      (EnvSync daemon listening)

1. Read .env from current directory
   Parse: 12 variables, 847 bytes

2. mDNS query: _envsync._tcp.local
   (UDP multicast to 224.0.0.251:5353)
   Question: "Any EnvSync peers on this network?"

                                                    3. mDNS response:
                                                       Service: "bob-envsync._envsync._tcp.local"
                                                       Host: bob-macbook.local
                                                       Port: 7733
                                                       TXT: fingerprint=SHA256:x7Km...R3qP
                                                            team=my-startup
                                                            version=1

4. DNS resolution: bob-macbook.local → 192.168.1.42

5. TCP connect to 192.168.1.42:7733
   (standard TCP three-way handshake: SYN → SYN-ACK → ACK)

6. Noise_XX Handshake (over TCP)
   ── Message 1 (32 bytes): ephemeral public key ──►
                                                    7. Verify: is Alice's fingerprint in our trust registry?
   ◄── Message 2 (~96 bytes): ephemeral + encrypted static + tag ──
8. Verify Bob's static key matches cached fingerprint
   ── Message 3 (~80 bytes): encrypted static + tag ──►
                                                    9. Mutual authentication complete.
                                                       Forward-secure channel established.

10. Send encrypted payload:
    ┌─────────────────────────────────────────┐
    │ Header (16 bytes)                       │
    │   Version: 1                            │
    │   Type: ENV_PUSH                        │
    │   Payload length: 847                   │
    │   File: ".env"                          │
    │   Variables: 12                         │
    │   Sequence: 47 (monotonic per team)     │
    │   Timestamp: 2026-02-28T13:45:00Z       │
    ├─────────────────────────────────────────┤
    │ Encrypted .env content (863 bytes)      │
    │   (847 bytes + 16 byte Poly1305 tag)    │
    └─────────────────────────────────────────┘

                                                    11. Decrypt payload
                                                    12. Validate: parse .env, check for anomalies
                                                    13. If changes detected:
                                                        - Backup current .env to store (versioned)
                                                        - Write new .env to project directory
                                                        - Write to audit log

    ◄── ACK (encrypted, 1 byte) ──

14. Display:
    ✓ Synced with @bob (192.168.1.42) — 0.21s
    12 variables, 847 bytes

TOTAL TIME: ~200ms (mDNS: ~50ms, TCP: ~1ms, Noise: ~5ms, Transfer: ~1ms, Overhead: ~143ms)
TOTAL NETWORK TRAFFIC: ~1.5KB
NO INTERNET CONNECTION REQUIRED
```

### 5.2 Flow 2: WAN Sync (Different Networks — TCP Hole Punch)

```
ALICE (Home WiFi)                RELAY (CF Worker)              BOB (Office WiFi)
─────────────────               ──────────────────              ─────────────────
$ envsync push --to @bob

1. mDNS query: no LAN peers
   Fallback to WAN...

2. WebSocket connect to
   relay.envsync.dev/signal
   ── Upgrade: websocket ──►
                                3. Durable Object created for
                                   team "my-startup"
                                   Alice registered as waiting
                                                                4. Background poller detects
                                                                   pending signal for @bob
                                                                   WebSocket connect to relay
                                ◄── Bob connects ──

5. Alice sends her endpoint:      6. Relay forwards:              7. Bob receives Alice's
   {public_ip, port, nat_type}       Alice's endpoint to Bob        public endpoint
                                     Bob's endpoint to Alice
8. Alice receives Bob's
   public endpoint

── SIMULTANEOUS TCP OPEN ──────────────────────────────────────────────
9. Alice: connect() to            10. Both SYN packets cross        11. Bob: connect() to
   Bob's public_ip:port               in flight. NATs see              Alice's public_ip:port
   (SO_REUSEADDR on same port)        outgoing→allow incoming          (SO_REUSEADDR on same port)

12. TCP connection established (if NATs are compatible)
    Success rate: ~70% (non-symmetric NATs)

13. Noise_XX handshake over the direct TCP connection
    (identical to LAN flow steps 6-9)

14. Encrypted .env transfer
    (identical to LAN flow steps 10-14)

── IF HOLE PUNCH FAILS ──────────────────────────────────────────────
15. Timeout after 5 seconds
16. Fallback to Layer 3 (Relay)
    (see Flow 3 below)

TOTAL TIME (success): ~800ms
TOTAL TIME (fallback to relay): ~2s
```

### 5.3 Flow 3: Async Relay (Peers Not Simultaneously Online)

```
ALICE (Online now)              RELAY (CF Worker + KV)          BOB (Offline, online later)
──────────────────              ──────────────────────          ──────────────────────────
$ envsync push --to @bob

1. Check: is @bob online?
   WebSocket signal → no response → @bob offline

2. Encrypt .env for Bob:
   a. Generate ephemeral X25519 keypair
   b. ECDH(ephemeral_private, bob_noise_static_public)
   c. HKDF → encryption_key
   d. XChaCha20-Poly1305.Seal(env_content)

3. Build envelope:
   {
     id: "blob_a7f3e2",
     sender: "SHA256:a3B7x...kQ9f",
     recipient: "SHA256:x7Km...R3qP",
     ephemeral_public: "base64...",
     nonce: "base64...(24 bytes)",
     ciphertext: "base64...(padded to 1KB boundary)",
     sequence: 48,
     timestamp: "2026-02-28T13:45:00Z",
     file: ".env",
     ttl: 259200  // 72 hours in seconds
   }

4. Upload:
   PUT /relay/my-startup/blob_a7f3e2
   Authorization: ES-SIG {signature of request body with Alice's key}
                                5. Verify Alice's signature
                                   Verify Alice is member of team "my-startup"
                                   Store in KV:
                                     Key: blob:my-startup:blob_a7f3e2
                                     Value: envelope JSON
                                     TTL: 72 hours

6. Display:
   ✓ Queued for @bob (relay, 72h TTL)
   📡 @bob will receive on next `envsync pull`

═══════════════════════════════  (3 HOURS LATER)  ═══════════════════════════════

                                                    7. Bob comes online
                                                       $ envsync pull

                                                    8. GET /relay/my-startup/pending
                                                       ?for=SHA256:x7Km...R3qP
                                                       Authorization: ES-SIG {signature}

                                9. Verify Bob's signature
                                   Return list of pending blobs:
                                   [{id: "blob_a7f3e2", sender: "SHA256:a3B7x...kQ9f", ...}]

                                                    10. GET /relay/my-startup/blob_a7f3e2

                                                    11. Decrypt:
                                                        a. ECDH(bob_noise_static_private, alice_ephemeral_public)
                                                        b. HKDF → decryption_key
                                                        c. XChaCha20-Poly1305.Open(ciphertext)

                                                    12. Validate parsed .env

                                                    13. If changes detected:
                                                        - Show diff (always, before applying)
                                                        - Prompt: "Apply 3 changes from @alice? [Y/n]"
                                                        - Backup current, write new

                                                    14. DELETE /relay/my-startup/blob_a7f3e2
                                                        (cleanup after successful download)

                                                    15. Display:
                                                        ✓ Received .env from @alice (3h ago via relay)
                                                        3 variables updated, 1 added
```

### 5.4 Flow 4: The Invite Flow (Onboarding a New Peer)

```
ALICE (Team Lead)               RELAY                          BOB (New Developer)
─────────────────               ─────                          ─────────────────
$ envsync invite @bob

1. Fetch Bob's SSH keys:
   GET https://github.com/bob.keys
   Response: "ssh-ed25519 AAAAC3Nz... bob@laptop"

2. Extract Ed25519 key, convert to X25519
   Compute expected fingerprint: SHA256:x7Km...R3qP

3. Generate invite token:
   6 random words from BIP39-like wordlist:
   "tiger-castle-moon-river-flame-hope"

4. Create invite:
   POST /invites
   {
     token_hash: SHA256("tiger-castle-moon-river-flame-hope"),
     team_id: "my-startup",
     team_name: "My Startup",
     inviter: {
       github: "alice",
       fingerprint: "SHA256:a3B7x...kQ9f",
       public_key: "base64..."
     },
     invited: {
       github: "bob",
       expected_fingerprint: "SHA256:x7Km...R3qP"
     },
     permissions: ["push", "pull"],
     expires_at: "2026-03-01T13:45:00Z"  // 24 hours
   }

                                5. Store invite in KV
                                   Key: invite:{token_hash}
                                   TTL: 24 hours

6. Display:
   ┌──────────────────────────────────────────────────┐
   │  ✦ Invite created for @bob                       │
   │                                                   │
   │  Tell Bob to run:                                 │
   │  envsync join tiger-castle-moon-river-flame-hope  │
   │                                                   │
   │  Expires in 24 hours.                             │
   │  Expected fingerprint: SHA256:x7Km...R3qP         │
   └──────────────────────────────────────────────────┘

                                                    Bob receives invite code via any channel
                                                    (Slack, email, in-person — the code is not secret
                                                    enough to be dangerous since it's bound to Bob's
                                                    GitHub identity)

                                                    $ envsync join tiger-castle-moon-river-flame-hope

                                                    7. GET /invites/{token_hash}

                                                    8. Display to Bob:
                                                       "Join team 'My Startup'?"
                                                       "Invited by: @alice (SHA256:a3B7x...kQ9f)"
                                                       "Your identity: @bob (SHA256:x7Km...R3qP)"
                                                       "[Y/n]"

                                                    9. Bob confirms. Register Bob in team:
                                                       PUT /teams/my-startup/members/bob
                                                       {
                                                         github: "bob",
                                                         fingerprint: "SHA256:x7Km...R3qP",
                                                         public_key: "base64...",
                                                         joined_at: "2026-02-28T14:00:00Z",
                                                         invited_by: "alice"
                                                       }

                                10. Delete invite (one-time use)
                                    Update team membership in KV

                                                    11. Store team config locally:
                                                        ~/.envsync/teams/my-startup/
                                                          team.toml
                                                          peers/alice.toml

                                                    12. Display:
                                                        ✓ Joined team "My Startup"
                                                        Run 'envsync pull' to get your .env
```

---

## 6. Cloudflare Worker Relay — Complete Design

### 6.1 Architecture

```
Cloudflare Edge Network (Global, 300+ PoPs)
│
├── Worker: relay.envsync.dev
│   ├── Router (Hono framework — lightweight, fast)
│   ├── Auth middleware (signature verification)
│   ├── Rate limiter (per-IP, per-team)
│   └── Endpoints:
│       ├── POST   /invites              (create invite)
│       ├── GET    /invites/:hash        (retrieve invite)
│       ├── DELETE /invites/:hash        (consume invite)
│       ├── PUT    /relay/:team/:blob    (upload encrypted blob)
│       ├── GET    /relay/:team/pending  (list pending for recipient)
│       ├── GET    /relay/:team/:blob    (download blob)
│       ├── DELETE /relay/:team/:blob    (cleanup after download)
│       ├── GET    /teams/:team/members  (list team members)
│       ├── PUT    /teams/:team/members/:user (add member)
│       ├── DELETE /teams/:team/members/:user (remove member)
│       └── GET    /health               (health check)
│
├── Durable Object: SignalingRoom
│   └── Per-team WebSocket coordination for TCP hole-punch
│       ├── webSocketMessage → exchange endpoints between peers
│       ├── webSocketClose → cleanup
│       └── Hibernation API (no cost when idle)
│
├── KV Namespace: ENVSYNC_DATA
│   ├── invite:{hash}           → invite JSON (24h TTL)
│   ├── blob:{team}:{blob_id}   → encrypted envelope (72h TTL)
│   ├── team:{team_id}:meta     → team metadata (persistent)
│   ├── team:{team_id}:members  → member list (persistent)
│   └── rate:{ip}:{window}      → request count (1min TTL)
│
└── Analytics Engine: ENVSYNC_ANALYTICS
    └── Anonymous usage events (sync count, error rate, latency)
```

### 6.2 Request Authentication

Every mutating request is authenticated via Ed25519 signature:

```
Authorization: ES-SIG timestamp={unix_ts},fingerprint={fp},signature={base64_sig}

Signed payload = "{method}\n{path}\n{timestamp}\n{body_sha256}"

Verification:
1. Parse fingerprint from header
2. Look up public key in team membership KV
3. Verify timestamp is within 5-minute window (prevents replay)
4. Verify Ed25519 signature of the canonical payload
5. If valid → process request
6. If invalid → 401 Unauthorized
```

### 6.3 Rate Limits (Free Tier Protection)

| Resource | Limit | Window | Response on Exceed |
|----------|-------|--------|--------------------|
| Requests per IP | 60 | 1 minute | 429 + Retry-After |
| Blob uploads per team | 50 | 1 day | 429 + upgrade prompt |
| Blob size | 64 KB | per blob | 413 Payload Too Large |
| Active blobs per team | 100 | rolling | 507 + cleanup old blobs |
| Invite creates per user | 10 | 1 hour | 429 |
| Team members (free) | 3 | persistent | 402 + upgrade prompt |
| Team members (paid) | unlimited | persistent | — |

### 6.4 Free Tier Math (Detailed)

```
Cloudflare Workers Free Tier:
  Requests:  100,000/day
  KV reads:  100,000/day
  KV writes: 1,000/day
  KV storage: 1 GB
  Durable Object requests: 1,000/day (included free)

Our usage model (per active team per day):
  Typical team: 3 peers, 2 syncs/day average
  
  Per sync event:
    If LAN (70% of syncs): 0 relay requests
    If WAN direct (21%):   2 WebSocket messages (signaling)
    If relay (9%):         1 PUT + 1 GET + 1 DELETE = 3 KV ops
  
  Per team per day (average):
    Relay requests: 2 syncs × 0.09 × 3 = 0.54 requests
    KV writes: 2 syncs × 0.09 × 1 = 0.18 writes
    KV reads:  2 syncs × 0.09 × 2 = 0.36 reads
    Signaling: 2 syncs × 0.21 × 2 = 0.84 DO requests

  At 1,000 active teams:
    Relay requests/day:  540  (of 100,000 limit → 0.5%)
    KV writes/day:       180  (of 1,000 limit → 18%)
    KV reads/day:        360  (of 100,000 limit → 0.4%)
    DO requests/day:     840  (of 1,000 limit → 84%)  ← TIGHTEST
    KV storage:          ~5MB (of 1GB → 0.5%)

  BOTTLENECK: Durable Object requests hit free limit at ~1,200 teams
  SOLUTION: At that scale, upgrade to Workers Paid ($5/mo) for 1M DO req/mo
```

> **Critical Insight:** The tightest constraint is actually Durable Object requests (for WebSocket signaling), not KV writes as I initially assessed. At ~1,200 active teams, we need to upgrade to Cloudflare Workers Paid plan ($5/month). This is fine — at 1,200 teams we'll have revenue.

---

## 7. CLI UX Design (The Million-Dollar Feel)

### 7.1 Design Principles

1. **Sub-second everything.** No command should take >1s on LAN. This is non-negotiable.
2. **Show, don't tell.** Use visual indicators (spinners, progress, tables) instead of walls of text.
3. **Errors are features.** Every error message must have: what went wrong, why, and what to do next.
4. **Color is information.** Green = success. Yellow = warning/action needed. Red = error. Cyan = info. Dim gray = metadata.
5. **Confirm before overwrite.** Always show diff before applying changes. Default to safe behavior.

### 7.2 Complete Command Reference

```
envsync — P2P environment variable synchronization

COMMANDS:
  init          Initialize EnvSync (reads SSH key, creates config)
  invite        Invite a teammate by GitHub username
  join          Accept an invite using a join code
  push          Send .env to peers (LAN auto-discover, relay fallback)
  pull          Receive .env from peers
  diff          Show differences between local and last-synced version
  peers         List all team members and their status
  revoke        Remove a peer's access to the team
  status        Show current sync state and pending changes
  backup        Create encrypted local backup of current .env
  restore       Restore .env from a previous version
  audit         Show sync history log
  config        View or modify configuration
  version       Print version information

FLAGS (global):
  -v, --verbose     Show detailed output (Noise handshake details, timing)
  -q, --quiet       Suppress all output except errors
  --no-color        Disable colored output
  --config PATH     Use alternate config file
  --file FILE       Target specific .env file (default: .env)

EXAMPLES:
  envsync init                              # First-time setup
  envsync invite @alice                     # Invite a teammate
  envsync join tiger-castle-moon-river      # Accept an invite
  envsync push                              # Sync to all peers
  envsync pull                              # Get latest from peers
  envsync push --file .env.production       # Sync specific file
  envsync diff --from @alice                # Diff against Alice's version
  envsync peers                             # List team members
  envsync revoke @bob                       # Remove Bob's access
  envsync backup                            # Encrypted local backup
  envsync restore --version 3               # Restore version 3
  envsync audit --last 20                   # Show last 20 sync events
```

### 7.3 Terminal Output Examples (Every Screen)

**Init:**
```
$ envsync init

  ✦ EnvSync v1.0.0

  ▸ Reading SSH key from ~/.ssh/id_ed25519
  ▸ Key type: Ed25519
  ▸ Fingerprint: SHA256:a3B7x9pQmN5kR2wL7hT8vY1cE4fG6jK0sI3dA8oU
  ▸ GitHub user: @prabi (matched via github.com/prabi.keys)
  ▸ Created config at ~/.envsync/config.toml
  ▸ Encrypted store initialized at ~/.envsync/store/

  ⚠ Your SSH key has no passphrase. This means your EnvSync
    encryption keys are only as secure as your filesystem.
    Consider adding a passphrase: ssh-keygen -p -f ~/.ssh/id_ed25519

  ✓ Ready. Run 'envsync invite @teammate' to start a team.
```

**Push (LAN success):**
```
$ envsync push

  ✦ Pushing .env (12 variables, 847 bytes)

  ▸ Scanning local network...
  ▸ Found @alice (192.168.1.42) — LAN direct
  ▸ Noise handshake ✓ (SHA256:x7Km...R3qP)
  ▸ Encrypted + sent in 0.03s

  @bob is offline — queuing to relay...
  ▸ Encrypted for @bob's key (SHA256:mN5p...wX2k)
  ▸ Uploaded to relay (blob_a7f3e2, TTL: 72h)

  ✓ Synced with 1/2 peers. 1 queued to relay.
```

**Pull (with diff):**
```
$ envsync pull

  ✦ Checking for updates...

  ▸ Found 1 pending sync from @alice (3h ago, via relay)

  ┌─────────────────────────────────────────────────────┐
  │ .env changes from @alice                            │
  ├─────────────────────┬───────────────┬───────────────┤
  │ Variable            │ Current       │ Incoming      │
  ├─────────────────────┼───────────────┼───────────────┤
  │ DATABASE_URL        │ (unchanged)   │               │
  │ API_KEY             │ sk_test_old   │ sk_test_new ← │
  │ STRIPE_SECRET       │ (missing)     │ sk_live_x  ← │
  │ REDIS_HOST          │ localhost     │ (removed)  ← │
  │ DEBUG               │ true          │ (unchanged)   │
  └─────────────────────┴───────────────┴───────────────┘

  Summary: 1 updated, 1 added, 1 removed, 9 unchanged

  Apply changes? [Y/n/d(iff)] y

  ▸ Backed up current .env → version 5
  ▸ Applied 3 changes

  ✓ .env updated from @alice
```

**Peers:**
```
$ envsync peers

  ┌──────────────────────────────────────────────────────────────────┐
  │ Team: my-startup (3 peers)                                       │
  ├──────────┬──────────────────────┬──────────┬─────────────────────┤
  │ User     │ Fingerprint          │ Status   │ Last Sync           │
  ├──────────┼──────────────────────┼──────────┼─────────────────────┤
  │ @prabi   │ SHA256:a3B7...kQ9f   │ ● you    │ —                   │
  │ @alice   │ SHA256:x7Km...R3qP   │ 🟢 online │ 2 minutes ago       │
  │ @bob     │ SHA256:mN5p...wX2k   │ 🔴 away   │ 3 hours ago (relay) │
  └──────────┴──────────────────────┴──────────┴─────────────────────┘
```

**Error (no peers):**
```
$ envsync push

  ✗ No peers configured.

  You need at least one teammate to sync with.

  Quick start:
    1. Invite a teammate:  envsync invite @their-github-username
    2. They run:           envsync join <the-code-you-get>
    3. Then sync:          envsync push

  Need help? https://envsync.dev/quickstart
```

**Error (network failure):**
```
$ envsync push --to @bob

  ✦ Pushing .env (12 variables, 847 bytes)

  ▸ No peers on local network
  ▸ Attempting WAN connection...
  ✗ TCP hole-punch failed (symmetric NAT detected)
  ▸ Falling back to relay...
  ✗ Relay unreachable: connection timeout

  Possible causes:
    1. No internet connection
    2. Corporate firewall blocking relay.envsync.dev
    3. Relay service temporarily down

  What to try:
    • Check internet: curl -I https://relay.envsync.dev/health
    • Use a different network (e.g., phone hotspot)
    • Try again in a few minutes

  Status page: https://status.envsync.dev
```
