import { jsx, jsxs } from "react/jsx-runtime";
import { PassThrough } from "node:stream";
import { createReadableStreamFromReadable } from "@react-router/node";
import { ServerRouter, UNSAFE_withComponentProps, Outlet, Meta, Links, ScrollRestoration, Scripts, useActionData, useNavigation, Form, useLoaderData, useLocation, Link } from "react-router";
import { isbot } from "isbot";
import { renderToPipeableStream } from "react-dom/server";
import { useState, useEffect } from "react";
const streamTimeout = 5e3;
function handleRequest(request, responseStatusCode, responseHeaders, routerContext, loadContext) {
  if (request.method.toUpperCase() === "HEAD") {
    return new Response(null, {
      status: responseStatusCode,
      headers: responseHeaders
    });
  }
  return new Promise((resolve, reject) => {
    let shellRendered = false;
    let userAgent = request.headers.get("user-agent");
    let readyOption = userAgent && isbot(userAgent) || routerContext.isSpaMode ? "onAllReady" : "onShellReady";
    let timeoutId = setTimeout(
      () => abort(),
      streamTimeout + 1e3
    );
    const { pipe, abort } = renderToPipeableStream(
      /* @__PURE__ */ jsx(ServerRouter, { context: routerContext, url: request.url }),
      {
        [readyOption]() {
          shellRendered = true;
          const body = new PassThrough({
            final(callback) {
              clearTimeout(timeoutId);
              timeoutId = void 0;
              callback();
            }
          });
          const stream = createReadableStreamFromReadable(body);
          responseHeaders.set("Content-Type", "text/html");
          pipe(body);
          resolve(
            new Response(stream, {
              headers: responseHeaders,
              status: responseStatusCode
            })
          );
        },
        onShellError(error) {
          reject(error);
        },
        onError(error) {
          responseStatusCode = 500;
          if (shellRendered) {
            console.error(error);
          }
        }
      }
    );
  });
}
const entryServer = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: handleRequest,
  streamTimeout
}, Symbol.toStringTag, { value: "Module" }));
function Layout({
  children
}) {
  return /* @__PURE__ */ jsxs("html", {
    lang: "en",
    children: [/* @__PURE__ */ jsxs("head", {
      children: [/* @__PURE__ */ jsx("meta", {
        charSet: "utf-8"
      }), /* @__PURE__ */ jsx("meta", {
        name: "viewport",
        content: "width=device-width, initial-scale=1"
      }), /* @__PURE__ */ jsx(Meta, {}), /* @__PURE__ */ jsx(Links, {})]
    }), /* @__PURE__ */ jsxs("body", {
      children: [children, /* @__PURE__ */ jsx(ScrollRestoration, {}), /* @__PURE__ */ jsx(Scripts, {})]
    })]
  });
}
const root = UNSAFE_withComponentProps(function App() {
  return /* @__PURE__ */ jsx(Outlet, {});
});
const route0 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  Layout,
  default: root
}, Symbol.toStringTag, { value: "Module" }));
const meta$5 = () => [{
  title: "City Services Portal | Fieldstone"
}, {
  name: "description",
  content: "Submit requests, check permit status, and access public records."
}];
const home = UNSAFE_withComponentProps(function HomePage() {
  return /* @__PURE__ */ jsx("main", {
    className: "min-h-screen bg-gray-50",
    children: /* @__PURE__ */ jsxs("div", {
      className: "mx-auto max-w-4xl px-4 py-16",
      children: [/* @__PURE__ */ jsx("h1", {
        className: "text-3xl font-bold text-gray-900",
        children: "City Services Portal"
      }), /* @__PURE__ */ jsx("p", {
        className: "mt-4 text-gray-600",
        children: "Submit requests, check permit status, and access public records."
      }), /* @__PURE__ */ jsxs("div", {
        className: "mt-10 grid grid-cols-1 gap-6 sm:grid-cols-3",
        children: [/* @__PURE__ */ jsx(ServiceCard, {
          title: "311 Requests",
          description: "Report potholes, broken streetlights, and other issues.",
          href: "/requests/new"
        }), /* @__PURE__ */ jsx(ServiceCard, {
          title: "Permits",
          description: "Check the status of a building or business permit.",
          href: "/permits/lookup"
        }), /* @__PURE__ */ jsx(ServiceCard, {
          title: "Public Records",
          description: "Submit a Freedom of Information Act request.",
          href: "/records/foia/new"
        })]
      })]
    })
  });
});
function ServiceCard({
  title,
  description,
  href
}) {
  return /* @__PURE__ */ jsxs("a", {
    href,
    className: "block rounded-lg border border-gray-200 bg-white p-6 shadow-sm hover:border-blue-500 hover:shadow-md transition-all",
    children: [/* @__PURE__ */ jsx("h2", {
      className: "text-lg font-semibold text-gray-900",
      children: title
    }), /* @__PURE__ */ jsx("p", {
      className: "mt-2 text-sm text-gray-600",
      children: description
    })]
  });
}
const route1 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: home,
  meta: meta$5
}, Symbol.toStringTag, { value: "Module" }));
function DynamicMetadataForm({ resourceType, value = {}, onChange }) {
  const [schema, setSchema] = useState(null);
  const [formData, setFormData] = useState(value);
  useEffect(() => {
    fetch(`/v1/config/schemas/${resourceType}`).then((r) => r.ok ? r.json() : null).then((data) => {
      if (data == null ? void 0 : data.schema) setSchema(data.schema);
    }).catch(() => null);
  }, [resourceType]);
  if (!(schema == null ? void 0 : schema.properties) || Object.keys(schema.properties).length === 0) {
    return null;
  }
  function handleChange(key, val) {
    const next = { ...formData, [key]: val };
    setFormData(next);
    onChange == null ? void 0 : onChange(next);
  }
  return /* @__PURE__ */ jsxs("fieldset", { className: "space-y-4 border-t border-gray-200 pt-4", children: [
    /* @__PURE__ */ jsx("legend", { className: "text-sm font-medium text-gray-700", children: "Additional information" }),
    Object.entries(schema.properties).map(([key, prop]) => {
      var _a;
      return /* @__PURE__ */ jsx(
        Field,
        {
          fieldKey: key,
          prop,
          required: ((_a = schema.required) == null ? void 0 : _a.includes(key)) ?? false,
          value: formData[key],
          onChange: (v) => handleChange(key, v)
        },
        key
      );
    })
  ] });
}
function Field({
  fieldKey,
  prop,
  required,
  value,
  onChange
}) {
  const label = prop.title ?? fieldKey.replace(/_/g, " ");
  const inputClass = "mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none text-sm";
  return /* @__PURE__ */ jsxs("div", { children: [
    /* @__PURE__ */ jsxs("label", { className: "block text-sm font-medium text-gray-700", children: [
      label,
      required && /* @__PURE__ */ jsx("span", { className: "ml-1 text-red-500", children: "*" })
    ] }),
    prop.description && /* @__PURE__ */ jsx("p", { className: "mt-0.5 text-xs text-gray-500", children: prop.description }),
    prop.enum ? /* @__PURE__ */ jsxs(
      "select",
      {
        required,
        value: value ?? "",
        onChange: (e) => onChange(e.target.value),
        className: inputClass,
        children: [
          /* @__PURE__ */ jsx("option", { value: "", children: "Select…" }),
          prop.enum.map((opt) => /* @__PURE__ */ jsx("option", { value: opt, children: opt }, opt))
        ]
      }
    ) : prop.type === "boolean" ? /* @__PURE__ */ jsx(
      "input",
      {
        type: "checkbox",
        required,
        checked: value ?? false,
        onChange: (e) => onChange(e.target.checked),
        className: "mt-2 h-4 w-4 rounded border-gray-300 text-blue-600"
      }
    ) : prop.type === "integer" || prop.type === "number" ? /* @__PURE__ */ jsx(
      "input",
      {
        type: "number",
        required,
        min: prop.minimum,
        max: prop.maximum,
        value: value ?? "",
        onChange: (e) => onChange(e.target.valueAsNumber),
        className: inputClass
      }
    ) : /* @__PURE__ */ jsx(
      "input",
      {
        type: "text",
        required,
        value: value ?? "",
        onChange: (e) => onChange(e.target.value),
        className: inputClass
      }
    )
  ] });
}
const meta$4 = () => [{
  title: "Submit a Request | Fieldstone"
}];
async function action({
  request
}) {
  const formData = await request.formData();
  let metadata = {};
  try {
    const raw = formData.get("metadata");
    if (raw) metadata = JSON.parse(raw);
  } catch {
  }
  const body = {
    request_type: formData.get("request_type"),
    description: formData.get("description"),
    submitter_email: formData.get("submitter_email") || void 0,
    location: {},
    metadata
  };
  const apiUrl = process.env.API_URL ?? "http://localhost:8080";
  try {
    const res = await fetch(`${apiUrl}/v1/requests`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(body)
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      return {
        error: err.error ?? "Submission failed. Please try again."
      };
    }
  } catch {
    return {
      error: "Could not reach the server. Please try again."
    };
  }
  return {
    success: true
  };
}
const requests_new = UNSAFE_withComponentProps(function NewRequestPage() {
  const actionData = useActionData();
  const navigation = useNavigation();
  const submitting = navigation.state === "submitting";
  const [metadata, setMetadata] = useState({});
  if (actionData == null ? void 0 : actionData.success) {
    return /* @__PURE__ */ jsx("main", {
      className: "mx-auto max-w-2xl px-4 py-16",
      children: /* @__PURE__ */ jsxs("div", {
        className: "rounded-lg border border-green-200 bg-green-50 p-6",
        children: [/* @__PURE__ */ jsx("h1", {
          className: "text-xl font-semibold text-green-900",
          children: "Request submitted"
        }), /* @__PURE__ */ jsx("p", {
          className: "mt-2 text-green-700",
          children: "Your service request has been received. You will receive updates at the email address provided."
        }), /* @__PURE__ */ jsx("a", {
          href: "/",
          className: "mt-4 inline-block text-sm text-green-800 underline",
          children: "Return to home"
        })]
      })
    });
  }
  return /* @__PURE__ */ jsxs("main", {
    className: "mx-auto max-w-2xl px-4 py-16",
    children: [/* @__PURE__ */ jsx("h1", {
      className: "text-2xl font-bold text-gray-900",
      children: "Submit a Service Request"
    }), /* @__PURE__ */ jsx("p", {
      className: "mt-2 text-gray-600",
      children: "Report an issue such as a pothole, broken streetlight, or code violation."
    }), (actionData == null ? void 0 : actionData.error) && /* @__PURE__ */ jsx("div", {
      className: "mt-4 rounded-lg border border-red-200 bg-red-50 p-4",
      children: /* @__PURE__ */ jsx("p", {
        className: "text-red-700",
        children: actionData.error
      })
    }), /* @__PURE__ */ jsxs(Form, {
      method: "post",
      className: "mt-8 space-y-6",
      children: [/* @__PURE__ */ jsx("input", {
        type: "hidden",
        name: "metadata",
        value: JSON.stringify(metadata)
      }), /* @__PURE__ */ jsxs("div", {
        children: [/* @__PURE__ */ jsx("label", {
          className: "block text-sm font-medium text-gray-700",
          children: "Request type"
        }), /* @__PURE__ */ jsxs("select", {
          name: "request_type",
          required: true,
          className: "mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none",
          children: [/* @__PURE__ */ jsx("option", {
            value: "",
            children: "Select a type…"
          }), /* @__PURE__ */ jsx("option", {
            value: "pothole",
            children: "Pothole"
          }), /* @__PURE__ */ jsx("option", {
            value: "streetlight",
            children: "Streetlight outage"
          }), /* @__PURE__ */ jsx("option", {
            value: "graffiti",
            children: "Graffiti"
          }), /* @__PURE__ */ jsx("option", {
            value: "illegal_dumping",
            children: "Illegal dumping"
          }), /* @__PURE__ */ jsx("option", {
            value: "other",
            children: "Other"
          })]
        })]
      }), /* @__PURE__ */ jsxs("div", {
        children: [/* @__PURE__ */ jsx("label", {
          className: "block text-sm font-medium text-gray-700",
          children: "Description"
        }), /* @__PURE__ */ jsx("textarea", {
          name: "description",
          required: true,
          rows: 4,
          className: "mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none",
          placeholder: "Describe the issue and its location…"
        })]
      }), /* @__PURE__ */ jsxs("div", {
        children: [/* @__PURE__ */ jsx("label", {
          className: "block text-sm font-medium text-gray-700",
          children: "Your email (optional)"
        }), /* @__PURE__ */ jsx("input", {
          type: "email",
          name: "submitter_email",
          className: "mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none",
          placeholder: "you@example.com"
        })]
      }), /* @__PURE__ */ jsx(DynamicMetadataForm, {
        resourceType: "service_request",
        onChange: setMetadata
      }), /* @__PURE__ */ jsx("button", {
        type: "submit",
        disabled: submitting,
        className: "w-full rounded-md bg-blue-600 px-4 py-2 text-white font-medium hover:bg-blue-700 disabled:opacity-50",
        children: submitting ? "Submitting…" : "Submit request"
      })]
    })]
  });
});
const route2 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  action,
  default: requests_new,
  meta: meta$4
}, Symbol.toStringTag, { value: "Module" }));
async function loader$2({
  params
}) {
  const apiUrl = process.env.API_URL ?? "http://localhost:8080";
  const res = await fetch(`${apiUrl}/v1/permits/${params.id}/status`);
  if (res.status === 404) {
    throw new Response("Permit not found", {
      status: 404
    });
  }
  if (!res.ok) {
    throw new Response("Failed to load permit", {
      status: 502
    });
  }
  const permit = await res.json();
  return {
    permit
  };
}
const meta$3 = ({
  data
}) => [{
  title: `Permit ${(data == null ? void 0 : data.permit.id.slice(0, 8)) ?? "…"} | Fieldstone`
}];
const permits_$id = UNSAFE_withComponentProps(function PermitStatusPage() {
  const {
    permit
  } = useLoaderData();
  return /* @__PURE__ */ jsxs("main", {
    className: "mx-auto max-w-2xl px-4 py-16",
    children: [/* @__PURE__ */ jsx("h1", {
      className: "text-2xl font-bold text-gray-900",
      children: "Permit Status"
    }), /* @__PURE__ */ jsxs("dl", {
      className: "mt-6 divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white",
      children: [/* @__PURE__ */ jsx(Row, {
        label: "Permit ID",
        value: permit.id,
        mono: true
      }), /* @__PURE__ */ jsx(Row, {
        label: "Type",
        value: permit.permit_type
      }), /* @__PURE__ */ jsx(Row, {
        label: "Address",
        value: permit.property_address
      }), /* @__PURE__ */ jsx(Row, {
        label: "Submitted",
        value: new Date(permit.submitted_at).toLocaleDateString()
      }), permit.issued_at && /* @__PURE__ */ jsx(Row, {
        label: "Issued",
        value: new Date(permit.issued_at).toLocaleDateString()
      }), permit.expires_at && /* @__PURE__ */ jsx(Row, {
        label: "Expires",
        value: new Date(permit.expires_at).toLocaleDateString()
      }), /* @__PURE__ */ jsxs("div", {
        className: "flex items-center gap-4 px-4 py-3",
        children: [/* @__PURE__ */ jsx("dt", {
          className: "w-28 shrink-0 text-sm font-medium text-gray-500",
          children: "Status"
        }), /* @__PURE__ */ jsx("dd", {
          children: /* @__PURE__ */ jsx(StatusBadge, {
            status: permit.status
          })
        })]
      })]
    })]
  });
});
function Row({
  label,
  value,
  mono
}) {
  return /* @__PURE__ */ jsxs("div", {
    className: "flex items-center gap-4 px-4 py-3",
    children: [/* @__PURE__ */ jsx("dt", {
      className: "w-28 shrink-0 text-sm font-medium text-gray-500",
      children: label
    }), /* @__PURE__ */ jsx("dd", {
      className: `text-sm text-gray-900 ${mono ? "font-mono text-xs" : ""}`,
      children: value
    })]
  });
}
const statusColors = {
  submitted: "bg-yellow-100 text-yellow-800",
  under_review: "bg-blue-100 text-blue-800",
  approved: "bg-green-100 text-green-800",
  rejected: "bg-red-100 text-red-800",
  expired: "bg-gray-100 text-gray-700"
};
function StatusBadge({
  status
}) {
  const cls = statusColors[status] ?? "bg-gray-100 text-gray-700";
  return /* @__PURE__ */ jsx("span", {
    className: `inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${cls}`,
    children: status.replace(/_/g, " ")
  });
}
const route3 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: permits_$id,
  loader: loader$2,
  meta: meta$3
}, Symbol.toStringTag, { value: "Module" }));
const navLinks = [{
  to: "/staff",
  label: "Dashboard"
}, {
  to: "/staff/requests",
  label: "Requests"
}, {
  to: "/staff/permits",
  label: "Permits"
}];
const staffLayout = UNSAFE_withComponentProps(function StaffLayout() {
  const {
    pathname
  } = useLocation();
  return /* @__PURE__ */ jsxs("div", {
    className: "min-h-screen bg-gray-50",
    children: [/* @__PURE__ */ jsx("nav", {
      className: "border-b border-gray-200 bg-white",
      children: /* @__PURE__ */ jsxs("div", {
        className: "mx-auto flex max-w-7xl items-center justify-between px-4 py-3",
        children: [/* @__PURE__ */ jsx(Link, {
          to: "/",
          className: "text-lg font-semibold text-gray-900",
          children: "Fieldstone Staff"
        }), /* @__PURE__ */ jsx("div", {
          className: "flex gap-1",
          children: navLinks.map(({
            to,
            label
          }) => /* @__PURE__ */ jsx(Link, {
            to,
            className: `rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${pathname === to ? "bg-gray-100 text-gray-900" : "text-gray-600 hover:text-gray-900 hover:bg-gray-50"}`,
            children: label
          }, to))
        })]
      })
    }), /* @__PURE__ */ jsx("main", {
      className: "mx-auto max-w-7xl px-4 py-8",
      children: /* @__PURE__ */ jsx(Outlet, {})
    })]
  });
});
const route4 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: staffLayout
}, Symbol.toStringTag, { value: "Module" }));
const meta$2 = () => [{
  title: "Dashboard | Fieldstone Staff"
}];
const staff_dashboard = UNSAFE_withComponentProps(function DashboardPage() {
  return /* @__PURE__ */ jsxs("div", {
    children: [/* @__PURE__ */ jsx("h1", {
      className: "text-2xl font-bold text-gray-900",
      children: "Dashboard"
    }), /* @__PURE__ */ jsx("p", {
      className: "mt-2 text-gray-500",
      children: "Welcome to the staff portal."
    })]
  });
});
const route5 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: staff_dashboard,
  meta: meta$2
}, Symbol.toStringTag, { value: "Module" }));
const meta$1 = () => [{
  title: "Requests | Fieldstone Staff"
}];
async function loader$1({
  request
}) {
  const apiUrl = process.env.API_URL ?? "http://localhost:8080";
  try {
    const res = await fetch(`${apiUrl}/v1/requests?limit=50`, {
      headers: {
        Authorization: request.headers.get("Authorization") ?? ""
      }
    });
    if (!res.ok) return {
      requests: [],
      total: 0
    };
    const data = await res.json();
    return {
      requests: data.requests ?? [],
      total: data.total ?? 0
    };
  } catch {
    return {
      requests: [],
      total: 0
    };
  }
}
const staff_requests = UNSAFE_withComponentProps(function StaffRequestsPage() {
  const {
    requests,
    total
  } = useLoaderData();
  return /* @__PURE__ */ jsxs("div", {
    children: [/* @__PURE__ */ jsxs("div", {
      className: "flex items-center justify-between",
      children: [/* @__PURE__ */ jsx("h1", {
        className: "text-2xl font-bold text-gray-900",
        children: "Service Requests"
      }), /* @__PURE__ */ jsxs("span", {
        className: "text-sm text-gray-500",
        children: [total, " total"]
      })]
    }), requests.length === 0 ? /* @__PURE__ */ jsx("p", {
      className: "mt-6 text-sm text-gray-500",
      children: "No requests yet."
    }) : /* @__PURE__ */ jsx("div", {
      className: "mt-6 overflow-hidden rounded-lg border border-gray-200 bg-white",
      children: /* @__PURE__ */ jsxs("table", {
        className: "min-w-full divide-y divide-gray-200",
        children: [/* @__PURE__ */ jsx("thead", {
          className: "bg-gray-50",
          children: /* @__PURE__ */ jsx("tr", {
            children: ["Type", "Description", "Status", "Submitted"].map((h) => /* @__PURE__ */ jsx("th", {
              className: "px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500",
              children: h
            }, h))
          })
        }), /* @__PURE__ */ jsx("tbody", {
          className: "divide-y divide-gray-200",
          children: requests.map((req) => /* @__PURE__ */ jsxs("tr", {
            className: "hover:bg-gray-50",
            children: [/* @__PURE__ */ jsx("td", {
              className: "px-4 py-3 text-sm font-medium text-gray-900",
              children: req.request_type
            }), /* @__PURE__ */ jsx("td", {
              className: "max-w-xs truncate px-4 py-3 text-sm text-gray-600",
              children: req.description
            }), /* @__PURE__ */ jsx("td", {
              className: "px-4 py-3 text-sm text-gray-500",
              children: req.status
            }), /* @__PURE__ */ jsx("td", {
              className: "px-4 py-3 text-sm text-gray-500",
              children: new Date(req.created_at).toLocaleDateString()
            })]
          }, req.id))
        })]
      })
    })]
  });
});
const route6 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: staff_requests,
  loader: loader$1,
  meta: meta$1
}, Symbol.toStringTag, { value: "Module" }));
const meta = () => [{
  title: "Permits | Fieldstone Staff"
}];
async function loader({
  request
}) {
  const apiUrl = process.env.API_URL ?? "http://localhost:8080";
  try {
    const res = await fetch(`${apiUrl}/v1/permits?limit=50`, {
      headers: {
        Authorization: request.headers.get("Authorization") ?? ""
      }
    });
    if (!res.ok) return {
      permits: [],
      total: 0
    };
    const data = await res.json();
    return {
      permits: data.permits ?? [],
      total: data.total ?? 0
    };
  } catch {
    return {
      permits: [],
      total: 0
    };
  }
}
const staff_permits = UNSAFE_withComponentProps(function StaffPermitsPage() {
  const {
    permits,
    total
  } = useLoaderData();
  return /* @__PURE__ */ jsxs("div", {
    children: [/* @__PURE__ */ jsxs("div", {
      className: "flex items-center justify-between",
      children: [/* @__PURE__ */ jsx("h1", {
        className: "text-2xl font-bold text-gray-900",
        children: "Permits"
      }), /* @__PURE__ */ jsxs("span", {
        className: "text-sm text-gray-500",
        children: [total, " total"]
      })]
    }), permits.length === 0 ? /* @__PURE__ */ jsx("p", {
      className: "mt-6 text-sm text-gray-500",
      children: "No permits yet."
    }) : /* @__PURE__ */ jsx("div", {
      className: "mt-6 overflow-hidden rounded-lg border border-gray-200 bg-white",
      children: /* @__PURE__ */ jsxs("table", {
        className: "min-w-full divide-y divide-gray-200",
        children: [/* @__PURE__ */ jsx("thead", {
          className: "bg-gray-50",
          children: /* @__PURE__ */ jsx("tr", {
            children: ["Type", "Address", "Status", "Submitted"].map((h) => /* @__PURE__ */ jsx("th", {
              className: "px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500",
              children: h
            }, h))
          })
        }), /* @__PURE__ */ jsx("tbody", {
          className: "divide-y divide-gray-200",
          children: permits.map((p) => /* @__PURE__ */ jsxs("tr", {
            className: "hover:bg-gray-50",
            children: [/* @__PURE__ */ jsx("td", {
              className: "px-4 py-3 text-sm font-medium text-gray-900",
              children: p.permit_type
            }), /* @__PURE__ */ jsx("td", {
              className: "px-4 py-3 text-sm text-gray-600",
              children: p.property_address
            }), /* @__PURE__ */ jsx("td", {
              className: "px-4 py-3 text-sm text-gray-500",
              children: p.status
            }), /* @__PURE__ */ jsx("td", {
              className: "px-4 py-3 text-sm text-gray-500",
              children: /* @__PURE__ */ jsx(Link, {
                to: `/permits/${p.id}`,
                className: "hover:underline",
                children: new Date(p.submitted_at).toLocaleDateString()
              })
            })]
          }, p.id))
        })]
      })
    })]
  });
});
const route7 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: staff_permits,
  loader,
  meta
}, Symbol.toStringTag, { value: "Module" }));
const serverManifest = { "entry": { "module": "/assets/entry.client-B2Jgwbm9.js", "imports": ["/assets/chunk-5KNZJZUH-DU3gRWdF.js"], "css": [] }, "routes": { "root": { "id": "root", "parentId": void 0, "path": "", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/root-sqtfWVfw.js", "imports": ["/assets/chunk-5KNZJZUH-DU3gRWdF.js"], "css": ["/assets/root-DKNWYgYW.css"], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/home": { "id": "routes/home", "parentId": "root", "path": void 0, "index": true, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/home-B07CydaF.js", "imports": ["/assets/chunk-5KNZJZUH-DU3gRWdF.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/requests.new": { "id": "routes/requests.new", "parentId": "root", "path": "requests/new", "index": void 0, "caseSensitive": void 0, "hasAction": true, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/requests.new-DkcbUEpm.js", "imports": ["/assets/chunk-5KNZJZUH-DU3gRWdF.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/permits.$id": { "id": "routes/permits.$id", "parentId": "root", "path": "permits/:id", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": true, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/permits._id-DiqWhCu7.js", "imports": ["/assets/chunk-5KNZJZUH-DU3gRWdF.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/staff-layout": { "id": "routes/staff-layout", "parentId": "root", "path": void 0, "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/staff-layout-B9u-1G-W.js", "imports": ["/assets/chunk-5KNZJZUH-DU3gRWdF.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/staff.dashboard": { "id": "routes/staff.dashboard", "parentId": "routes/staff-layout", "path": "staff", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/staff.dashboard-z2a-0W0V.js", "imports": ["/assets/chunk-5KNZJZUH-DU3gRWdF.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/staff.requests": { "id": "routes/staff.requests", "parentId": "routes/staff-layout", "path": "staff/requests", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": true, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/staff.requests-BPQOwxAH.js", "imports": ["/assets/chunk-5KNZJZUH-DU3gRWdF.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/staff.permits": { "id": "routes/staff.permits", "parentId": "routes/staff-layout", "path": "staff/permits", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": true, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/staff.permits-DzctGtGY.js", "imports": ["/assets/chunk-5KNZJZUH-DU3gRWdF.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 } }, "url": "/assets/manifest-67c3f875.js", "version": "67c3f875", "sri": void 0 };
const assetsBuildDirectory = "build/client";
const basename = "/";
const future = { "unstable_optimizeDeps": false, "v8_passThroughRequests": false, "unstable_trailingSlashAwareDataRequests": false, "unstable_previewServerPrerendering": false, "v8_middleware": false, "v8_splitRouteModules": false, "v8_viteEnvironmentApi": false };
const ssr = true;
const isSpaMode = false;
const prerender = [];
const routeDiscovery = { "mode": "lazy", "manifestPath": "/__manifest" };
const publicPath = "/";
const entry = { module: entryServer };
const routes = {
  "root": {
    id: "root",
    parentId: void 0,
    path: "",
    index: void 0,
    caseSensitive: void 0,
    module: route0
  },
  "routes/home": {
    id: "routes/home",
    parentId: "root",
    path: void 0,
    index: true,
    caseSensitive: void 0,
    module: route1
  },
  "routes/requests.new": {
    id: "routes/requests.new",
    parentId: "root",
    path: "requests/new",
    index: void 0,
    caseSensitive: void 0,
    module: route2
  },
  "routes/permits.$id": {
    id: "routes/permits.$id",
    parentId: "root",
    path: "permits/:id",
    index: void 0,
    caseSensitive: void 0,
    module: route3
  },
  "routes/staff-layout": {
    id: "routes/staff-layout",
    parentId: "root",
    path: void 0,
    index: void 0,
    caseSensitive: void 0,
    module: route4
  },
  "routes/staff.dashboard": {
    id: "routes/staff.dashboard",
    parentId: "routes/staff-layout",
    path: "staff",
    index: void 0,
    caseSensitive: void 0,
    module: route5
  },
  "routes/staff.requests": {
    id: "routes/staff.requests",
    parentId: "routes/staff-layout",
    path: "staff/requests",
    index: void 0,
    caseSensitive: void 0,
    module: route6
  },
  "routes/staff.permits": {
    id: "routes/staff.permits",
    parentId: "routes/staff-layout",
    path: "staff/permits",
    index: void 0,
    caseSensitive: void 0,
    module: route7
  }
};
const allowedActionOrigins = false;
export {
  allowedActionOrigins,
  serverManifest as assets,
  assetsBuildDirectory,
  basename,
  entry,
  future,
  isSpaMode,
  prerender,
  publicPath,
  routeDiscovery,
  routes,
  ssr
};
