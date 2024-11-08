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
  UsersRoundIcon,
  SettingsIcon,
  EditIcon,
} from "lucide-react";
import batchRoutes from "./Job/Batch";
import interactiveRoutes from "./Job/Interactive";
import datasetRoutes from "./Data";
import { NavGroupProps } from "@/components/sidebar/nav-main";
import AuthedRouter from "./AuthedRouter";

const portalRoutes: SidebarItem[] = [
  {
    path: "overview",
    icon: BarChartBigIcon,
    children: [],
    route: {
      path: "overview/*",
      lazy: () => import("./Overview"),
    },
  },
  {
    path: "job",
    icon: FlaskConicalIcon,
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
    icon: BriefcaseIcon,
    children: [
      {
        route: {
          path: "microservice/*",
        },
      },
      {
        route: {
          path: "serverless/*",
        },
      },
    ],
  },
  {
    path: "image",
    icon: BoxIcon,
    children: [
      {
        route: {
          path: "createimage/*",
          lazy: () => import("./Image/CreateImageList"),
        },
      },
      {
        route: {
          path: "uploadimage/*",
          lazy: () => import("./Image/UploadImageList"),
        },
      },
    ],
  },
  {
    path: "data",
    icon: DatabaseIcon,
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
    icon: UsersRoundIcon,
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
    icon: SettingsIcon,
    children: [],
    route: {
      path: "setting",
      lazy: () => import("./Setting"),
    },
  },
  {
    path: "feedback",
    icon: MessageSquareMoreIcon,
    children: [],
    route: {
      path: "feedback",
      lazy: () => import("./Feedback"),
    },
  },
];

const userSidebarGroups: NavGroupProps[] = [
  {
    title: "作业与服务",
    items: [
      {
        title: "集群概览",
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
            title: "微服务",
            url: "service/microservice",
          },
          {
            title: "无服务",
            url: "service/serverless",
          },
        ],
      },
    ],
  },
  {
    title: "数据与镜像",
    items: [
      {
        title: "我的镜像",
        url: "image",
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
        title: "账户管理",
        url: "account",
        icon: UsersRoundIcon,
        items: [
          {
            title: "成员管理",
            url: "account/member",
          },
        ],
      },
      {
        title: "系统设置",
        url: "setting",
        icon: SettingsIcon, // 假设你有一个系统设置的图标
        items: [],
      },
      {
        title: "问题反馈",
        url: "feedback",
        icon: EditIcon, // 假设你有一个问题反馈的图标
        items: [],
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
      element: <h1>404</h1>,
    },
  ],
};
