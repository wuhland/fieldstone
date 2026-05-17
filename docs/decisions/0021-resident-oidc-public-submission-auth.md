# Resident Identity via Configurable OIDC for Public Submissions

## Status

Accepted

## Context and Problem Statement

The three public submission endpoints (`POST /v1/requests`, `POST /v1/records/foia`,
`POST /v1/permits`) previously required no authentication. This created three compounding
problems:

1. **DoS/spam surface** — no barrier prevents flooding the system with garbage submissions.
   IP-based rate limiting is the only protection, which is trivially bypassed by distributed
   attacks.
2. **Unverifiable contact information** — submitter email is self-reported and unverified.
   City staff cannot reliably follow up. For FOIA requests this is a legal compliance risk:
   if the requester's contact is fake or mistyped, the city cannot fulfill its statutory
   response obligation.
3. **No resident-side tracking** — submitters have no way to check status on their own
   submissions without a separately issued tracking number.

The core question: how should the system establish verified identity for public submissions
without creating prohibitive friction for residents?

## Decision Drivers

* Residents should not need to maintain a separate Fieldstone account — cities already have
  civic identity infrastructure or can use established government identity providers.
* Identity implementation must not be Fieldstone's responsibility: the system should validate
  tokens, not issue them. Auth complexity (credential storage, account recovery, MFA) belongs
  to a purpose-built identity provider.
* The solution must not introduce commercial platform dependencies. A resident's interaction
  with city government services should not be visible to Google, Apple, or Meta.
* The architecture should be consistent: staff identity already uses OIDC with a configurable
  provider. Resident identity should follow the same pattern.
* Self-hosted deployments (cities with no external connectivity or strong data sovereignty
  requirements) must have a workable path.

## Considered Options

* **Require no auth / keep public** — status quo; no friction but retains all three problems.
* **Magic-link / passwordless email (self-implemented)** — Fieldstone issues its own tokens
  after email verification; requires SMTP infrastructure and custom JWT issuance.
* **Delegate to a commercial OAuth provider (Google, Apple)** — familiar UX but introduces
  commercial platform dependency; not all residents have accounts; city IT must register OAuth
  apps.
* **Require OIDC, provider configurable** — same pattern as staff auth; cities configure a
  resident OIDC issuer URL; Fieldstone validates tokens without issuing them.

## Decision Outcome

Chosen option: **require OIDC authentication for public submissions, with the resident
identity provider configurable via `RESIDENT_OIDC_ISSUER_URL`**.

Fieldstone remains an OIDC relying party only — it never issues tokens. The gateway
initialises a second JWKS cache at startup (one per issuer) and uses the `iss` claim to
route incoming tokens to the correct validation path. Resident tokens receive the synthetic
role `"resident"` regardless of the token's own claims, so no role management is needed
in the identity provider.

For US municipalities, the recommended provider is **Login.gov**: GSA-operated, explicit
data minimisation policy, no commercial data sharing, and a growing base of residents who
already have accounts from federal service interactions. Internationally, equivalent
government identity providers (GOV.UK One Login, etc.) or self-hosted providers
(Keycloak, Authentik) work identically.

When `RESIDENT_OIDC_ISSUER_URL` is not configured, the resident-facing routes require a
staff JWT — this preserves backward compatibility while making the security intent
explicit: unauthenticated submission is a conscious configuration choice, not a default.

### Gateway changes

The gateway strips all `X-Fieldstone-*` headers from incoming client requests (global
middleware), then re-injects `X-Fieldstone-Sub`, `X-Fieldstone-Role`, and
`X-Fieldstone-Email` after successful JWT validation. Domain services trust these headers
as the authoritative caller identity without re-parsing the JWT. This centralises token
validation at the gateway and avoids distributing OIDC configuration across every service.

### Domain service changes

`resident_id TEXT` is added to the `service_requests`, `foia_requests`, and `permits`
tables. On submission the column is populated from `X-Fieldstone-Sub`. On `GET /{id}`,
if the caller's role is `"resident"`, the handler returns 403 unless `resident_id` matches
the caller's sub. Staff reads are unchanged.

### Positive Consequences

* Verified contact identity on every submission — city staff always have a real, reachable
  identity to follow up with.
* FOIA legal compliance: the requester of record is verifiable.
* Residents can retrieve their own submission history using their OIDC token.
* DoS/spam barrier: account creation with a government identity provider is a meaningful
  hurdle compared to a public HTTP endpoint.
* No new auth complexity in Fieldstone: JWK cache, token parsing, and expiry validation
  are already implemented; the second issuer reuses the same code path.
* Privacy: Login.gov's data minimisation policy means the city is not routing resident
  interactions through a commercial advertising platform.

### Negative Consequences

* Cities must configure a resident OIDC provider before residents can self-submit.
  Until then, staff must submit on behalf of residents.
* Login.gov requires a partnership agreement (IAA). The process takes weeks; smaller
  cities may lack the procurement capacity.
* Resident OIDC startup requires a reachable issuer — same operational constraint that
  staff OIDC already imposes.
* Older or less digitally literate residents may face more friction than an anonymous
  submission form. Cities should consider accessibility accommodations (staff-assisted
  submission kiosks, phone intake) as a complement.

## Pros and Cons of the Options

### Keep public (no auth)

* Good, because zero friction for residents.
* Bad, because garbage submissions are indistinguishable from real ones.
* Bad, because FOIA response obligations cannot be reliably met with unverified contact info.
* Bad, because there is no path to resident-side status tracking without a separate tracking
  number mechanism.

### Magic-link / passwordless email (self-implemented)

* Good, because no external dependency at runtime — city controls the entire flow.
* Good, because low friction (no account required, email is the identity).
* Bad, because Fieldstone must implement and operate SMTP, token generation, token storage,
  and expiry — each a potential security liability and operational burden.
* Bad, because email deliverability becomes a dependency: misconfigured SMTP means residents
  cannot authenticate.
* Bad, because it duplicates work that purpose-built identity providers do better.

### Delegate to commercial OAuth (Google, Apple)

* Good, because familiar UX and low friction for residents with existing accounts.
* Good, because the provider handles MFA, account recovery, and credential security.
* Bad, because a resident's interaction with city services is visible to a commercial
  advertising platform — a meaningful privacy concern for municipal government.
* Bad, because residents without Google/Apple accounts are excluded or must create one.
* Bad, because city IT must register and maintain OAuth applications with each provider,
  adding operational overhead.
