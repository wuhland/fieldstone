import { Link, Outlet, useLocation } from "react-router";

const navLinks = [
  { to: "/staff", label: "Dashboard" },
  { to: "/staff/requests", label: "Requests" },
  { to: "/staff/permits", label: "Permits" },
];

export default function StaffLayout() {
  // TODO(fieldstone): validate JWT from cookie/session and redirect to OIDC
  // provider if missing. DEV_DISABLE_AUTH=true on the gateway lets all
  // requests through without a token during local development.
  const { pathname } = useLocation();

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
          <Link to="/" className="text-lg font-semibold text-gray-900">
            Fieldstone Staff
          </Link>
          <div className="flex gap-1">
            {navLinks.map(({ to, label }) => (
              <Link
                key={to}
                to={to}
                className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                  pathname === to
                    ? "bg-gray-100 text-gray-900"
                    : "text-gray-600 hover:text-gray-900 hover:bg-gray-50"
                }`}
              >
                {label}
              </Link>
            ))}
          </div>
        </div>
      </nav>
      <main className="mx-auto max-w-7xl px-4 py-8">
        <Outlet />
      </main>
    </div>
  );
}
