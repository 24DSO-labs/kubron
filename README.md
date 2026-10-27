# Kubron

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

- **Multi-cluster management** — connect and switch between clusters from a single window
- **Cluster dashboard** — nodes, workloads, events and cluster health at a glance, with extra panels for a quicker overview
- **Workloads** — pods, deployments, statefulsets, daemonsets, jobs and cronjobs
- **Pod tools** — live logs, exec into containers, port forwarding
- **Argo CD** — applications, sync status and health directly in the UI
- **Local users & roles** — admin/user roles with secure password auth (PBKDF2), lockout after failed attempts
- **Audit log** — track who did what, with log rotation
- **No telemetry** — your data stays on your machine

## Getting started

### Download

Grab the latest release for your platform on the [Releases](https://github.com/VadimIl224/kubron/releases) page.

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

See [CONTRIBUTING.md](CONTRIBUTING.md) (DCO sign-off required).
