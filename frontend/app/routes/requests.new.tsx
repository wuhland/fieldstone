import { useState } from "react";
import { Form, useActionData, useNavigation } from "react-router";
import type { ActionFunctionArgs, MetaFunction } from "react-router";
import DynamicMetadataForm from "@/components/forms/DynamicMetadataForm";

export const meta: MetaFunction = () => [
  { title: "Submit a Request | Fieldstone" },
];

export async function action({ request }: ActionFunctionArgs) {
  const formData = await request.formData();

  let metadata: Record<string, unknown> = {};
  try {
    const raw = formData.get("metadata") as string;
    if (raw) metadata = JSON.parse(raw);
  } catch {}

  const body = {
    request_type: formData.get("request_type"),
    description: formData.get("description"),
    submitter_email: formData.get("submitter_email") || undefined,
    location: {},
    metadata,
  };

  const apiUrl = process.env.API_URL ?? "http://localhost:8080";
  try {
    const res = await fetch(`${apiUrl}/v1/requests`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      return { error: (err as { error?: string }).error ?? "Submission failed. Please try again." };
    }
  } catch {
    return { error: "Could not reach the server. Please try again." };
  }

  return { success: true };
}

export default function NewRequestPage() {
  const actionData = useActionData<typeof action>();
  const navigation = useNavigation();
  const submitting = navigation.state === "submitting";
  const [metadata, setMetadata] = useState<Record<string, unknown>>({});

  if (actionData?.success) {
    return (
      <main className="mx-auto max-w-2xl px-4 py-16">
        <div className="rounded-lg border border-green-200 bg-green-50 p-6">
          <h1 className="text-xl font-semibold text-green-900">Request submitted</h1>
          <p className="mt-2 text-green-700">
            Your service request has been received. You will receive updates at the
            email address provided.
          </p>
          <a href="/" className="mt-4 inline-block text-sm text-green-800 underline">
            Return to home
          </a>
        </div>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-16">
      <h1 className="text-2xl font-bold text-gray-900">Submit a Service Request</h1>
      <p className="mt-2 text-gray-600">
        Report an issue such as a pothole, broken streetlight, or code violation.
      </p>

      {actionData?.error && (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-4">
          <p className="text-red-700">{actionData.error}</p>
        </div>
      )}

      <Form method="post" className="mt-8 space-y-6">
        {/* Carries serialized custom-field values from DynamicMetadataForm */}
        <input type="hidden" name="metadata" value={JSON.stringify(metadata)} />

        <div>
          <label className="block text-sm font-medium text-gray-700">Request type</label>
          <select
            name="request_type"
            required
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none"
          >
            <option value="">Select a type…</option>
            <option value="pothole">Pothole</option>
            <option value="streetlight">Streetlight outage</option>
            <option value="graffiti">Graffiti</option>
            <option value="illegal_dumping">Illegal dumping</option>
            <option value="other">Other</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">Description</label>
          <textarea
            name="description"
            required
            rows={4}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none"
            placeholder="Describe the issue and its location…"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">
            Your email (optional)
          </label>
          <input
            type="email"
            name="submitter_email"
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none"
            placeholder="you@example.com"
          />
        </div>

        <DynamicMetadataForm resourceType="service_request" onChange={setMetadata} />

        <button
          type="submit"
          disabled={submitting}
          className="w-full rounded-md bg-blue-600 px-4 py-2 text-white font-medium hover:bg-blue-700 disabled:opacity-50"
        >
          {submitting ? "Submitting…" : "Submit request"}
        </button>
      </Form>
    </main>
  );
}
