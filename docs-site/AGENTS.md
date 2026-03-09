# Scion Documentation Meta-Guide

This guide outlines the organizational philosophy, architecture, and curation standards for the Scion documentation project. It serves as a blueprint for both human and AI contributors to ensure a consistent and navigable documentation experience.

## 1. Documentation Dimensions

Scion documentation must navigate three primary dimensions:

| Dimension | Variants |
| :--- | :--- |
| **User Persona** | Casual Local User, Power User, Team Developer, Platform Ops, Contributor |
| **Operational Mode** | Solo (Local-only, zero-config), Hosted (Distributed, Hub-centric) |
| **Interface** | CLI (Primary), Web Dashboard (Visualization & Control) |

## 2. Content Architecture

The documentation uses a persona-driven architecture that segregates content based on user goals:

### 1. Introduction & Foundations
*   **Target**: Everyone
*   **Content**: What is Scion, core concepts, supported harnesses, glossary, release notes.

### 2. Getting Started
*   **Target**: Casual Local User
*   **Content**: Quickstart (installation, API keys), basic workflows (start/stop/attach, Tmux sessions), workspace basics.

### 3. Advanced Local Usage
*   **Target**: Power User
*   **Content**: Agent configuration (`settings.json`), templates & roles, harness deep dives (agent credentials, custom images), workstation server mode.

### 4. Hub User Guide
*   **Target**: Team Developer
*   **Content**: Connecting to a Hub, using the Web Dashboard, remote workflows, secret management.

### 5. Hub Administration
*   **Target**: Platform Ops
*   **Content**: Architecture deep dive, deploying the Hub, provisioning Brokers, security, observability & operations.

### 6. Contributor's Guide
*   **Target**: Core Developer
*   **Content**: Local development setup, architecture & codebase tour, design catalog.

### 7. Technical Reference
*   **Target**: Everyone
*   **Content**: CLI Reference, API Reference, Configuration Schema.

## 3. Handling the Intersections

### Interface Intersection (CLI vs. Web)
*   **Action-Oriented Documentation**: For any user task, the documentation should provide instructions for both interfaces using a tabbed or side-by-side format:
    *   **CLI**: `scion start ...`
    *   **Web**: Navigation path in the UI.

## 4. Starlight Sidebar Configuration

The `astro.config.mjs` sidebar reflects this hierarchy. When adding new files, ensure they are registered in the sidebar:

```javascript
sidebar: [
    {
        label: 'Introduction & Foundations',
        items: [ /* ... */ ],
    },
    {
        label: 'Getting Started',
        items: [ /* ... */ ],
    },
    {
        label: 'Advanced Local Usage',
        items: [ /* ... */ ],
    },
    {
        label: 'Hub User Guide',
        items: [ /* ... */ ],
    },
    {
        label: 'Hub Administration',
        items: [ /* ... */ ],
    },
    // ...
]
```

## 5. Curation Standards

1.  **Code-First Truth**: Documentation should be updated in the same PR as the feature implementation.
2.  **Source of Truth**:
    *   Always check `.design/` for architectural intent.
    *   Consult `pkg/` for implementation details.
    *   Verify CLI flags in `cmd/`.
3.  **Persona-Specific Tone**:
    *   *User docs* should be practical and task-oriented.
    *   *Admin docs* should be technical and detail-oriented.
    *   *Reference docs* should be exhaustive and dry.
4.  **Diagrams**: Use **D2** for all diagrams. Include them in `d2` code blocks within Markdown files.
5.  **Cross-Linking**: Link from high-level guides to specific references to avoid duplication.

## 6. Agent Workflow & Verification

### Status Reporting
Before asking the user a question or completing a task, use the following commands to update the agent's status:

*   **Asking User**: `sciontool status ask_user "<question>"`
*   **Task Completed**: `sciontool status task_completed "<task title>"`

### Verification Steps
*   **Link Check**: Ensure all relative links between documents are valid.
*   **Sidebar Verification**: Ensure new pages are added to `docs-site/astro.config.mjs`.
*   **Formatting**: Use standard Markdown. Starlight supports [Callouts](https://starlight.astro.build/guides/authoring-content/#asides) (Note, Tip, Caution, Danger).

**Note on D2**: The `d2` CLI may not be available in all environments. If you cannot run `./check-d2.sh`, ensure your D2 syntax is correct according to [D2 documentation](https://d2lang.com/tour/intro).
