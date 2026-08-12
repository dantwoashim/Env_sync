# DevContract

DevContract is a Go CLI for making local development setup part of the
repository instead of an informal handoff between teammates.

A project describes its runtimes, environment variables, services, bootstrap
steps, health checks, and run targets in `.devcontract/contract.yaml`. The CLI
can execute that contract, diagnose the machine, and exchange encrypted local
environment state between trusted developers.

[Contract reference](docs/CONTRACT.md) |
[Architecture](docs/ARCHITECTURE.md) |
[Threat model](docs/THREAT_MODEL.md) |
[Self-hosting](docs/SELF_HOSTING.md)

## What works today

- scaffold and validate a versioned repository contract;
- bootstrap local tools and services after an explicit trust review;
- run project-defined health checks;
- scan repository text surfaces for likely secret leaks;
- invite and join trusted project members;
- push and pull encrypted `.env` revisions;
- deliver directly over the LAN or through an encrypted relay fallback;
- keep encrypted local backups and revision history;
- register scoped machine identities for relay use;
- expose the same workflow through a companion VS Code extension.

DevContract manages developer environments. Production secret injection and
hosted enterprise secret governance remain outside its scope.

## Install

macOS and Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/dantwoashim/DevContract/main/scripts/install.sh | bash
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/dantwoashim/DevContract/main/scripts/install.ps1 | iex
```

The installers prefer a published binary and can build from source when Go is
available. The current release is also available from
[GitHub Releases](https://github.com/dantwoashim/DevContract/releases/latest).

Build directly:

```bash
git clone https://github.com/dantwoashim/DevContract.git
cd DevContract
go build -o devcontract ./
```

The module currently targets Go `1.25.9`.

## First project

From the repository you want to manage:

```bash
devcontract init
devcontract bootstrap
devcontract doctor
```

Invite a teammate and exchange a revision:

```bash
devcontract invite teammate
devcontract push
devcontract pull
```

Review the generated `.devcontract/contract.yaml` before running bootstrap.
[`examples/contracts`](examples/contracts) contains complete examples.

## Command map

| Command | Purpose |
| --- | --- |
| `init` | Create local identity and scaffold a contract |
| `bootstrap` | Execute reviewed setup steps |
| `doctor` | Check runtimes, services, and contract health |
| `guard scan` | Search configured text surfaces for likely secrets |
| `invite`, `join` | Manage trusted human membership |
| `push`, `pull` | Exchange encrypted environment revisions |
| `backup`, `restore`, `rollback` | Inspect and recover local history |
| `service-key` | Register a scoped relay machine identity |
| `limits` | Show relay limits reported by the current deployment |

## Architecture

- Human identity is derived from an Ed25519 SSH key.
- A reachable trusted peer receives revisions over the direct transport.
- The relay stores a separately encrypted blob for each recipient when direct
  delivery is unavailable.
- Local revision and backup history is encrypted at rest.
- The relay can observe delivery metadata even though it cannot read payloads.
- A compromised developer machine can access secrets available to that user.

The relay source lives in [`relay`](relay). The extension in [`extension`](extension)
calls the CLI rather than reimplementing contract logic.

## Repository map

| Path | Responsibility |
| --- | --- |
| `cmd` | CLI commands |
| `internal` | Contracts, crypto, transport, history, and execution |
| `relay` | Optional Cloudflare Worker relay |
| `extension` | VS Code companion |
| `action` | GitHub Action integration |
| `examples` | Example contracts and environment files |
| `docs` | Protocol, security, operations, and release documentation |

## Verify

The canonical repository gate requires Go, Node.js, npm, Bash, and Make:

```bash
make verify
```

It runs repository hygiene, Go tests and vet, relay tests, extension tests,
repository identity checks, and the MCP documentation self-check.

Focused checks:

```bash
go test ./...
go vet ./...
make test-relay
make test-extension
```

## License

MIT. See [`LICENSE`](LICENSE). Security reports follow the process in
[`SECURITY.md`](SECURITY.md).
