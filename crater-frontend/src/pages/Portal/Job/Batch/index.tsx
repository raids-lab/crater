import { RouteObject } from "react-router-dom";

const batchRoutes: RouteObject[] = [
  {
    index: true,
    lazy: () => import("./BatchOverview"),
  },
  {
    path: "new-custom",
    lazy: () => import("../New/Custom"),
  },
  {
    path: "new-tensorflow",
    lazy: () => import("../New/Tensorflow"),
  },
  {
    path: "new-pytorch",
    lazy: () => import("../New/Pytorch"),
  },
  {
    path: ":id",
    lazy: () => import("../Interactive/Detail"),
  },
];

export default batchRoutes;
