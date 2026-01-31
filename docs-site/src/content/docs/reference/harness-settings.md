---
title: Harness-Specific Settings
---

This document describes how to configure individual LLM tools and harnesses inside a Scion agent.

## Purpose
While Scion manages the orchestration and execution of containers, the tools running *inside* those containers (like the Gemini CLI or Claude Code) often have their own configuration systems.

## Locations
Each agent has a dedicated "Home" directory that is mounted into the container. Harness-specific settings are typically found in a hidden subdirectory:
- **Gemini**: `/home/gemini/.gemini/settings.json`
- **Claude**: `/home/claude/.claude.json` (or similar)
- **Opencode**: `/home/opencode/opencode.json`

## Seeding from Templates
When an agent is created, Scion copies the contents of the template's `home/` directory into the new agent's home directory. This allows you to pre-configure tools, tools allowlists, and default prompts at the template level.

## Key Concepts
- **Tools**: Allowlists of local or remote functions the LLM is permitted to call.
- **Profiles**: Harness-level profiles (distinct from Scion profiles) that control model parameters.
- **Credentials**: How API keys are injected and stored within the harness-specific configuration.
