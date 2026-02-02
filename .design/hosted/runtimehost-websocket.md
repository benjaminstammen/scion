# Runtime Host WebSocket Design

> **Status**: Consolidation document - centralizes WebSocket design currently scattered across multiple files.

This document consolidates the WebSocket-related design for communication between the Hub and Runtime Hosts. The WebSocket connection serves two primary purposes:

1. **Control Channel**: Hub-initiated commands to Runtime Hosts (for NAT/firewall traversal)
2. **PTY Streaming**: Bidirectional terminal access from browsers to agent containers

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Control Channel Protocol](#3-control-channel-protocol)
4. [PTY Streaming](#4-pty-streaming)
5. [Authentication](#5-authentication)
6. [Connection Lifecycle](#6-connection-lifecycle)
7. [Transport Selection](#7-transport-selection)
8. [Future Considerations](#8-future-considerations)

---

## 1. Overview

### 1.1 Problem Statement

Runtime Hosts often run in environments where the Hub cannot directly initiate HTTP connections:
- Developer laptops behind NAT
- On-premise servers with firewall restrictions
- Containers without public endpoints

Additionally, interactive terminal access requires low-latency bidirectional streaming that cannot be achieved with REST APIs alone.

### 1.2 Solution

The Runtime Host initiates a persistent WebSocket connection to the Hub. This connection:
- **Traverses NAT/firewalls**: Host-initiated outbound connections typically succeed
- **Enables bidirectional communication**: Hub can send commands; Host can send events
- **Supports stream multiplexing**: Multiple PTY sessions over a single connection

### 1.3 Design Goals

| Goal | Description |
|------|-------------|
| NAT traversal | Hosts behind NAT/firewalls can receive Hub commands |
| Low latency | Real-time PTY streaming for interactive use |
| Simplicity | Minimal connection management complexity |
| Resilience | Graceful reconnection with state reconciliation |
| Security | Authenticated connections with TLS |

---

## 2. Architecture

### 2.1 High-Level Flow

```
┌─────────────────┐                    ┌─────────────────┐
│   Scion Hub     │                    │  Runtime Host   │
│                 │◄───────────────────│  (behind NAT)   │
│                 │   WebSocket        │                 │
│  Control Plane  │   Control Channel  │  Host Agent     │
│                 │   (Host-initiated) │                 │
└─────────────────┘                    └─────────────────┘
        │                                      │
        │  Commands (Hub → Host)               │
        │  ◄────────────────────               │
        │  Events (Host → Hub)                 │
        │  ────────────────────►               │
        │                                      │
        │  Multiplexed Streams                 │
        │  ◄────────────────────►              │
        │  (PTY, Logs)                         │
```

### 2.2 Connection Topology

```
                    ┌──────────────────────────┐
                    │       Scion Hub          │
                    │                          │
                    │  ┌────────────────────┐  │
                    │  │ Control Channel    │  │
                    │  │ Manager            │  │
                    │  │                    │  │
Browser ──WS──►     │  │ ┌──────┐ ┌──────┐  │  │ ◄──WS── Runtime Host A
                    │  │ │Host A│ │Host B│  │  │
                    │  │ └──────┘ └──────┘  │  │ ◄──WS── Runtime Host B
                    │  └────────────────────┘  │
                    │                          │
                    │  ┌────────────────────┐  │
                    │  │ Stream Mapper      │  │
                    │  │ (browser WS →      │  │
                    │  │  host stream ID)   │  │
                    │  └────────────────────┘  │
                    └──────────────────────────┘
```

### 2.3 WebSocket Endpoints

| Endpoint | Direction | Purpose |
|----------|-----------|---------|
| `WS /api/v1/runtime-hosts/connect` | Host → Hub | Control channel (commands, events, streams) |
| `WS /api/v1/agents/{id}/pty` | Browser → Hub | PTY access (proxied to host) |
| `WS /api/v1/agents/{id}/events` | Browser → Hub | Agent status stream |
| `WS /api/v1/groves/{id}/events` | Browser → Hub | Grove-wide events |

---

## 3. Control Channel Protocol

### 3.1 Connection Establishment

**Endpoint:**
```
WS /api/v1/runtime-hosts/connect
```

**Prerequisites:**
- Host must have registered at least one grove via REST API
- Host must have a valid shared secret (from registration flow)

**HMAC Authentication Headers:**
```
X-Scion-Host-ID: host-abc123
X-Scion-Timestamp: 2025-01-24T10:00:00Z
X-Scion-Nonce: random-nonce-xyz
X-Scion-Signature: HMAC-SHA256(secret, "{hostId}:{timestamp}:{nonce}:GET:/api/v1/runtime-hosts/connect")
```

### 3.2 Initial Handshake

**Host → Hub (connect message):**
```json
{
  "type": "connect",
  "hostId": "string",
  "version": "1.2.3",
  "groves": [
    {
      "groveId": "string",
      "mode": "connected",
      "profiles": ["docker", "k8s-dev"]
    }
  ],
  "capabilities": {
    "webPty": true,
    "sync": true,
    "attach": true
  },
  "supportedHarnesses": ["claude", "gemini"],
  "resources": {
    "cpuAvailable": "4",
    "memoryAvailable": "8Gi"
  }
}
```

**Hub → Host (connected acknowledgment):**
```json
{
  "type": "connected",
  "hostId": "string",
  "hubTime": "2025-01-24T10:00:00Z",
  "groves": [
    {
      "groveId": "string",
      "groveName": "string",
      "mode": "connected"
    }
  ]
}
```

### 3.3 Message Envelope

All messages use a consistent envelope structure:

**Command (Hub → Host):**
```json
{
  "type": "command",
  "id": "cmd-uuid",
  "command": "create_agent",
  "payload": { ... }
}
```

**Response (Host → Hub):**
```json
{
  "type": "response",
  "id": "cmd-uuid",
  "success": true,
  "payload": { ... },
  "error": null
}
```

**Event (Host → Hub):**
```json
{
  "type": "event",
  "event": "agent_status",
  "payload": { ... }
}
```

### 3.4 Command Types

| Command | Description |
|---------|-------------|
| `create_agent` | Create and start a new agent |
| `start_agent` | Start a stopped agent |
| `stop_agent` | Stop a running agent |
| `delete_agent` | Delete an agent |
| `exec` | Execute a command in an agent |
| `open_stream` | Open a multiplexed stream (PTY, logs) |
| `close_stream` | Close a multiplexed stream |
| `ping` | Keepalive ping |

### 3.5 Event Types

| Event | Description |
|-------|-------------|
| `agent_status` | Agent status change |
| `agent_created` | Agent creation completed |
| `agent_deleted` | Agent deletion completed |
| `heartbeat` | Periodic health check |
| `resource_update` | Resource availability changed |

---

## 4. PTY Streaming

### 4.1 Stream Multiplexing

PTY sessions are multiplexed over the control channel using stream frames. Each stream has a unique `streamId` assigned by the Hub.

**Open Stream Command (Hub → Host):**
```json
{
  "type": "command",
  "id": "cmd-456",
  "command": "open_stream",
  "payload": {
    "streamId": "stream-xyz",
    "agentId": "agent-123",
    "streamType": "pty",
    "options": {
      "cols": 120,
      "rows": 40
    }
  }
}
```

**Stream Data Frame:**
```json
{
  "type": "stream",
  "streamId": "stream-xyz",
  "data": "base64-encoded-bytes"
}
```

**Stream Close:**
```json
{
  "type": "stream_close",
  "streamId": "stream-xyz",
  "reason": "client disconnected"
}
```

### 4.2 Browser PTY Flow

```
┌─────────┐         ┌─────────┐         ┌─────────────┐         ┌───────────┐
│ Browser │         │   Hub   │         │ Runtime Host│         │ Container │
└────┬────┘         └────┬────┘         └──────┬──────┘         └─────┬─────┘
     │                   │                     │                      │
     │ WS connect        │                     │                      │
     │ /agents/{id}/pty  │                     │                      │
     │──────────────────►│                     │                      │
     │                   │                     │                      │
     │                   │ open_stream         │                      │
     │                   │ (streamId=xyz)      │                      │
     │                   │────────────────────►│                      │
     │                   │                     │                      │
     │                   │                     │ tmux attach          │
     │                   │                     │─────────────────────►│
     │                   │                     │                      │
     │                   │ stream response     │                      │
     │                   │◄────────────────────│                      │
     │                   │                     │                      │
     │ ◄─────────────────────────────────────► │ ◄───────────────────►│
     │         bidirectional data flow         │   tmux I/O           │
     │                   │                     │                      │
```

### 4.3 PTY Message Format

**Browser ↔ Hub (client WebSocket):**
```json
{
  "type": "data",
  "data": "base64-encoded-bytes"
}
```

**Resize Message:**
```json
{
  "type": "resize",
  "cols": 120,
  "rows": 40
}
```

### 4.4 Stream ID Mapping

The Hub maintains a mapping between:
- Client WebSocket connections (browser PTY endpoints)
- Stream IDs on Runtime Host control channels

```
Browser WS Connection ←→ streamId ←→ Host Control Channel
```

This allows multiple browsers to attach to different agents via the same Runtime Host control channel.

---

## 5. Authentication

### 5.1 Control Channel Authentication

The control channel WebSocket upgrade request is authenticated using HMAC:

```
HMAC-SHA256(shared_secret, "{hostId}:{timestamp}:{nonce}:GET:{path}")
```

**Headers:**
| Header | Description |
|--------|-------------|
| `X-Scion-Host-ID` | Runtime Host identifier |
| `X-Scion-Timestamp` | RFC 3339 timestamp |
| `X-Scion-Nonce` | Random nonce for replay prevention |
| `X-Scion-Signature` | HMAC signature |

### 5.2 Session-Based Trust

Once the WebSocket is established with HMAC authentication:
- **Hub → Host commands** use session-based trust (no per-message signing)
- **Host → Hub requests** requiring authorization must use separate HMAC-authenticated HTTP requests

**Rationale:**
- WebSocket runs over TLS, providing transport-level integrity
- Initial connection establishes host identity
- Similar trust model to SSH after key exchange
- Avoids per-message cryptographic overhead

### 5.3 Browser WebSocket Authentication

Browsers cannot set custom HTTP headers on WebSocket connections. Two methods are supported:

**1. Query Parameter Token:**
```
WS /api/v1/agents/{id}/pty?token=<bearer-token>
```

**2. Ticket-Based Auth (Recommended):**
```
POST /api/v1/auth/ws-ticket
→ { "ticket": "...", "expiresAt": "..." }

WS /api/v1/agents/{id}/pty?ticket=<ticket>
```

Tickets are single-use and expire after 60 seconds.

---

## 6. Connection Lifecycle

### 6.1 Heartbeat

- **Interval**: Host sends heartbeat every 30 seconds
- **Timeout**: Hub marks host as `disconnected` after 90 seconds without heartbeat
- **Format**:
  ```json
  {
    "type": "event",
    "event": "heartbeat",
    "payload": {
      "status": "online",
      "agentCount": 3,
      "resources": { ... }
    }
  }
  ```

### 6.2 Reconnection

- **Backoff**: Exponential backoff (1s, 2s, 4s, ... max 60s)
- **Session Resumption**: On reconnect, Hub sends list of expected agents for reconciliation
- **Stream Recovery**: Active streams are terminated on disconnect; clients must re-attach

### 6.3 Graceful Shutdown

**Host Shutdown:**
1. Host sends `shutting_down` heartbeat
2. Hub marks host as `offline`
3. Hub fails pending commands for this host
4. Active streams are terminated

**Hub Shutdown:**
1. Hub sends disconnect to connected hosts
2. Hosts enter reconnection loop
3. Commands queue up locally (if host supports it)

---

## 7. Transport Selection

The Hub supports two transport modes for communicating with Runtime Hosts:

| Transport | Use Case | Selection Criteria |
|-----------|----------|-------------------|
| WebSocket Control Channel | Hosts behind NAT/firewalls | Host has active WS connection |
| Direct HTTP | Hosts with reachable endpoints | Host has registered `endpoint` URL |

**Selection Logic:**

```
When Hub needs to send command to Host:
1. If Host has active control channel → use WebSocket
2. If Host has registered endpoint and status == "online" → attempt Direct HTTP
3. Otherwise → return 502 runtime_error
```

**Command Mapping:**

| Control Channel Command | Direct HTTP Equivalent |
|------------------------|----------------------|
| `create_agent` | `POST /api/v1/agents` |
| `stop_agent` | `POST /api/v1/agents/{id}/stop` |
| `delete_agent` | `DELETE /api/v1/agents/{id}` |
| `exec` | `POST /api/v1/agents/{id}/exec` |
| `open_stream` | `GET /api/v1/agents/{id}/attach` (WebSocket) |

---

## 8. Future Considerations

### 8.1 Alternative Transport: gRPC / HTTP/2

The current WebSocket + JSON approach could be replaced with gRPC for improved efficiency:

| Aspect | Current (WS+JSON) | gRPC/HTTP/2 |
|--------|-------------------|-------------|
| Framing | Manual JSON envelopes | Native protobuf framing |
| Multiplexing | Custom `streamId` management | Native HTTP/2 streams |
| Binary data | Base64 overhead (~33%) | Native binary transport |
| Type safety | JSON schema / manual validation | Proto definitions with codegen |
| Browser support | Native WebSocket API | Requires gRPC-Web proxy |

**Trade-offs:**
- gRPC-Web requires a proxy (Envoy) for browser support
- JSON is human-readable and easier to debug
- gRPC provides stronger API contracts and better tooling

**Decision**: Start with WebSocket + JSON for simplicity and universal browser support. Consider gRPC migration when performance or type safety becomes a bottleneck.

### 8.2 Hybrid: Command Queue + On-Demand WebSocket

For horizontal scalability, a hybrid approach could decouple command delivery from streaming:

```
┌────────────────────────────────────────────────────────────────┐
│                      Hybrid Architecture                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   CRUD Operations (Polling)          Interactive Streams (WS)  │
│   ─────────────────────────          ────────────────────────  │
│                                                                 │
│   Hub ──► Queue ──► Host             Browser ◄──► Hub ◄──► Host│
│       (DB-backed)                         (WebSocket relay)    │
│                                                                 │
│   • create_agent                     • PTY attachment          │
│   • stop_agent                       • Live log streaming      │
│   • delete_agent                                               │
│   • exec (async)                                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Benefits:**
- Hub instances are stateless (any can write commands, any can serve polls)
- Commands persist through Hub restarts
- Easier horizontal scaling

**Limitations:**
- Polling introduces latency (5-10 seconds)
- Requires on-demand WebSocket for PTY

**Decision**: Deferred. Current design uses persistent WebSocket. Revisit if horizontal scaling becomes a priority.

### 8.3 Open Questions

1. **Stream-ready WebSocket**: Should hosts maintain a persistent "stream-ready" WebSocket, or connect on-demand per stream?
2. **Stream token expiration**: How to handle cleanup of unused stream tokens?
3. **WebRTC**: Can browser-to-host PTY bypass the Hub using WebRTC in some scenarios?

---

## Related Documentation

| Document | Relevant Sections |
|----------|-------------------|
| [hub-api.md](hub-api.md) | Section 8 (WebSocket Endpoints), Section 11 (Host Control Plane Protocol), Section 15 (Future Considerations) |
| [runtime-host-api.md](runtime-host-api.md) | Section 3.2 (Control Channel), Section 4.2 (Attach PTY) |
| [auth/runtime-host-auth.md](auth/runtime-host-auth.md) | Section 10.4 (Hub-to-Host Communication), Section 10.5 (WebSocket Message Auth) |
| [server-implementation-design.md](server-implementation-design.md) | Section 12.5 (WebSocket Proxying) |
| [web-frontend-design.md](web-frontend-design.md) | Terminal component WebSocket integration |
