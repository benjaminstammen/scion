# Broker Plugin Design

This document covers the design concerns specific to message broker plugins — the RPC interface, message delivery patterns, authentication, and operational behavior. For the general plugin system architecture (discovery, loading, lifecycle, registry), see [scion-plugins.md](scion-plugins.md).

## Broker Plugin RPC Interface

The plugin-side broker interface is intentionally minimal:

```go
type MessageBrokerPlugin interface {
    Configure(config map[string]string) error
    Publish(ctx context.Context, topic string, msg *messages.StructuredMessage) error
    Subscribe(pattern string) error
    Unsubscribe(pattern string) error
    Close() error
}
```

The `broker.MessageBroker` interface is small and maps well to `net/rpc`. The main design challenge — `Subscribe()` uses a callback-based `MessageHandler` that cannot cross process boundaries — is solved by having the plugin deliver inbound messages via the hub API (see below).

### Host-Side Adapter

The host-side adapter wraps the plugin RPC client to satisfy the existing `broker.MessageBroker` interface:

```go
type brokerPluginClient struct {
    client *rpc.Client
}

func (b *brokerPluginClient) Publish(ctx context.Context, topic string, msg *StructuredMessage) error {
    return b.client.Call("Plugin.Publish", &PublishArgs{Topic: topic, Msg: msg}, nil)
}

func (b *brokerPluginClient) Subscribe(pattern string, handler MessageHandler) (Subscription, error) {
    // handler is not forwarded to the plugin — inbound delivery happens via hub API
    err := b.client.Call("Plugin.Subscribe", pattern, nil)
    // Return a Subscription that calls Plugin.Unsubscribe on cancel
}
```

The adapter's `Subscribe()` tells the plugin to start listening on the external broker. The `MessageHandler` callback is not used for plugin brokers — messages arrive via the hub API instead, where the hub dispatches them through its existing `DispatchAgentMessage()` path.

## Message Delivery Architecture

### Subscription Delivery via Hub API

Rather than polling or reverse RPC, the broker plugin delivers inbound messages through the hub's existing API.

The hub already exposes authenticated endpoints for message delivery:
- `POST /api/v1/agents/{agentId}/message` — deliver to a specific agent
- `POST /api/v1/groves/{groveId}/broadcast` — broadcast to a grove

**Message flow:**

```
Outbound (hub -> external):
  Hub -> broker.Publish() -> [RPC] -> Plugin -> NATS/Redis

Inbound (external -> hub):
  NATS/Redis -> Plugin -> hub API (POST /api/v1/broker/inbound) -> Hub dispatches to agent
```

This approach:
- Reuses existing authenticated infrastructure — no new transport to build
- The hub API already handles agent dispatch, fan-out, and audit logging
- No streaming or polling required over the RPC boundary
- Keeps the plugin RPC interface simple: only `Publish()`, `Subscribe()`, `Unsubscribe()`, and `Close()`

### Inbound Message Routing

When the broker plugin receives a message from the external system (NATS/Redis), it needs to know where to deliver it. The recommended approach is **pass-through delivery**: the plugin delivers all inbound messages to a dedicated hub API endpoint with the original topic, and the hub handles routing internally.

**New internal API endpoint:**

```
POST /api/v1/broker/inbound
Body: { "topic": "<original-topic>", "message": <StructuredMessage> }
```

The hub's routing logic parses the topic (e.g., `scion.grove.<groveId>.agent.<agentSlug>.messages`) and dispatches to the appropriate agent or grove. This keeps routing logic in the hub where it belongs and avoids duplicating topic-parsing logic in every plugin.

**Note:** This routing logic is not strictly plugin-specific — it's a general broker subsystem concern. The topic convention and routing rules should be defined as part of the broader messaging architecture, with the plugin inbound endpoint being one consumer of that routing layer.

## Plugin Authentication for Hub API Callbacks

**Decision: Reuse broker HMAC authentication** with identity-aware logging.

When a broker plugin delivers inbound messages via the hub API, it authenticates using the existing broker HMAC mechanism. The plugin receives broker credentials as part of its `Configure()` config map.

**Logging concern:** Plugin API calls must not appear in logs as if they came from the runtime broker itself. To distinguish plugin activity:
- The plugin should include a `X-Scion-Plugin-Name` header (or similar identifier) on its hub API calls
- The hub's request logging should surface this header to differentiate plugin-originated requests from runtime broker requests
- This is a logging/observability concern, not a security boundary — the HMAC credentials grant the same access either way

If the identity conflation causes issues beyond logging (e.g., rate limiting, audit trails), a dedicated plugin credential type can be introduced later.

