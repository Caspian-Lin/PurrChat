# OneBot Bot API Operations

`GET /api/bot/v1/health` returns an unauthenticated aggregate health snapshot for the Universal WebSocket service. It never returns credentials, bot IDs, message contents, or per-installation data. Per-bot connection details remain owner-only at `GET /api/bots/:id/ws-status`.

## Metrics

The health response contains counters for accepted/rejected connections, frames read/written, event publish/delivery/drop attempts, action lifecycle outcomes, queue overflows, protocol errors, and total Action latency in nanoseconds. `active` and `queue_depth` are gauges.

Alert on sustained non-zero `events_dropped`, `queue_overflows`, or rapidly increasing `protocol_errors`. Investigate `action_failed / action_completed`, and calculate mean Action latency as `action_latency_nanoseconds / action_completed` when the completed count is non-zero.

## Default Capacity And SLOs

| Setting | Default |
| --- | --- |
| Total connections | 1000 |
| Connections per Bot | 3 |
| Concurrent Actions per connection | 8 |
| Outbound queue per connection | 64 |
| Maximum frame/message size | 16 KiB |
| Action timeout | 30 s |
| Read timeout | 90 s |

Operational targets: maintain zero intentional event drops, keep queue overflows below 0.1% of event delivery attempts, and investigate any Action timeout. Delivery is at-least-once only after the client ACKs an event; reconnect with `resume_from` and retry Actions using a stable `client_message_id` where applicable.

## Safe Diagnostics

Use `trace_id` from an Action response and the credential prefix recorded by credential audit logs to correlate incidents. Do not log bearer tokens, authorization headers, complete message payloads, or raw credentials. The fake client at `apps/backend/internal/botws/testkit` is intended for CI and local protocol tests; it connects only to a supplied endpoint and has no QQ, NapCat, or public-network dependency.
