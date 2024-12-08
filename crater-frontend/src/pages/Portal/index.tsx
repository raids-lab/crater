import { SidebarItem } from "@/components/layout/Sidebar";
import { Navigate, RouteObject } from "react-router-dom";
import DashboardLayout from "@/components/layout/Dashboard";
import {
  BarChartBigIcon,
  BriefcaseIcon,
  BoxIcon,
  DatabaseIcon,
  FlaskConicalIcon,
  MessageSquareMoreIcon,
  SettingsIcon,
} from "lucide-react";
import batchRoutes from "./Job/Batch";
import interactiveRoutes from "./Job/Interactive";
import datasetRoutes from "./Data";
import { NavGroupProps } from "@/components/sidebar/nav-main";
import AuthedRouter from "./AuthedRouter";
import NotFound from "@/components/layout/NotFound";
import FeedBack from "./Feedback";
import UserSettings from "./Setting/UserSetting";

const portalRoutes: SidebarItem[] = [
  {
    path: "overview",
    children: [],
    route: {
      path: "overview/*",
      lazy: () => import("./Overview"),
    },
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
    children: [
      {
        route: {
          path: "tensorboard/*",
          element: <>TODO</>,
        },
      },
      {
        route: {
          path: "model/*",
          element: <>TODO</>,
        },
      },
    ],
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
      {
        route: {
          path: "platform",
          lazy: () => import("./Setting"),
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
    title: "作业与服务",
    items: [
      {
        title: "平台概览",
        url: "overview",
        icon: BarChartBigIcon,
      },
      {
        title: "我的作业",
        url: "job",
        icon: FlaskConicalIcon,
        isActive: true,
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
        title: "我的服务",
        url: "service",
        icon: BriefcaseIcon,
        items: [
          {
            title: "Tensorboard",
            url: "service/tensorboard",
          },
          {
            title: "模型托管",
            url: "service/model",
          },
        ],
      },
    ],
  },
  {
    title: "数据与镜像",
    items: [
      {
        title: "镜像管理",
        url: "image",
        icon: BoxIcon,
        items: [
          {
            title: "镜像仓库",
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
        url: "data",
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
        ],
      },
    ],
  },
  {
    title: "其他",
    items: [
      {
        title: "设置",
        url: "setting",
        icon: SettingsIcon,
        items: [
          {
            title: "用户设置",
            url: "setting/user",
          },
          {
            title: "平台设置",
            url: "setting/platform",
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