## Circular Message Delivery Prevention

When a broker plugin delivers an inbound message via the hub API, the hub must not re-publish that message through the broker plugin (creating an infinite loop).

**Path delineation:**
- **Outbound path**: Hub -> `MessageBrokerProxy.PublishMessage()` -> `broker.Publish()` -> plugin RPC -> external system
- **Inbound path**: External system -> plugin -> hub API -> `DispatchAgentMessage()` directly (bypasses broker)

**Primary responsibility: the plugin.** The broker plugin must filter out messages that originated from the hub before delivering them back. The plugin has the most context about the external broker's implementation-specific metadata (message headers, sender fields, deduplication IDs) and is best positioned to identify echoes. Strategies include:
- Tagging outbound messages with a scion origin marker and filtering on inbound
- Using separate external broker topics/channels for inbound vs outbound
- Maintaining a message ID seen-set with TTL

**Secondary defense: host-side circuit breaker.** As a safety net, the hub should implement rate-limit-based circuit breaking on the plugin inbound API endpoint. If the inbound message rate exceeds a configurable threshold (suggesting a delivery loop), the circuit breaker trips and rejects further messages with a backoff period. This protects against plugin bugs or misconfiguration without requiring the hub to understand implementation-specific echo detection.

## RPC Transport Considerations

With hub API callbacks handling inbound message delivery, the `net/rpc` streaming limitation is fully mitigated — the plugin RPC interface is purely request/response (Publish, Subscribe, Unsubscribe, Close, Configure). This validates the `net/rpc` choice for broker plugins.

If future requirements add RPC methods that benefit from streaming (e.g., bulk publish, health telemetry feeds), switching broker plugins to gRPC may be warranted. This should be monitored during the NATS plugin implementation.

## Plugin Behavior During Hub Unavailability

If the hub API is temporarily unavailable when the broker plugin tries to deliver an inbound message, the plugin should:

1. **Buffer messages with bounded limits** — a configurable in-memory buffer (default: 1000 messages or 10MB, whichever is hit first) prevents unbounded memory growth
2. **Retry with exponential backoff** — standard retry pattern (1s, 2s, 4s, ... up to 30s) for transient hub API failures
3. **Drop oldest on buffer overflow** — when the buffer is full, drop the oldest undelivered messages and log warnings with drop counts
4. **Report health degradation** — the plugin should respond to `GetInfo()` / health check RPC calls with degraded status when buffering or dropping messages, allowing the hub to surface this in operational dashboards

The specifics will be refined during Phase 2 (NATS plugin implementation) when real failure modes are encountered. The initial implementation can start with drop-and-warn behavior and add buffering as needed.

## Multiple Active Brokers

The current design loads one active message broker. Future support for multiple active brokers would follow a gateway/router pattern:

- Multiple broker plugins loaded and active simultaneously (e.g., NATS for inter-agent messaging, Redis for notifications)
- A routing layer determines which broker handles each `Publish()` and `Subscribe()` call based on topic patterns, message types, or explicit configuration
- The plugin manager's registry (keyed by `type:name`) already supports loading multiple broker plugins — the missing piece is the routing logic

This is deferred but the registry and plugin lifecycle design intentionally accommodate it.

## Active Broker Selection

For message brokers, the active broker is selected in server config:

```yaml
# In hub/broker server config
message_broker: nats   # selects the "nats" plugin (or "inprocess" for built-in)
```

The hub server resolves this through the plugin registry. If the named broker is `inprocess`, the built-in implementation is used. Otherwise, the plugin manager dispenses the named broker plugin and the host-side adapter wraps it as a `broker.MessageBroker`.

## Implementation Phases (Broker-Specific)

### Phase 2: Message Broker Plugins (depends on Phase 1 plugin infrastructure)
- NATS broker plugin (first external implementation)
- Host-side adapter wrapping plugin RPC client as `broker.MessageBroker`
- Hub API inbound endpoint (`/api/v1/broker/inbound`) for plugin message delivery
- Plugin authentication via broker HMAC with plugin identity headers
- Echo filtering in plugin and circuit breaker on hub inbound endpoint
- Inbound message routing logic in hub (topic parsing and dispatch)
- Test the full lifecycle: discovery, loading, configuration, publish, subscribe, inbound delivery, shutdown

## Related Design Documents

- [Plugin System](scion-plugins.md) - General plugin architecture and management
- [Message Broker](hosted/hub-messaging.md) - Current messaging architecture
- [Hosted Architecture](hosted/hosted-architecture.md) - Hub/broker separation
