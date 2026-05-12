import { useLoaderData } from "react-router";
import type { LoaderFunctionArgs, MetaFunction } from "react-router";

export const meta: MetaFunction = () => [
  { title: "Requests | Fieldstone Staff" },
];

interface ServiceRequest {
  id: string;
  request_type: string;
  status: string;
  description: string;
  created_at: string;
}

export async function loader({ request }: LoaderFunctionArgs) {
  const apiUrl = process.env.API_URL ?? "http://localhost:8080";
  try {
    const res = await fetch(`${apiUrl}/v1/requests?limit=50`, {
      headers: {
        Authorization: request.headers.get("Authorization") ?? "",
      },
    });
    if (!res.ok) return { requests: [], total: 0 };
    const data = await res.json();
    return { requests: (data.requests ?? []) as ServiceRequest[], total: data.total ?? 0 };
  } catch {
    return { requests: [], total: 0 };
  }
}

export default function StaffRequestsPage() {
  const { requests, total } = useLoaderData<typeof loader>();

  return (
    <div>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Service Requests</h1>
        <span className="text-sm text-gray-500">{total} total</span>
      </div>

      {requests.length === 0 ? (
        <p className="mt-6 text-sm text-gray-500">No requests yet.</p>
      ) : (
        <div className="mt-6 overflow-hidden rounded-lg border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                {["Type", "Description", "Status", "Submitted"].map((h) => (
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
              {requests.map((req) => (
                <tr key={req.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm font-medium text-gray-900">
                    {req.request_type}
                  </td>
                  <td className="max-w-xs truncate px-4 py-3 text-sm text-gray-600">
                    {req.description}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">{req.status}</td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {new Date(req.created_at).toLocaleDateString()}
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
