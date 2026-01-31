---
title: Orchestrator Settings (settings.yaml)
---

This document describes the configuration for the Scion orchestrator, managed through `settings.yaml` (or `settings.json`) files.

## Purpose
The orchestrator settings define the available infrastructure components (Runtimes), the tools that can be run (Harnesses), and how they are combined into environments (Profiles). It also contains client-side configuration for connecting to a Scion Hub or using cloud storage for persistence.

## Locations
- **Global**: `~/.scion/settings.yaml` (User-wide defaults)
- **Grove**: `.scion/settings.yaml` (Project-specific overrides)

## Key Sections
- **Profiles**: Named configurations that tie a runtime to a set of default environment variables and harness overrides.
- **Runtimes**: Configuration for execution backends like Docker, macOS Apple Virtualization, or Kubernetes.
- **Harnesses**: Definitions for agent harnesses, including container images and default volumes.
- **Hub**: Settings for connecting to a Scion Hub API.
- **Bucket**: Settings for cloud storage bucket persistence (e.g., GCS).
- **CLI**: General CLI behavior settings.

For a detailed walkthrough of orchestrator settings and environment variable substitution, see the [Local Governance Guide](/guides/local-governance/).

## Environment Variable Overrides
Most top-level settings can be overridden using environment variables with the `SCION_` prefix (e.g., `SCION_ACTIVE_PROFILE`).
