import { RouteObject } from "react-router-dom";
import AdminJobOverview from "./Overview";
import { Base } from "@/pages/Portal/Job/Detail/Base";

const adminJobRoutes: RouteObject[] = [
  {
    index: true,
    element: <AdminJobOverview />,
  },
  {
    path: ":name",
    element: <Base />,
  },
];

export default adminJobRoutes;
