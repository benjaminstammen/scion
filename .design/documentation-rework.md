# Scion Documentation Rework

## Overview

The current documentation structure divides content strictly by "Developer" vs "Administrator" and then tries to weave the "Solo" vs "Hosted" operational modes within those sections. As the capabilities of Scion have expanded, this structure risks overwhelming users who only care about a specific operational mode or level of complexity.

This rework proposes a persona-driven documentation architecture that clearly segregates content based on the user's specific goals and environment:

1.  **Casual Local User**: "I just want an AI agent to help me code right now."
2.  **Advanced User**: "I want to customize my local agents, create templates, and tweak harnesses."
3.  **Hub User (Team Developer)**: "I am connecting to my company's Scion Hub to run agents remotely."
4.  **Hub Admin (Platform Ops)**: "I need to deploy, secure, and manage the Scion Hub and Broker infrastructure."

## Rationale for Change

- **Reduced Cognitive Load**: A casual local user shouldn't have to wade through Kubernetes deployment concepts or Hub authentication flows to figure out how to start a local Gemini agent.
- **Clearer Progression**: Users can naturally "level up" from a Casual Local User to an Advanced User, and eventually to a Hub User.
- **Better Feature Discovery**: Advanced features (like custom templates, runtime configurations) are currently mixed with basic concepts. Grouping them by persona clarifies what is "standard" vs "advanced."
- **Targeted Entry Points**: Different users enter the documentation with different intentions. Explicitly naming the personas in the navigation helps users self-select the right path immediately.

## Proposed New Architecture

### 1. Introduction & Foundations
*A quick landing zone for everyone to understand what Scion is.*
*   **What is Scion?**: Core value proposition.
*   **Architecture 101**: Brief overview of Solo vs. Hosted modes.
*   **Glossary**: Quick definitions (Grove, Agent, Harness, Hub, Broker).

### 2. Getting Started (The Casual Local User)
*Targeted at the single developer wanting immediate value on their local machine. Zero-config focus.*
*   **Quickstart**: Installation (macOS, Linux), configuring an API key, and starting the first agent.
*   **Basic Workflows**:
    *   Starting and stopping agents.
    *   Viewing agent output and logs.
    *   Interacting with an agent (`scion attach`).
*   **Workspace Basics**: How Scion manages Git worktrees automatically.

### 3. Advanced Local Usage (The Power User)
*Targeted at developers who want to push the boundaries of local agents.*
*   **Agent Configuration**: Deep dive into `.scion/settings.json` and CLI flags.
*   **Templates & Roles**: Creating and managing custom agent templates (e.g., Code Reviewer vs. Test Writer).
*   **Harness Deep Dive**: Configuring specific harnesses (Gemini, Claude) and tweaking system prompts.
*   **Runtimes**: Switching between Docker, Apple Virtualization, and podman.
*   **Resource Management**: Setting limits on local agent containers.

<!-- feedback - this section should include details on starting and using the "workstation server mode" - noting how to configure network bridge (currently automatic for podman) this can be a 'bonus' content, with a pointer to the hub user guide as being mostly relevant to this workstation server mode -->

### 4. Hub User Guide (The Team Developer)
*Targeted at developers using a centrally managed Scion Hub. Assumes someone else set it up.*
*   **Connecting to a Hub**: Login flows (OAuth), selecting a Hub context.
*   **The Web Dashboard**: Navigating the UI, viewing team Groves and Agents.
*   **Remote Workflows**: Dispatching agents to remote brokers from the CLI or Web.
*   **Secret Management**: Using Hub-managed secrets instead of local environment variables.
*   **Collaboration**: Viewing other team members' agents and sharing context.

### 5. Hub Administration (Platform Ops)
*Targeted at DevOps, Platform Engineers, and System Administrators.*
*   **Architecture Deep Dive**: Hub, Brokers, Database, and Web Server interactions.
*   **Deploying the Hub**:
    *   Local/VM deployment.
    *   Kubernetes deployment.
    *   Configuring persistence (SQLite vs. external databases).
*   **Provisioning Brokers**: Setting up runtime brokers to execute workloads.
*   **Identity & Security**:
    *   Configuring OAuth providers.
    *   Role-Based Access Control (RBAC) and permissions.
    *   Managing credentials securely.
*   **Observability & Operations**: Exporting metrics, centralized logging, and monitoring agent health.

### 6. Contributor's Guide
*Targeted at people contributing to Scion itself.*
*   **Local Development Setup**: Compiling Scion, running the Vite dev server.
*   **Architecture & Codebase Tour**: Overview of `pkg/`, `cmd/`, and `web/`.
*   **Design Catalog**: Index of architectural decision records (ADRs) and design specs.

### 7. Technical Reference
*Exhaustive, machine-generated or highly structured documentation.*
*   **CLI Command Reference**: Auto-generated from Cobra.
*   **API Reference**: REST and WebSocket endpoints for Hub/Broker.
*   **Configuration Schema**: Complete reference for `settings.json` and Hub config files.

## Implementation Strategy

1.  **Restructure the Astro/Starlight Sidebar**: Update `docs-site/astro.config.mjs` to reflect the new top-level categories.
2.  **Migrate Existing Content**: Move existing Markdown files into the new directory structure, updating internal links.
3.  **Content Gap Analysis**: Identify areas where recent code changes (e.g., new Broker auth flows, web dashboard features) lack coverage in the new structure and create placeholder pages.
4.  **Rewrite Introductions**: Ensure the landing page of each section clearly states *who* the section is for and *what* they will learn.