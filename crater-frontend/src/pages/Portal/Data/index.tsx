import { RouteObject } from "react-router-dom";
import { apiGetDataset } from "@/services/api/dataset";
import { DataView } from "@/components/custom/DataView";

const datasetRoutes: RouteObject[] = [
  {
    index: true,
    element: <DataView apiGetDataset={apiGetDataset} sourceType="dataset" />,
  },
  {
    path: ":id",
    lazy: () => import("./DatasetShare"),
  },
];

const modelRoutes: RouteObject[] = [
  {
    index: true,
    element: <DataView apiGetDataset={apiGetDataset} sourceType="model" />,
  },
  {
    path: ":id",
    lazy: () => import("./ModelShare"),
  },
];

const shareFileRoutes: RouteObject[] = [
  {
    index: true,
    element: <DataView apiGetDataset={apiGetDataset} sourceType="sharefile" />, // 传入不同的resourceType
  },
  {
    path: ":id",
    lazy: () => import("./ShareFileShare"), // 确保返回符合LazyRouteFunction类型的组件加载器
  },
];

export { datasetRoutes, modelRoutes, shareFileRoutes };
