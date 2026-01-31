---
title: Server Configuration (Hub & Runtime Host)
---

This document describes the configuration for the Scion Hub (State Server) and the Scion Runtime Host services.

## Purpose
Server configuration controls the operational behavior of the Scion backend components in a "Hosted" or distributed architecture. This includes network settings, database connections, and security configurations.

## Locations
- **Config File**: `~/.scion/server.yaml` or `./server.yaml`.
- **Environment Variables**: Overridden using the `SCION_SERVER_` prefix.

## Key Sections
- **Hub**: Configuration for the central Hub API server (Port, Host, Timeout, CORS).
- **RuntimeHost**: Configuration for the execution host service (Enabled status, Hub endpoint, Host ID).
- **Database**: Connection settings for the persistence layer (SQLite or PostgreSQL).
- **Auth**: Settings for development authentication and tokens.
- **Logging**: Log level and format (Text or JSON) for system observability.

## Environment Variables
Server settings use a nested naming convention for environment variables. For example, `SCION_SERVER_HUB_PORT` maps to the `hub.port` setting, and `SCION_SERVER_DATABASE_DRIVER` maps to `database.driver`.
