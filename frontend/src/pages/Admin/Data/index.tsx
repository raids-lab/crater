// i18n-processed-v1.1.0 (no translatable strings)
import { RouteObject } from "react-router-dom";
import { AdminDatasetTable } from "@/components/custom/adminDataset";

const admindatasetRoutes: RouteObject[] = [
  {
    index: true,
    element: <AdminDatasetTable />,
  },
  {
    path: ":id",
    lazy: () => import("./DatasetShare"),
  },
];

export default admindatasetRoutes;
