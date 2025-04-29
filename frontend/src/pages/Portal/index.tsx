import { Navigate, RouteObject } from "react-router-dom";
import DashboardLayout from "@/components/layout/Dashboard";
import {
  BarChartBigIcon,
  BoxIcon,
  DatabaseIcon,
  FlaskConicalIcon,
  MessageSquareMoreIcon,
  SettingsIcon,
  SquareChartGanttIcon,
  ShoppingBagIcon,
} from "lucide-react";
import batchRoutes from "./Job/Batch";
import interactiveRoutes from "./Job/Interactive";
import datasetRoutes from "./Data";
import { NavGroupProps } from "@/components/sidebar/types";
import AuthedRouter from "./AuthedRouter";
import NotFound from "@/components/layout/NotFound";
import FeedBack from "./Feedback";
import UserSettings from "./Setting/UserSetting";
import modelRoutes from "./Data/modelIndex";
import AssignmentTemplateList from "./Job/Store";
import NvidiaOverview from "@/components/monitor/NvidiaOverview";
import ResourseOverview from "@/components/monitor/ResourceOverview";
import NetworkOverview from "@/components/monitor/NetworkOverview";
import UserDetail from "@/components/custom/UserDetail";
import shareFileRoutes from "./Data/ShareFile";

const portalRoutes: RouteObject[] = [
  {
    path: "overview/*",
    lazy: () => import("./Overview"),
  },
  {
    path: "monitor",
    children: [
      {
        path: "gpu",
        element: <NvidiaOverview />,
      },
      {
        path: "node",
        element: <ResourseOverview />,
      },
      {
        path: "network",
        element: <NetworkOverview />,
      },
    ],
  },
  {
    path: "job",
    children: [
      {
        path: "inter/*",
        children: interactiveRoutes,
      },
      {
        path: "batch/*",
        children: batchRoutes,
      },
    ],
  },
  {
    path: "modal",
    element: <AssignmentTemplateList />,
  },
  {
    path: "image",
    children: [
      {
        path: "createimage/*",
        lazy: () => import("./Image/Registry"),
      },
      {
        path: "uploadimage/*",
        lazy: () => import("./Image/Image"),
      },
    ],
  },
  {
    path: "data",
    children: [
      {
        path: "filesystem/*",
        lazy: () => import("./Data/FileSystem"),
      },
      {
        path: "dataset/*",
        children: datasetRoutes,
      },
      {
        path: "model/*",
        children: modelRoutes,
      },
      {
        path: "sharefile/*",
        children: shareFileRoutes,
      },
    ],
  },
  {
    path: "account",
    children: [
      {
        path: "member",
      },
    ],
  },
  {
    path: "user/*",
    children: [
      {
        path: ":name",
        element: <UserDetail />,
      },
    ],
  },
  {
    path: "setting",
    children: [
      {
        path: "user",
        element: <UserSettings />,
      },
    ],
  },
  {
    path: "feedback",
    element: <FeedBack />,
  },
];

const userSidebarGroups: NavGroupProps[] = [
  {
    title: "资源与监控",
    items: [
      {
        title: "平台概览",
        url: "overview",
        icon: SquareChartGanttIcon,
      },
      {
        title: "集群监控",
        icon: BarChartBigIcon,
        items: [
          {
            title: "GPU 大盘",
            url: "monitor/gpu",
          },
          {
            title: "空闲资源",
            url: "monitor/node",
          },
          {
            title: "网络监控",
            url: "monitor/network",
          },
        ],
      },
    ],
  },
  {
    title: "作业与服务",
    items: [
      {
        title: "我的作业",
        icon: FlaskConicalIcon,
        items: [
          {
            title: "自定义作业",
            url: "job/batch",
          },
          {
            title: "Jupyter Lab",
            url: "job/inter",
          },
        ],
      },
      {
        title: "作业模板",
        url: "modal",
        icon: ShoppingBagIcon,
      },
    ],
  },
  {
    title: "数据与镜像",
    items: [
      {
        title: "镜像管理",
        icon: BoxIcon,
        items: [
          {
            title: "镜像制作",
            url: "image/createimage",
          },
          {
            title: "镜像列表",
            url: "image/uploadimage",
          },
        ],
      },
      {
        title: "数据管理",
        icon: DatabaseIcon,
        items: [
          {
            title: "文件系统",
            url: "data/filesystem",
          },
          {
            title: "数据集",
            url: "data/dataset",
          },
          {
            title: "模型",
            url: "data/model",
          },
          {
            title: "共享文件",
            url: "data/sharefile",
          },
        ],
      },
    ],
  },
  {
    title: "其他",
    items: [
      {
        title: "设置",
        icon: SettingsIcon,
        items: [
          {
            title: "用户设置",
            url: "setting/user",
          },
        ],
      },
      {
        title: "帮助与反馈",
        url: "feedback",
        icon: MessageSquareMoreIcon,
      },
    ],
  },
];

export const portalRoute: RouteObject = {
  path: "/portal",
  element: (
    <AuthedRouter>
      <DashboardLayout groups={userSidebarGroups} />
    </AuthedRouter>
  ),
  children: [
    {
      index: true,
      element: <Navigate to="overview" replace />,
    },
    ...portalRoutes,
    {
      path: "*",
      element: <NotFound />,
    },
  ],
};
