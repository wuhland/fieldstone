# Next.js 15 App Router for the Frontend

## Status

Accepted

## Context and Problem Statement

Fieldstone needs a frontend for two audiences: citizens submitting requests and checking permit status (public, no authentication), and city staff managing records, reviewing permits, and administering the system (authenticated, feature-rich). What frontend technology should be used?

## Decision Drivers

* The frontend must support both a public citizen portal and an authenticated staff portal within the same codebase.
* TypeScript is strongly preferred — the API client and form components benefit from type safety.
* The `DynamicMetadataForm` component renders city-registered JSON Schemas as form fields; this is a non-trivial component that benefits from a mature React ecosystem.
* The system should be deployable as a standalone container alongside the Go services.
* The frontend must work with the OpenAPI-described Go API without a language mismatch.

## Considered Options

* **Next.js 15 with App Router** — React meta-framework with server components, file-based routing, TypeScript-first, standalone output mode.
* **React + Vite (SPA)** — client-side-only SPA with Vite bundler; no server-side rendering.
* **Go HTML templates** — server-rendered HTML from the Go services using `html/template`.
* **HTMX + Go templates** — hypermedia-driven frontend using Go-rendered HTML with HTMX for interactivity.

## Decision Outcome

Chosen option: **Next.js 15 with App Router**, because the route group feature (`(citizen)/` and `(staff)/`) maps cleanly to the two user audiences, the App Router's layout system handles the staff auth guard elegantly, and the React ecosystem provides the mature component primitives needed for `DynamicMetadataForm`.

The frontend is deployed as a standalone Next.js container (using `output: "standalone"`) alongside the Go services. Caddy routes `/v1/*` to the gateway and everything else to the Next.js container. TypeScript strict mode is enabled throughout.

### Positive Consequences

* App Router route groups (`(citizen)/`, `(staff)/`) provide clean URL structure without leaking the routing convention into URLs.
* The layout system makes the staff auth guard a single `layout.tsx` rather than a per-page check.
* `DynamicMetadataForm` is straightforward to implement with React hooks (`useEffect` for schema fetch, conditional rendering based on schema property types).
* Next.js standalone output produces a self-contained Docker image with no external runtime dependencies.
* The large React/TypeScript ecosystem provides accessible UI components, form validation, and data fetching patterns.
* The typed API client in `lib/api.ts` provides compile-time assurance that frontend calls match the expected contract.

### Negative Consequences

* Next.js is a more complex dependency than a pure SPA — it includes a Node.js runtime, which adds a container to the deployment stack.
* The App Router is a relatively new paradigm (introduced in Next.js 13); some React patterns from the Pages Router era do not apply directly.
* Server components and the client/server boundary require care — components that use browser APIs (sessionStorage for the auth stub) must be marked `"use client"`.
* JavaScript/TypeScript builds (tsc, eslint, next build) add to CI time and require Node.js tooling.

## Pros and Cons of the Options

### React + Vite (SPA)

* Good, because it is a simpler deployment — static files served by Caddy, no Node.js runtime needed.
* Good, because the build output is just HTML/CSS/JS.
* Bad, because there is no server-side rendering — initial page load is a blank HTML shell until JavaScript executes, which is poor for the public citizen portal.
* Bad, because routing and layout composition require additional libraries (React Router) that Next.js provides out of the box.

### Go HTML templates

* Good, because it eliminates the JavaScript build pipeline entirely.
* Good, because templates run in the same process as the Go API — no cross-origin concerns in development.
* Bad, because `DynamicMetadataForm` — which renders arbitrary JSON Schemas as interactive forms — would require significant custom JavaScript without a component framework.
* Bad, because the Go template ecosystem lacks the UI component ecosystem available to React.
* Bad, because staff portal interactivity (sorting, filtering, inline editing) is difficult to implement well with server-rendered templates.

### HTMX + Go templates

* Good, because it dramatically reduces JavaScript complexity while keeping interactivity.
* Good, because the Go server renders both the API responses and the UI, eliminating the mismatch concern.
* Bad, because `DynamicMetadataForm` still requires custom JavaScript to render schema-driven form fields dynamically.
* Bad, because HTMX is less familiar than React to most frontend contributors.
* Neutral, because HTMX is a genuinely strong choice for a mostly-server-rendered application and would be worth reconsidering if the React complexity proves burdensome.
