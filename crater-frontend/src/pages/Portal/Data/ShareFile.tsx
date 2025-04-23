import { RouteObject } from "react-router-dom";
import Modeldetail from "./Model"; // 复用Model组件但传入不同参数

const shareFileRoutes: RouteObject[] = [
  {
    index: true,
    element: <Modeldetail sourceType="sharefile" />, // 传入不同的resourceType
  },
  {
    path: ":id",
    lazy: () => import("./ShareFileShare"), // 确保返回符合LazyRouteFunction类型的组件加载器
  },
];

export default shareFileRoutes;
