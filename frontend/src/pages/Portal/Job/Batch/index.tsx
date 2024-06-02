import { RouteObject } from "react-router-dom";

const batchRoutes: RouteObject[] = [
  {
    index: true,
    lazy: () => import("./BatchOverview"),
  },
  {
    path: "new-training",
    lazy: () => import("../New/Training"),
  },
  {
    path: "new-tensorflow",
    lazy: () => import("../New/Tensorflow"),
  },
  {
    path: ":id",
    lazy: () => import("./Detail"),
  },
];

export default batchRoutes;
