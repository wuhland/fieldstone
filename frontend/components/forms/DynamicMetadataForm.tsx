"use client";

import { useEffect, useState } from "react";

interface JSONSchema {
  type?: string;
  properties?: Record<string, JSONSchemaProperty>;
  required?: string[];
}

interface JSONSchemaProperty {
  type?: string;
  title?: string;
  description?: string;
  enum?: string[];
  minimum?: number;
  maximum?: number;
}

interface Props {
  resourceType: string;
  value?: Record<string, unknown>;
  onChange?: (data: Record<string, unknown>) => void;
}

/**
 * DynamicMetadataForm renders form fields from a city-registered JSON Schema.
 * Fetches the schema for a resource type and renders appropriate inputs.
 * Falls back gracefully (renders nothing) if no schema is registered.
 */
export default function DynamicMetadataForm({ resourceType, value = {}, onChange }: Props) {
  const [schema, setSchema] = useState<JSONSchema | null>(null);
  const [formData, setFormData] = useState<Record<string, unknown>>(value);

  useEffect(() => {
    fetch(`/v1/config/schemas/${resourceType}`)
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (data?.schema) setSchema(data.schema);
      })
      .catch(() => null);
  }, [resourceType]);

  if (!schema?.properties || Object.keys(schema.properties).length === 0) {
    return null;
  }

  function handleChange(key: string, val: unknown) {
    const next = { ...formData, [key]: val };
    setFormData(next);
    onChange?.(next);
  }

  return (
    <fieldset className="space-y-4 border-t border-gray-200 pt-4">
      <legend className="text-sm font-medium text-gray-700">
        Additional information
      </legend>

      {Object.entries(schema.properties).map(([key, prop]) => (
        <Field
          key={key}
          fieldKey={key}
          prop={prop}
          required={schema.required?.includes(key) ?? false}
          value={formData[key]}
          onChange={(v) => handleChange(key, v)}
        />
      ))}
    </fieldset>
  );
}

function Field({
  fieldKey,
  prop,
  required,
  value,
  onChange,
}: {
  fieldKey: string;
  prop: JSONSchemaProperty;
  required: boolean;
  value: unknown;
  onChange: (v: unknown) => void;
}) {
  const label = prop.title ?? fieldKey.replace(/_/g, " ");
  const inputClass =
    "mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none text-sm";

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700">
        {label}
        {required && <span className="ml-1 text-red-500">*</span>}
      </label>
      {prop.description && (
        <p className="mt-0.5 text-xs text-gray-500">{prop.description}</p>
      )}

      {prop.enum ? (
        <select
          required={required}
          value={(value as string) ?? ""}
          onChange={(e) => onChange(e.target.value)}
          className={inputClass}
        >
          <option value="">Select…</option>
          {prop.enum.map((opt) => (
            <option key={opt} value={opt}>
              {opt}
            </option>
          ))}
        </select>
      ) : prop.type === "boolean" ? (
        <input
          type="checkbox"
          required={required}
          checked={(value as boolean) ?? false}
          onChange={(e) => onChange(e.target.checked)}
          className="mt-2 h-4 w-4 rounded border-gray-300 text-blue-600"
        />
      ) : prop.type === "integer" || prop.type === "number" ? (
        <input
          type="number"
          required={required}
          min={prop.minimum}
          max={prop.maximum}
          value={(value as number) ?? ""}
          onChange={(e) => onChange(e.target.valueAsNumber)}
          className={inputClass}
        />
      ) : (
        <input
          type="text"
          required={required}
          value={(value as string) ?? ""}
          onChange={(e) => onChange(e.target.value)}
          className={inputClass}
        />
      )}
    </div>
  );
}
