# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Desktop app (Wails + Go + Vue) with multi-cluster management and namespace scoping.
- Cluster dashboard: health, capacity, problem pods, recent changes, expiring certificates.
- Resource browser for built-in Kubernetes types and CRDs, with a Monaco YAML editor.
- Pod tools: logs, exec, port forwarding. Deployment scale, restart and rollback.
  Node cordon, drain and shell.
- Optional metrics stack (Prometheus, kube-state-metrics, node-exporter) with charts.
- Helm: releases, history and diff, rollback, chart catalog, install with dry-run, upgrade.
- Argo CD: applications, ApplicationSets, projects, sync and other actions, diff, graph.
- Local users with `admin` / `operator` / `user` roles, audit log with rotation,
  auto-logout on inactivity.
- Public extension API (`pkg/ext`) for external modules.
- Experimental web mode (`cmd/kubron-server`).
- Project docs: SECURITY.md, CODE_OF_CONDUCT.md, issue and pull request templates,
  `.editorconfig`, `.gitattributes`.
  