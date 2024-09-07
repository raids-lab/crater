import { RouteObject } from "react-router-dom";

import AdminDataset from "./Dataset";
const admindatasetRoutes: RouteObject[] = [
  {
    index: true,
    element: <AdminDataset />,
  },
  {
    path: ":id",
    lazy: () => import("./DatasetShare"),
  },
];

export default admindatasetRoutes;
