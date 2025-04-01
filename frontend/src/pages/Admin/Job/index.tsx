import { RouteObject } from "react-router-dom";
import AdminJobOverview from "./Overview";

const adminJobRoutes: RouteObject[] = [
  {
    index: true,
    element: <AdminJobOverview />,
  },
  {
    path: ":name",
    lazy: () => import("../../Portal/Job/Detail/Base"),
  },
];

export default adminJobRoutes;
