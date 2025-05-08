// i18n-processed-v1.1.0 (no translatable strings)
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
