import { RouteObject } from "react-router-dom";

import Modeldetail from "./Model";
const modelRoutes: RouteObject[] = [
  {
    index: true,
    element: <Modeldetail sourceType="model" />,
  },
  {
    path: ":id",
    lazy: () => import("./ModelShare"),
  },
];

export default modelRoutes;
