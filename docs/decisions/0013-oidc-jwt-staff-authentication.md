# OIDC with JWT Validation for Staff Authentication

## Status

Accepted

## Context and Problem Statement

Staff users (city employees and contractors) need to authenticate to access the Fieldstone management interface and staff API endpoints. Fieldstone must not be its own identity provider — cities already have identity systems (Active Directory, Azure Entra ID, Google Workspace, Okta) that manage staff accounts, MFA policies, and offboarding. How should Fieldstone authenticate staff?

## Decision Drivers

* Cities must be able to use their existing identity provider — Fieldstone must not require staff to maintain a separate set of credentials.
* MFA, password policies, and account deprovisioning must be managed by the city's existing IdP, not by Fieldstone.
* The gateway must be able to validate tokens without a round-trip to the identity provider on every request.
* The implementation must be auditable: the spec requires checking expiry, issuer, audience, and signature — no shortcuts.

## Considered Options

* **OIDC with JWT validation at the gateway** — staff authenticate via the city's OIDC provider; the gateway validates JWT Bearer tokens against the provider's JWKS.
* **Session-based authentication with Fieldstone's own user database** — staff register and log in to Fieldstone directly; sessions stored server-side.
* **API key authentication** — staff are issued long-lived API keys stored in the identity service.
* **mTLS (mutual TLS)** — clients present certificates for authentication.

## Decision Outcome

Chosen option: **OIDC with JWT validation at the gateway**, because it delegates all credential management, MFA, and account lifecycle to the city's existing IdP, and stateless JWT validation at the gateway scales without requiring a session store.

The gateway fetches the OIDC discovery document at startup to locate the JWKS endpoint. JWKS keys are cached in-process with a 5-minute TTL. Every token is validated for: signature (against the JWKS), expiry (`exp` claim), issuer (`iss` must match `OIDC_ISSUER_URL`), and audience (`aud` must match `OIDC_AUDIENCE`). The `Claims` struct is attached to the request context for downstream use.

### Positive Consequences

* Staff use their existing city credentials (SSO) — no new password to manage.
* MFA enforcement, password rotation, and account deprovisioning are handled by the city's IdP — Fieldstone is not in that critical path.
* JWT validation is stateless — the gateway can validate tokens without a database or cache lookup on every request.
* Any OIDC-compliant provider works: Azure Entra ID, Google Workspace, Okta, Keycloak, Authentik.
* The identity service stores `oidc_sub` (the stable user identifier from the JWT), not passwords or session tokens.

### Negative Consequences

* The gateway cannot start in production without a reachable OIDC provider (JWKS fetch at startup).
* Local development requires either a real OIDC provider or a dev stub (e.g., Dex, Keycloak in Docker).
* Revoking a JWT before its expiry requires either short-lived tokens (minutes) or a token revocation list — OIDC alone does not provide instant revocation.
* The audience and issuer values must be correctly configured per environment; misconfiguration silently accepts or rejects all tokens.

## Pros and Cons of the Options

### Session-based authentication with Fieldstone's own user database

* Good, because sessions can be invalidated instantly on logout or account disable.
* Good, because it does not require the city to configure an OIDC application.
* Bad, because Fieldstone would need to implement password hashing, reset flows, MFA, and account lockout — each a potential security liability.
* Bad, because staff must maintain a separate set of credentials outside the city's identity governance process.
* Bad, because a session store (Redis or DB) becomes a required dependency and a potential bottleneck.

### API key authentication

* Good, because it is simple to implement and does not require an external dependency.
* Bad, because API keys are long-lived credentials — a leaked key grants access until it is manually revoked.
* Bad, because key issuance and rotation are operationally burdensome at scale.
* Bad, because there is no standard way to associate API keys with identity governance (MFA, offboarding).
