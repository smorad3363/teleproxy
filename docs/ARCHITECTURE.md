# Architecture

## System boundary

Teleproxy is split into two logical planes.

### Control Plane

A Go service owns the Web Panel, internal HTTP API, Telegram bot orchestration, SQLite persistence, users, credits, referrals, sponsors, admins/RBAC, audit, settings, backups metadata, health and node management.

### Proxy Data Plane

`telemt` owns MTProto traffic, per-user proxy secrets, runtime traffic counters, quota/expiry enforcement exposed by the upstream integration, Sponsor/AdTag behavior, Middle Proxy behavior and proxy runtime health.

## Lifecycle invariant

Control Plane failure or restart must not stop a healthy Proxy Data Plane. A Proxy Node failure must not make the Web Panel or Bot unavailable.

This means:

- no raw Docker socket inside the Control Plane container;
- host-level lifecycle changes are performed by a narrow allowlisted manager/CLI;
- telemt runs as a separate service/container with its own restart policy and healthcheck;
- control-plane readiness may report a degraded proxy dependency without failing its own liveness.

## Control Plane stack

- Go
- standard library or lightweight HTTP routing
- server-rendered/embedded frontend for MVP
- SQLite with WAL, foreign keys and bounded busy timeout
- internal REST API with one shared Problem Details error contract

Avoid adding PostgreSQL, Redis, a separate Node.js backend, Kafka, Kubernetes or microservices for the MVP unless a measured requirement appears.

## Persistence

SQLite is the Control Plane source of truth for administrative/domain state. Traffic accounting must never write per packet. Usage is aggregated and flushed in bounded batches/transactions.

Sensitive mutations use transactions. Migrations are versioned, additive when possible and restart-safe.

## telemt integration

Treat telemt as an external dependency. Do not copy upstream implementation into this repository.

The integration adapter owns:

- authenticated Control API calls;
- health/readiness checks;
- user create/update/enable/disable/secret rotation;
- quota/expiry synchronization where needed;
- per-user Sponsor AdTag mapping;
- safe error normalization into the Teleproxy error contract.

The telemt management API must be reachable only on an internal network/loopback-equivalent boundary, with a narrow source allowlist where supported and a non-empty Authorization value. It must not be published as a public host port.

## Deployment boundary

Docker Compose is the MVP deployment unit. Persistent data and configuration live outside ephemeral containers.

The installer owns first-install host preparation and persists installation state. Panel port selection is a host-level concern and therefore is performed by the installer/manager, not by arbitrary Web requests.

## Panel port invariant

On first install:

1. choose a random port from a configured high-port range;
2. verify it is not already listening/bound and does not collide with configured published ports;
3. persist the chosen port before service activation;
4. start the stack;
5. healthcheck the exact published port;
6. on failure, retain enough state for a safe retry/rollback.

A normal rerun preserves the existing selected port. Changing the port later is an explicit operation with conflict check, backup, healthcheck and rollback.

## Future node model

The schema and service boundaries should remain ready for multiple Proxy Nodes and Relay Nodes, but the first implementation targets one local Control Plane plus one local/managed telemt node. Do not build distributed consensus or multi-control-plane behavior in the MVP.
