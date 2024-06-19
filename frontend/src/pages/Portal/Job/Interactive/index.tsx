import { RouteObject } from "react-router-dom";

const interactiveRoutes: RouteObject[] = [
  {
    index: true,
    lazy: () => import("./InterOverview"),
  },
  {
    path: "new-jupyter",
    lazy: () => import("../New/Jupyter"),
  },
  {
    path: ":id",
    lazy: () => import("../Detail/Interactive"),
  },
];

export default interactiveRoutes;
