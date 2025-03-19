import { RouteItem } from "@/components/layout/Sidebar";
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
import Monitor from "../Embed/Monitor";
import modelRoutes from "./Data/modelindex";
import AssignmentTemplateList from "./Job/Store";

const OVERVIEW_DASHBOARD = import.meta.env.VITE_GRAFANA_OVERVIEW;
const AVAILABLE_DASHBOARD = import.meta.env.VITE_GRAFANA_SCHEDULE;
const NETWORK_DASHBOARD = import.meta.env.VITE_GRAFANA_NETWORK;

const portalRoutes: RouteItem[] = [
  {
    path: "overview",
    children: [],
    route: {
      path: "overview/*",
      lazy: () => import("./Overview"),
    },
  },
  {
    path: "monitor",
    children: [
      {
        route: {
          path: "gpu",
          element: <Monitor baseSrc={OVERVIEW_DASHBOARD} />,
        },
      },
      {
        route: {
          path: "node",
          element: <Monitor baseSrc={AVAILABLE_DASHBOARD} />,
        },
      },
      {
        route: {
          path: "network",
          element: <Monitor baseSrc={NETWORK_DASHBOARD} />,
        },
      },
    ],
  },
  {
    path: "job",
    children: [
      {
        route: {
          path: "inter/*",
          children: interactiveRoutes,
        },
      },
      {
        route: {
          path: "batch/*",
          children: batchRoutes,
        },
      },
    ],
  },
  {
    path: "service",
    route: {
      path: "modal",
      element: <AssignmentTemplateList />,
    },
    children: [],
  },
  {
    path: "image",
    children: [
      {
        route: {
          path: "createimage/*",
          lazy: () => import("./Image/Registry"),
        },
      },
      {
        route: {
          path: "uploadimage/*",
          lazy: () => import("./Image/Image"),
        },
      },
    ],
  },
  {
    path: "data",
    children: [
      {
        route: {
          path: "filesystem/*",
          lazy: () => import("./Data/FileSystem"),
        },
      },
      {
        route: {
          path: "dataset/*",
          children: datasetRoutes,
        },
      },
      {
        route: {
          path: "model/*",
          children: modelRoutes,
        },
      },
    ],
  },
  {
    path: "account",
    children: [
      {
        route: {
          path: "member",
        },
      },
    ],
  },
  {
    path: "setting",
    children: [
      {
        route: {
          path: "user",
          element: <UserSettings />,
        },
      },
    ],
  },
  {
    path: "feedback",
    children: [],
    route: {
      path: "feedback",
      element: <FeedBack />,
    },
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
            title: "批处理作业",
            url: "job/batch",
          },
          {
            title: "交互式作业",
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
    ...portalRoutes.map((item) => {
      return (
        item.route ?? {
          path: item.path,
          children: item.children.map((child) => child.route),
        }
      );
    }),
    {
      path: "*",
      element: <NotFound />,
    },
  ],
};
