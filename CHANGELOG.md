# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Helm:** Chart upgrade preview now includes the new chart's notes and a diff of old vs new chart default values.

### Fixed

- Helm: fixed issues #37 and #38.

## [0.2.0] - 2026-10-29

## [0.2.0] - 2026-10-29

### Added

- Helm: upgrade a release to another chart version ("Chart version" tab). Choose the repository
  and version, keep or edit the current values, preview the manifest diff, then apply with live
  progress and cancel. Downgrades and deprecated versions are flagged. The preview also shows the
  new chart's notes and a diff of the old and new chart default values.
- Helm: audit-log labels for install, upgrade, rollback, uninstall and preview actions.
- Multilingual UI (English base + Russian) on vue-i18n v11: language switcher


### Fixed

- Helm: the "Edit values" tab showed a raw translation key instead of its title.
- Helm: the "Add" button on the repositories page showed a stray "≥" instead of "+".
- Helm: repository badges (`oci`, `auth`, `no TLS verify`) and `deprecated` in the catalog are now translated.
- Helm: the "release changed" warning and the empty operations list showed literal `{{ ... }}`
  placeholders; untranslated Russian strings in the Helm section.
- Helm: OCI chart versions with build metadata (`1.0.0+build`) were listed with `_` and could not
  be selected.
- Docs: the 0.1.0 notes said cancel is available for rollback and uninstall; it is available for
  install and upgrade only (the Helm SDK takes no context for the other two).
- Web mode: idle sessions are now cleaned up in the background; per-user resource limits are
  configurable in the server config; a Helm operation is bound to its owner session (only the
  owner can cancel it).

## [0.1.0] - 2026-10-28

First public release.

### Added

- Desktop app (Wails + Go + Vue) with multi-cluster management and namespace scoping.
- Cluster dashboard: health, capacity, problem pods, recent changes, expiring certificates.
- Resource browser for built-in Kubernetes types and CRDs, with a Monaco YAML editor.
- Pod tools: logs, exec, port forwarding. Deployment scale, restart and rollback.
  Node cordon, drain and shell.
- Optional metrics stack (Prometheus, kube-state-metrics, node-exporter) with charts.
- Helm: releases, history and diff, rollback, chart catalog (including OCI registries),
  install with dry-run, upgrade, update indicator. Live progress with cancel for install,
  upgrade, rollback and uninstall, an operations journal view, Secret and ConfigMap
  release storage.
- Argo CD: applications, ApplicationSets, projects, sync and other actions, diff, graph.
- Local users with `admin` / `operator` / `user` roles, audit log with rotation and client
  address, auto-logout on inactivity.
- Public extension API (`pkg/ext`) for external modules.
- Experimental web mode (`cmd/kubron-server`): several concurrent sessions, per-session
  cluster managers with per-user resource limits, idle informers released after a TTL.

[Unreleased]: https://github.com/VadimIl224/kubron/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/VadimIl224/kubron/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/VadimIl224/kubron/releases/tag/v0.1.0
