# Launch Content Templates

## Hacker News Post

**Title:** Show HN: EnvSync – Secure .env file sync using SSH keys, no accounts needed

**Body:**

Hi HN, I built EnvSync — a CLI tool that syncs .env files between developers using their existing SSH keys.

**The problem:** Every team shares secrets through Slack DMs, 1Password vaults, or encrypted zip files. It's clunky, insecure, and breaks onboarding.

**How it works:**
1. `envsync init` reads your Ed25519 SSH key
2. `envsync invite @teammate` creates a 6-word code
3. `envsync push` / `envsync pull` syncs .env files

**Architecture:** Noise_XX over TCP for LAN peers, encrypted relay (Cloudflare Worker) for remote. Zero-knowledge — the relay never sees plaintext.

**Stack:** Go, XChaCha20-Poly1305, HKDF, X25519, Ed25519. 8 dependencies. ~15MB binary.

**Free tier:** 3 team members, 10 relay syncs/day. Open source CLI.

GitHub: https://github.com/envsync/envsync

---

## dev.to Article Outline

**Title:** I Built a $0 Infrastructure Tool That Syncs .env Files Using SSH Keys

1. The problem with .env file sharing
2. Why existing solutions suck (1Password CLI, Vault, Doppler — all require accounts)
3. The "aha" moment: SSH keys are already everywhere
4. Architecture deep-dive: Noise → mDNS → Cloudflare relay
5. The crypto: why XChaCha20-Poly1305 + HKDF, not AES-GCM
6. Free tier economics: $0 infrastructure on Cloudflare free tier
7. What I learned building a CLI that feels premium

---

## Twitter Thread

1/ I built a CLI that syncs .env files between developers.
Zero accounts. Zero servers. Uses your SSH key.

`envsync push` → encrypted → LAN or relay → `envsync pull`

[terminal GIF]

2/ The trust model is simple:
- Your SSH key = your identity
- First connect = fingerprint check (TOFU)
- Invite via 6-word code
- Revoke anytime

No usernames, no passwords, no OAuth.

3/ How sync works:
- LAN: mDNS discovery → TCP → Noise_XX handshake → encrypted tunnel
- Remote: encrypt with recipient's X25519 key → upload to CF Worker → recipient downloads + decrypts

The relay NEVER sees your secrets.

4/ Free forever:
- 3 team members
- 10 relay syncs/day
- Full E2E encryption
- Open source CLI

Infrastructure cost: $0 (Cloudflare free tier)

5/ Try it:

```bash
curl -fsSL https://envsync.dev/install.sh | bash
envsync init
envsync invite @yourteammate
```

GitHub: https://github.com/envsync/envsync

---

## Discord Server Structure

- #announcements
- #general
- #support
- #feature-requests
- #security
- #contributing
