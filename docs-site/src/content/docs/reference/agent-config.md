---
title: Agent & Template Configuration (scion-agent.json)
---

This document describes the configuration for Scion agent blueprints (templates) and individual agent instances.

## Purpose
The `scion-agent.json` file specifies how a particular agent should be executed. It defines the harness to use, the container image, environment variables, volume mounts, and runtime-specific settings like Kubernetes resources.

## Locations
- **Templates**: `.scion/templates/<template-name>/scion-agent.json`
- **Agent Instances**: `.scion/agents/<agent-name>/scion-agent.json`

## Key Fields
- **harness**: The name of the harness to use (e.g., `gemini`, `claude`).
- **image**: The container image to run (overrides the harness default).
- **env**: A list of environment variables to inject into the agent.
- **volumes**: Host and workspace volumes to mount into the agent.
- **kubernetes**: Resource requests, limits, and namespace settings for the Kubernetes runtime.
- **useTmux**: Whether to run the agent in a tmux session for interactivity.

## Resolution
When an agent is created from a template, the template's `scion-agent.json` serves as the base. Instance-specific overrides can be applied during the `scion start` command and are persisted in the agent's directory.
