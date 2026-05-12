export default function HomePage() {
  return (
    <main className="min-h-screen bg-gray-50">
      <div className="mx-auto max-w-4xl px-4 py-16">
        <h1 className="text-3xl font-bold text-gray-900">
          City Services Portal
        </h1>
        <p className="mt-4 text-gray-600">
          Submit requests, check permit status, and access public records.
        </p>

        <div className="mt-10 grid grid-cols-1 gap-6 sm:grid-cols-3">
          <ServiceCard
            title="311 Requests"
            description="Report potholes, broken streetlights, and other issues."
            href="/requests/new"
          />
          <ServiceCard
            title="Permits"
            description="Check the status of a building or business permit."
            href="/permits/lookup"
          />
          <ServiceCard
            title="Public Records"
            description="Submit a Freedom of Information Act request."
            href="/records/foia/new"
          />
        </div>
      </div>
    </main>
  );
}

function ServiceCard({
  title,
  description,
  href,
}: {
  title: string;
  description: string;
  href: string;
}) {
  return (
    <a
      href={href}
      className="block rounded-lg border border-gray-200 bg-white p-6 shadow-sm hover:border-blue-500 hover:shadow-md transition-all"
    >
      <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
      <p className="mt-2 text-sm text-gray-600">{description}</p>
    </a>
  );
}
