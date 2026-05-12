# Architecture

Fieldstone is a set of independently deployable Go services sharing a single PostgreSQL
instance and NATS JetStream message bus.

## Service map

```
Public internet
       │
  ┌────▼────┐
  │  Caddy  │  TLS termination, reverse proxy
  └────┬────┘
       │
  ┌────▼────┐
  │ Gateway │  JWT validation, rate limiting, request routing
  └────┬────┘
       │  HTTP (sync)
   ┌───┴──────────────────┐
   │                      │
┌──▼──┐  ┌────────┐  ┌────▼───┐
│Permits│ │Requests│  │Records │  Domain services
└──┬──┘  └───┬────┘  └───┬────┘
   └──────────┴───────────┘
              │ NATS JetStream (async)
        ┌─────┴──────┐
   ┌────▼────┐  ┌────▼────┐  ┌──────────┐  ┌────────┐
   │  Audit  │  │Webhooks │  │ Identity │  │Workflow│
   └─────────┘  └─────────┘  └──────────┘  └────────┘
```

## Data isolation

All services share one PostgreSQL instance but each owns a separate schema:

| Service  | Schema   | DSN search_path |
|----------|----------|-----------------|
| identity | identity | identity,public  |
| permits  | permits  | permits,public   |
| requests | requests | requests,public  |
| records  | records  | records,public   |
| audit    | audit    | audit,public     |
| webhooks | webhooks | webhooks,public  |

Services never reference another service's schema in queries.

## Event flow

1. HTTP request arrives at a domain service handler
2. Handler writes to its database (commits transaction)
3. After commit, handler puts event on a buffered channel (size 1000)
4. Background goroutine drains the channel, publishing to NATS JetStream
5. Failed NATS publish is logged at ERROR but does NOT fail the HTTP request
6. Subscribers (audit, webhooks, notify, extensions) receive events independently

## Workflow validation

Before any status change, domain services call the workflow service:

```
POST /v1/workflow/:resource_type/validate
{"from": "submitted", "to": "under_review", "role": "reviewer"}
→ 200 if allowed, 422 if not
```

Status transition logic lives in YAML config, not in domain service code.
