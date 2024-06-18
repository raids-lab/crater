import { RouteObject } from "react-router-dom";
import VolcanoOverview from "./Volcano/Overview";
import { globalJobUrl, store } from "@/utils/store";
import ColocateOverview from "./Colocate/Overview";

const jobType = store.get(globalJobUrl);
const batchRoutes: RouteObject[] = [
  {
    index: true,
    element: jobType === "aijobs" ? <ColocateOverview /> : <VolcanoOverview />,
  },
  {
    path: ":id",
    lazy: () => import("../Interactive/Detail"),
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
];

export default batchRoutes;
