import { RouteObject } from "react-router-dom";

const interactiveRoutes: RouteObject[] = [
  {
    index: true,
    lazy: () => import("./InterOverview"),
  },
  {
    path: "new-jupyter-vcjobs",
    lazy: () => import("../New/Jupyter"),
  },
  {
    path: "new-jupyter-aijobs",
    lazy: () => import("../New/ColocateJupyter"),
  },
  {
    path: ":id",
    lazy: () => import("../Detail/Interactive"),
  },
];

export default interactiveRoutes;
