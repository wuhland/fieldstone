"use client";

import { useState } from "react";
import DynamicMetadataForm from "@/components/forms/DynamicMetadataForm";

export default function NewRequestPage() {
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setSubmitting(true);
    const form = new FormData(e.currentTarget);
    await fetch("/v1/requests", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        request_type: form.get("request_type"),
        description: form.get("description"),
        submitter_email: form.get("submitter_email"),
        location: {},
        metadata: {},
      }),
    });
    setSubmitting(false);
    setSubmitted(true);
  }

  if (submitted) {
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

      <form onSubmit={handleSubmit} className="mt-8 space-y-6">
        <div>
          <label className="block text-sm font-medium text-gray-700">
            Request type
          </label>
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
          <label className="block text-sm font-medium text-gray-700">
            Description
          </label>
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

        {/* Custom fields defined by the city for service_request metadata */}
        <DynamicMetadataForm resourceType="service_request" />

        <button
          type="submit"
          disabled={submitting}
          className="w-full rounded-md bg-blue-600 px-4 py-2 text-white font-medium hover:bg-blue-700 disabled:opacity-50"
        >
          {submitting ? "Submitting…" : "Submit request"}
        </button>
      </form>
    </main>
  );
}
