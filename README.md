# NullRoute

NullRoute is a Go + GoBGP project for continuously publishing blackhole routes from open-source blocklists into your network via BGP.

Primary use case: feed a router (for example MikroTik) with dynamic null/blackhole routes so unwanted traffic gets dropped quickly and centrally.

## Features

- Pulls from common open-source blocklist feeds
- Parses IPv4/IPv6 CIDRs (or single IPs)
- Prefix-length guardrails to avoid over-broad announcements
- Allowlist support for prefixes you never want announced
- Differential updates (add/remove only changed prefixes)
- Emits blackhole community (`65535:666` by default)
- Works well in containers and Kubernetes

## Architecture

- `nullroute` app fetches + normalizes source prefixes
- `gobgpd` maintains BGP sessions and route table
- `nullroute` calls `gobgp global rib add/del` for sync operations

## Quick start

1. Build binaries:

```bash
make build
```

2. Copy and edit config:

```bash
cp examples/nullroute-config.yaml ./config.yaml
```

3. Ensure `gobgpd` is running and peered, then run:

```bash
./bin/nullroute -config ./config.yaml
```

## Kubernetes

See `deploy/k8s/` for a pod with two containers:

- `gobgpd` (BGP daemon)
- `nullroute` (blocklist sync loop)

Use `hostNetwork: true` when your peer expects direct node reachability for BGP.
Use an image pull secret named `ghcr-pull-secret` in the deployment namespace.
In kroyio clusters, create it once in `sekrets` with:
`spillway.kroy.io/replicate-to: all`.

## MikroTik example

See [docs/mikrotik.md](docs/mikrotik.md).

## Default feeds in sample config

- Team Cymru Fullbogons (IPv4 + IPv6)
- Spamhaus DROP
- Spamhaus EDROP
- FireHOL Level 1

You should validate policy and legal/compliance requirements before enforcing any third-party list in production.

## Repo layout

- `cmd/nullroute`: executable
- `internal/config`: YAML config parsing/validation
- `internal/sources`: source fetch + prefix parsing
- `internal/syncer`: state diff + GoBGP CLI integration
- `examples/`: sample app and GoBGP configs
- `deploy/k8s`: Kubernetes manifests
- `docs/`: integration guides

## License

MIT
