import { RouteObject } from "react-router-dom";

const batchRoutes: RouteObject[] = [
  {
    index: true,
    lazy: () => import("./BatchOverview"),
  },
  {
    path: "new-aijobs",
    lazy: () => import("../New/Custom"),
  },
  {
    path: "new-vcjobs",
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
    path: "new-spjobs",
    lazy: () => import("../New/Sparse"),
  },
  {
    path: ":id",
    lazy: () => import("../Interactive/Detail"),
  },
];

export default batchRoutes;
