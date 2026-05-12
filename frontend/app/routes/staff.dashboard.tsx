import type { MetaFunction } from "react-router";

export const meta: MetaFunction = () => [
  { title: "Dashboard | Fieldstone Staff" },
];

export default function DashboardPage() {
  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
      <p className="mt-2 text-gray-500">
        {/* TODO(fieldstone): summary stats from GET /v1/permits and /v1/requests */}
        Welcome to the staff portal.
      </p>
    </div>
  );
}
