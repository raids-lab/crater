import { RouteObject } from "react-router-dom";

import Modeldetail from "./Model";
const modelRoutes: RouteObject[] = [
  {
    index: true,
    element: <Modeldetail />,
  },
  {
    path: ":id",
    lazy: () => import("./ModelShare"),
  },
];

export default modelRoutes;
