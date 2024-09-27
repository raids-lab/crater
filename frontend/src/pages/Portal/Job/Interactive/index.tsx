import { RouteObject } from "react-router-dom";
import InterOverview from "./InterOverview";
import ColocateInterOverview from "./ColocateInterOverview";
import { globalJobUrl, store } from "@/utils/store";

const jobType = store.get(globalJobUrl);
const interactiveRoutes: RouteObject[] = [
  {
    index: true,
    element:
      jobType === "aijobs" ? <ColocateInterOverview /> : <InterOverview />,
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
