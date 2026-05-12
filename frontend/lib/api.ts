// Typed API client for the Fieldstone gateway.
// Generated against the OpenAPI spec at api/openapi/gateway.yaml.
// TODO(fieldstone): replace manual types with generated client once spec is stable.

const BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? `HTTP ${res.status}`);
  }
  return res.json();
}

// ─── Service Requests ────────────────────────────────────────────────────────

export interface ServiceRequest {
  id: string;
  department_id: string;
  request_type: string;
  status: string;
  description: string;
  location: Record<string, unknown>;
  submitter_email?: string;
  assigned_to?: string;
  metadata: Record<string, unknown>;
  closed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateServiceRequestInput {
  request_type: string;
  description: string;
  submitter_email?: string;
  location?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
}

export const requests = {
  list: (token: string) =>
    request<ServiceRequest[]>("/v1/requests", {
      headers: { Authorization: `Bearer ${token}` },
    }),
  get: (id: string) => request<ServiceRequest>(`/v1/requests/${id}`),
  create: (input: CreateServiceRequestInput) =>
    request<ServiceRequest>("/v1/requests", {
      method: "POST",
      body: JSON.stringify(input),
    }),
};

// ─── Permits ─────────────────────────────────────────────────────────────────

export interface Permit {
  id: string;
  department_id: string;
  permit_type: string;
  status: string;
  applicant: Record<string, unknown>;
  property_address: string;
  metadata: Record<string, unknown>;
  submitted_at: string;
  issued_at?: string;
  expires_at?: string;
}

export const permits = {
  list: (token: string) =>
    request<Permit[]>("/v1/permits", {
      headers: { Authorization: `Bearer ${token}` },
    }),
  get: (id: string) => request<Permit>(`/v1/permits/${id}`),
  getStatus: (id: string) => request<Permit>(`/v1/permits/${id}/status`),
};

// ─── Schemas ──────────────────────────────────────────────────────────────────

export interface FieldSchema {
  id: string;
  resource_type: string;
  schema: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export const schemas = {
  get: (resourceType: string, token?: string) =>
    request<FieldSchema>(`/v1/config/schemas/${resourceType}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    }),
  put: (resourceType: string, schema: Record<string, unknown>, token: string) =>
    request<FieldSchema>(`/v1/config/schemas/${resourceType}`, {
      method: "PUT",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ schema }),
    }),
};

// ─── Webhooks ─────────────────────────────────────────────────────────────────

export interface Webhook {
  id: string;
  url: string;
  events: string[];
  description?: string;
  enabled: boolean;
  fail_count: number;
  created_at: string;
}

export const webhooks = {
  list: (token: string) =>
    request<Webhook[]>("/v1/webhooks", {
      headers: { Authorization: `Bearer ${token}` },
    }),
  create: (input: Omit<Webhook, "id" | "fail_count" | "created_at" | "enabled"> & { secret: string }, token: string) =>
    request<Webhook>("/v1/webhooks", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify(input),
    }),
  delete: (id: string, token: string) =>
    request<void>(`/v1/webhooks/${id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
    }),
};
