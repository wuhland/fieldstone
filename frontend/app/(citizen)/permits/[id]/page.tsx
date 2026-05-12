"use client";

import { useEffect, useState } from "react";

interface PermitStatus {
  id: string;
  permit_type: string;
  status: string;
  property_address: string;
  submitted_at: string;
  issued_at?: string;
  expires_at?: string;
}

export default function PermitStatusPage({ params }: { params: { id: string } }) {
  const [permit, setPermit] = useState<PermitStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch(`/v1/permits/${params.id}/status`)
      .then((r) => (r.ok ? r.json() : Promise.reject(r.status)))
      .then(setPermit)
      .catch(() => setError("Permit not found or unavailable."));
  }, [params.id]);

  if (error) {
    return (
      <main className="mx-auto max-w-2xl px-4 py-16">
        <div className="rounded-lg border border-red-200 bg-red-50 p-6">
          <p className="text-red-700">{error}</p>
        </div>
      </main>
    );
  }

  if (!permit) {
    return (
      <main className="mx-auto max-w-2xl px-4 py-16">
        <div className="animate-pulse space-y-4">
          <div className="h-6 w-48 rounded bg-gray-200" />
          <div className="h-4 w-64 rounded bg-gray-200" />
        </div>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-16">
      <h1 className="text-2xl font-bold text-gray-900">Permit Status</h1>

      <dl className="mt-6 space-y-4">
        <Row label="Permit ID" value={permit.id} />
        <Row label="Type" value={permit.permit_type} />
        <Row label="Address" value={permit.property_address} />
        <Row label="Submitted" value={new Date(permit.submitted_at).toLocaleDateString()} />
        {permit.issued_at && (
          <Row label="Issued" value={new Date(permit.issued_at).toLocaleDateString()} />
        )}
        {permit.expires_at && (
          <Row label="Expires" value={new Date(permit.expires_at).toLocaleDateString()} />
        )}
        <div className="flex gap-2 items-center">
          <dt className="w-32 text-sm font-medium text-gray-500">Status</dt>
          <dd>
            <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium bg-blue-100 text-blue-800">
              {permit.status}
            </span>
          </dd>
        </div>
      </dl>
    </main>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex gap-2">
      <dt className="w-32 text-sm font-medium text-gray-500">{label}</dt>
      <dd className="text-sm text-gray-900">{value}</dd>
    </div>
  );
}
