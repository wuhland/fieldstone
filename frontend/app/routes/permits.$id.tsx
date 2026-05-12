import { useLoaderData } from "react-router";
import type { LoaderFunctionArgs, MetaFunction } from "react-router";

interface Permit {
  id: string;
  permit_type: string;
  status: string;
  property_address: string;
  submitted_at: string;
  issued_at?: string;
  expires_at?: string;
}

export async function loader({ params }: LoaderFunctionArgs) {
  const apiUrl = process.env.API_URL ?? "http://localhost:8080";
  const res = await fetch(`${apiUrl}/v1/permits/${params.id}/status`);
  if (res.status === 404) {
    throw new Response("Permit not found", { status: 404 });
  }
  if (!res.ok) {
    throw new Response("Failed to load permit", { status: 502 });
  }
  const permit: Permit = await res.json();
  return { permit };
}

export const meta: MetaFunction<typeof loader> = ({ data }) => [
  { title: `Permit ${data?.permit.id.slice(0, 8) ?? "…"} | Fieldstone` },
];

export default function PermitStatusPage() {
  const { permit } = useLoaderData<typeof loader>();

  return (
    <main className="mx-auto max-w-2xl px-4 py-16">
      <h1 className="text-2xl font-bold text-gray-900">Permit Status</h1>

      <dl className="mt-6 divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white">
        <Row label="Permit ID" value={permit.id} mono />
        <Row label="Type" value={permit.permit_type} />
        <Row label="Address" value={permit.property_address} />
        <Row label="Submitted" value={new Date(permit.submitted_at).toLocaleDateString()} />
        {permit.issued_at && (
          <Row label="Issued" value={new Date(permit.issued_at).toLocaleDateString()} />
        )}
        {permit.expires_at && (
          <Row label="Expires" value={new Date(permit.expires_at).toLocaleDateString()} />
        )}
        <div className="flex items-center gap-4 px-4 py-3">
          <dt className="w-28 shrink-0 text-sm font-medium text-gray-500">Status</dt>
          <dd><StatusBadge status={permit.status} /></dd>
        </div>
      </dl>
    </main>
  );
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center gap-4 px-4 py-3">
      <dt className="w-28 shrink-0 text-sm font-medium text-gray-500">{label}</dt>
      <dd className={`text-sm text-gray-900 ${mono ? "font-mono text-xs" : ""}`}>{value}</dd>
    </div>
  );
}

const statusColors: Record<string, string> = {
  submitted:    "bg-yellow-100 text-yellow-800",
  under_review: "bg-blue-100 text-blue-800",
  approved:     "bg-green-100 text-green-800",
  rejected:     "bg-red-100 text-red-800",
  expired:      "bg-gray-100 text-gray-700",
};

function StatusBadge({ status }: { status: string }) {
  const cls = statusColors[status] ?? "bg-gray-100 text-gray-700";
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${cls}`}>
      {status.replace(/_/g, " ")}
    </span>
  );
}
