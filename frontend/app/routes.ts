import { type RouteConfig, index, layout, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("requests/new", "routes/requests.new.tsx"),
  route("permits/:id", "routes/permits.$id.tsx"),
  layout("routes/staff-layout.tsx", [
    route("staff", "routes/staff.dashboard.tsx"),
    route("staff/requests", "routes/staff.requests.tsx"),
    route("staff/permits", "routes/staff.permits.tsx"),
  ]),
] satisfies RouteConfig;
