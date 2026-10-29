# Kubron

[![CI](https://github.com/VadimIl224/kubron/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/VadimIl224/kubron/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/VadimIl224/kubron?include_prereleases)](https://github.com/VadimIl224/kubron/releases)

A lightweight, open-source Kubernetes desktop IDE built with **Go** and **Wails** — a fast, private alternative to Lens with **Argo CD built in**. No telemetry, no mandatory account, your kubeconfig never leaves your machine.

<!-- Screenshot will be added in docs/img/ -->
<!-- ![Kubron](docs/img/screenshot-dashboard.png) -->

## Why Kubron

| | Kubron | Lens | k9s |
|---|---|---|---|
| UI | Desktop + web (WIP) | Desktop | Terminal |
| Built with | Go + Wails (native webview) | Electron | Go TUI |
| Argo CD support | Built in | Paid extension | Via plugins |
| Account required | No | Yes | No |
| Telemetry | None | Yes | None |
| License | Apache-2.0 | Proprietary + OSS | Apache-2.0 |

## Features

### Clusters
- **Multi-cluster management** — add clusters from a kubeconfig, switch between them from a single window, and give each one a name, color, icon and environment tag (`dev` / `test` / `stage` / `prod`)
- **Namespace scoping** — pick one, several or all namespaces; works with restricted RBAC (no cluster-wide `list` required) and reports partial failures instead of hiding them
- **Live updates** — shared informers keep lists and the dashboard fresh without hammering the API server

### Dashboard
- **Cluster overview** — nodes, workloads, capacity, storage and cluster health at a glance
- **Problem detection** — problem pods, blockers, warnings, hot pods close to their limits and nodes with low headroom
- **Recent changes** — see what changed in the cluster right before something broke
- **Expiring certificates** — spot soon-to-expire certs before they cause an outage
- **Resource usage** — top pods and usage sparklines (with the optional metrics stack)

### Resources
- **Wide resource coverage** — workloads (Pods, Deployments, StatefulSets, DaemonSets, ReplicaSets, Jobs, CronJobs), networking (Services, Endpoints, EndpointSlices, Ingresses, IngressClasses, NetworkPolicies), config (ConfigMaps, Secrets, ResourceQuotas, LimitRanges, HPA, PDB), storage (PV, PVC, StorageClasses), RBAC (Roles, ClusterRoles, bindings, ServiceAccounts), webhooks, leases, priority and runtime classes
- **Custom resources** — CRDs are discovered automatically and shown with their `additionalPrinterColumns`, like `kubectl get`
- **Detail panels** — conditions, events, owners, containers, volumes, affinity, tolerations and more, one click away
- **YAML editor** — Monaco-based view/edit with diff; apply changes or create new objects from a manifest
- **Bulk actions** — act on many objects at once; one failure doesn't stop the rest
- **Keyboard shortcuts** — `Alt+P/D/N/E/S` to jump to Pods, Deployments, Nodes, Events and Settings

### Operations
- **Pod tools** — live logs, terminal (exec) in containers, port forwarding to pods and services, eviction
- **Deployments** — scale, restart, view revision history and roll back
- **Nodes** — cordon/uncordon, drain, and open a shell on a node
- **Metrics** — one-click optional metrics stack (Prometheus, kube-state-metrics, node-exporter) with charts for pods, workloads, nodes, PVCs and ingresses

### Helm
- **Releases** — list, inspect values/manifest/notes, view history, diff revisions, roll back, uninstall
- **Chart catalog** — manage repositories (including basic auth and an option to skip TLS verification), search charts and browse versions
- **Install & upgrade** — dry-run preview before installing; upgrade releases to a newer chart version
- **Update hints** — see which installed releases have a newer chart available

### Argo CD
- **Built in, no extra install** — applications, ApplicationSets and projects with sync and health status in the UI
- **Actions** — sync (with options), refresh / hard refresh, terminate operation, roll back, suspend/resume auto-sync, delete with or without cascade
- **Diff & graph** — live-vs-desired diff and a resource graph for each application
- **Project access** — roles and access of Argo CD projects

### Security & access control
- **Local users & roles** — `admin`, `operator` and `user` roles built on fine-grained permissions (read, secrets, operate, write, exec, port-forward, cluster and user management, audit)
- **Safe defaults** — PBKDF2-SHA256 password hashing (600k iterations), lockout after failed attempts, auto-logout on inactivity that closes terminals and port forwards
- **Audit log** — track who did what, with log rotation
- **Extension points** — a public `pkg/ext` API for external modules (e.g. SSO/OIDC, custom storage)
- **No telemetry** — no mandatory account, your kubeconfig never leaves your machine

## Getting started

### Download

Grab the latest release for your platform on the [Releases](https://github.com/VadimIl224/kubron/releases) page:

| Platform | File |
|---|---|
| Windows (x64) | `Kubron_<version>_windows-amd64.zip` |
| Linux (x64) | `Kubron_<version>_linux-amd64.tar.gz` |
| macOS (Apple Silicon and Intel) | `Kubron_<version>_macos-universal.zip` |

Checksums are in `SHA256SUMS.txt`.

> The builds are not code-signed yet. On Windows, SmartScreen may warn you (More info → Run anyway).
> On macOS, right-click the app and choose Open, or run `xattr -dr com.apple.quarantine Kubron.app`.
> On Linux, GTK 3 and WebKitGTK 4.1 are required (on Ubuntu 24.04: `libwebkit2gtk-4.1-0`).

### Build from source

Requirements: [Go](https://go.dev/dl/), [Node.js](https://nodejs.org/) and the [Wails CLI](https://wails.io/docs/gettingstarted/installation).

```bash
git clone https://github.com/VadimIl224/kubron.git
cd kubron
wails build
```

## Web mode (experimental)

Kubron can also run as a local web server:

```bash
cd frontend && npm ci && npm run build && cd ..
go run ./cmd/kubron-server --listen 127.0.0.1:8080
```

> The web UI is under active development and not fully wired to the backend yet — it currently listens on loopback only, without remote access. See the [roadmap](#roadmap).

## Roadmap

- [x] Desktop app (Wails + Go + Vue) with local auth and audit log
- [x] Argo CD integration
- [ ] Web mode with multi-user support and session auth
- [ ] SSO/OIDC (planned as a commercial add-on)
- [x] Helm releases management

## License

The Kubron core is open source under [Apache-2.0](LICENSE).

Paid subscription features are planned. The first one — **SSO/OIDC login for organizations** — will ship as a separate commercial module. The open core will always include local users, roles, audit log and emergency admin access.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) (DCO sign-off required). Please follow the
[Code of Conduct](CODE_OF_CONDUCT.md). To report a vulnerability, see [SECURITY.md](SECURITY.md).
Release notes: see [CHANGELOG.md](CHANGELOG.md).
