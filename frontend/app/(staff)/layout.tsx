"use client";

import { useEffect, useState } from "react";

export default function StaffLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [authorized, setAuthorized] = useState(false);

  useEffect(() => {
    // TODO(fieldstone): implement OIDC auth check
    // Check for valid JWT in session storage; redirect to OIDC provider if missing
    const token = sessionStorage.getItem("fieldstone_token");
    if (!token) {
      // In production, redirect to OIDC provider
      // window.location.href = "/auth/login";
    }
    setAuthorized(true); // stub: always authorized in dev
  }, []);

  if (!authorized) {
    return null;
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="border-b border-gray-200 bg-white px-4 py-3">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <span className="text-lg font-semibold text-gray-900">
            Fieldstone Staff Portal
          </span>
          <div className="flex gap-4 text-sm">
            <a href="/dashboard" className="text-gray-600 hover:text-gray-900">Dashboard</a>
            <a href="/requests" className="text-gray-600 hover:text-gray-900">Requests</a>
            <a href="/permits" className="text-gray-600 hover:text-gray-900">Permits</a>
          </div>
        </div>
      </nav>
      <main className="mx-auto max-w-7xl px-4 py-8">{children}</main>
    </div>
  );
}
