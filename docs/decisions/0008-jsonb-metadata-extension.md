# JSONB Metadata Columns with JSON Schema Validation for Custom Fields

## Status

Accepted

## Context and Problem Statement

Different cities need different fields on the same resources. City A might require a contractor license number on permits; City B might need a historic district flag; City C might need a neighborhood classification. How should Fieldstone support city-specific custom fields without requiring schema migrations or code changes?

## Decision Drivers

* Cities must be able to add custom fields without forking, modifying, or redeploying Fieldstone.
* Custom fields must be validated — a city that requires `contractor_license` should not accept a permit without it.
* The solution must work across all major entities (permits, service requests, FOIA requests) with a consistent API.
* The implementation must not require per-city schema migrations that could be run against a shared database.

## Considered Options

* **JSONB `metadata` column with JSON Schema validation** — a single `metadata jsonb` column per entity; a `field_schemas` table stores JSON Schema per resource type; validation runs on every write.
* **Entity-Attribute-Value (EAV) tables** — a `custom_fields` table with `(entity_id, field_name, field_value)` rows.
* **Per-city schema migrations** — cities run additional migrations to add typed columns to entity tables.
* **Separate custom tables per city** — each city gets its own extension table joined to the core entity.

## Decision Outcome

Chosen option: **JSONB `metadata` column with JSON Schema validation**, because it requires no schema migration for each new field, the validation contract is explicit and machine-readable (JSON Schema draft-07), and JSONB is natively indexed and queryable in PostgreSQL when needed.

The `identity.field_schemas` table stores one JSON Schema per `resource_type`. On every `POST`/`PATCH` to a resource, the handler fetches the registered schema (cached in-process for 60 seconds using `internal/validate`) and validates the request's `metadata` field, returning 422 with field-level errors on failure. Cities manage schemas via `GET/PUT /v1/config/schemas/:resource_type`.

### Positive Consequences

* Adding a custom field is a single API call (`PUT /v1/config/schemas/permit`) — no migrations, no restarts.
* JSON Schema is a well-understood, language-neutral standard; city developers can write and validate schemas with standard tooling.
* The `DynamicMetadataForm` frontend component reads the schema and renders appropriate form fields automatically.
* JSONB values are queryable with PostgreSQL operators (`metadata->>'field'`, `@>`) when reporting needs arise.
* One implementation covers all resource types uniformly.

### Negative Consequences

* JSONB columns cannot be indexed as efficiently as typed columns for range queries; complex reporting on custom fields may require GIN indexes or materialized views.
* JSON Schema validation is a runtime check, not a compile-time guarantee — schema bugs appear as 422 errors in production.
* Cities that need referential integrity (e.g., `contractor_license` must exist in a license table) cannot express this in JSON Schema alone.
* The 60-second schema cache means a schema update takes up to a minute to propagate to all service instances.

## Pros and Cons of the Options

### Entity-Attribute-Value (EAV) tables

* Good, because it is relational and typed fields could have foreign keys.
* Bad, because queries involving custom fields require complex pivoting (one row per field value → one column per field).
* Bad, because reporting is extremely difficult; analytics tools that expect tabular data struggle with EAV.
* Bad, because there is no natural place to define the schema/validation rules for a field.

### Per-city schema migrations

* Good, because custom fields are typed columns with all the benefits of PostgreSQL types and indexes.
* Good, because the ORM or query layer can express custom fields as normal struct fields.
* Bad, because each new field requires a migration — a coordination step between city admin and Fieldstone operator.
* Bad, because migrations cannot easily be run in a shared-schema environment (ADR-0003).
* Bad, because it completely breaks the "no forking" extensibility model.
