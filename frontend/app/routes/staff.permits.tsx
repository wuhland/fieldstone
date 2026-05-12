import { Link, useLoaderData } from "react-router";
import type { LoaderFunctionArgs, MetaFunction } from "react-router";

export const meta: MetaFunction = () => [
  { title: "Permits | Fieldstone Staff" },
];

interface Permit {
  id: string;
  permit_type: string;
  property_address: string;
  status: string;
  submitted_at: string;
}

export async function loader({ request }: LoaderFunctionArgs) {
  const apiUrl = process.env.API_URL ?? "http://localhost:8080";
  try {
    const res = await fetch(`${apiUrl}/v1/permits?limit=50`, {
      headers: {
        Authorization: request.headers.get("Authorization") ?? "",
      },
    });
    if (!res.ok) return { permits: [], total: 0 };
    const data = await res.json();
    return { permits: (data.permits ?? []) as Permit[], total: data.total ?? 0 };
  } catch {
    return { permits: [], total: 0 };
  }
}

export default function StaffPermitsPage() {
  const { permits, total } = useLoaderData<typeof loader>();

  return (
    <div>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Permits</h1>
        <span className="text-sm text-gray-500">{total} total</span>
      </div>

      {permits.length === 0 ? (
        <p className="mt-6 text-sm text-gray-500">No permits yet.</p>
      ) : (
        <div className="mt-6 overflow-hidden rounded-lg border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                {["Type", "Address", "Status", "Submitted"].map((h) => (
                  <th
                    key={h}
                    className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {permits.map((p) => (
                <tr key={p.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm font-medium text-gray-900">
                    {p.permit_type}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">{p.property_address}</td>
                  <td className="px-4 py-3 text-sm text-gray-500">{p.status}</td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    <Link
                      to={`/permits/${p.id}`}
                      className="hover:underline"
                    >
                      {new Date(p.submitted_at).toLocaleDateString()}
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
