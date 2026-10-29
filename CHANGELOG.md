 # Changelog

 All notable changes to this project are documented in this file.

 The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
 and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

 ## [Unreleased]

 ### Fixed

 - Helm: the "Add" button on the repositories page showed a stray "≥" instead of "+".
 - Helm: repository badges (`oci`, `auth`, `no TLS verify`) and `deprecated` in the catalog are now translated.
 - Docs: the 0.1.0 notes said cancel is available for rollback and uninstall; it is available for
   install and upgrade only (the Helm SDK takes no context for the other two).

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

 [Unreleased]: https://github.com/VadimIl224/kubron/compare/v0.1.0...HEAD
 [0.1.0]: https://github.com/VadimIl224/kubron/releases/tag/v0.1.0
