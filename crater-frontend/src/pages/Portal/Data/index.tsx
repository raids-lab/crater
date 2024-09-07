import { RouteObject } from "react-router-dom";

import DatasetDetail from "./Dataset";
const datasetRoutes: RouteObject[] = [
  {
    index: true,
    element: <DatasetDetail />,
  },
  {
    path: ":id",
    lazy: () => import("./DatasetShare"),
  },
];

export default datasetRoutes;
