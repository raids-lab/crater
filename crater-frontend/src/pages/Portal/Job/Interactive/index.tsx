import { RouteObject } from "react-router-dom";
import InterOverview from "./InterOverview";
import { Base } from "../Detail/Base";

const interactiveRoutes: RouteObject[] = [
  {
    index: true,
    element: <InterOverview />,
  },
  {
    path: "new-jupyter-vcjobs",
    lazy: () => import("../New/Jupyter"),
  },
  {
    path: "new-jupyter-aijobs",
    lazy: () => import("../New/EmiasJupyter"),
  },
  {
    path: ":name",
    element: <Base />,
  },
];

export default interactiveRoutes;
