import { RouteItem } from "@/components/layout/Sidebar";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense } from "react";
import { Navigate, RouteObject } from "react-router-dom";
import { Role } from "@/services/api/auth";
import DashboardLayout from "@/components/layout/Dashboard";
import { User } from "./User";
// import Volcano from "./Job/Volcano";
import Resource from "./Cluster/Resource";
import AccountDetail from "./Account/Detail";
import {
  BoxIcon,
  SettingsIcon,
  DatabaseIcon,
  FlaskConicalIcon,
  ServerIcon,
  UserRoundIcon,
  UsersRoundIcon,
  BarChartBigIcon,
  AlarmClockIcon,
} from "lucide-react";
import { Account } from "./Account";
import admindatasetRoutes from "./Data";
import { NavGroupProps } from "@/components/sidebar/types";
import AdminJob from "./Job";
import NotFound from "@/components/layout/NotFound";
import UserDetail from "@/components/custom/UserDetail";
import SystemSetting from "../Portal/Setting/SystemSetting";
import Monitor from "../Embed/Monitor";
import CronPolicy from "./Job/CronPolicy";

const OVERVIEW_DASHBOARD = import.meta.env.VITE_GRAFANA_OVERVIEW;

const routeItems: RouteItem[] = [
  {
    path: "cluster",
    children: [
      {
        route: {
          path: "node/*",
          lazy: () => import("./Cluster/Node"),
        },
      },
      {
        route: {
          path: "label",
          lazy: () => import("./Cluster/Label"),
        },
      },
      {
        route: {
          path: "resource",
          element: <Resource />,
        },
      },
    ],
  },
  {
    path: "monitor",
    children: [
      {
        route: {
          path: "network",
          element: <Monitor baseSrc={OVERVIEW_DASHBOARD} />,
        },
      },
      {
        route: {
          path: "io",
          element: <Monitor baseSrc={OVERVIEW_DASHBOARD} />,
        },
      },
    ],
  },
  {
    path: "account",
    route: {
      path: "account/*",
      children: [
        {
          index: true,
          element: <Account />,
        },
        {
          path: ":id",
          element: <AccountDetail />,
        },
      ],
    },
    children: [],
  },
  {
    path: "user",
    children: [],
    route: {
      path: "user/*",
      children: [
        {
          index: true,
          element: <User />,
        },
        {
          path: ":name",
          element: <UserDetail />,
        },
      ],
    },
  },
  {
    path: "job",
    route: {
      path: "job/*",
      element: <AdminJob />,
    },
    children: [],
  },
  {
    path: "cron",
    children: [],
    route: {
      path: "cron",
      element: <CronPolicy />,
    },
  },
  // {
  //   path: "image",
  //   icon: BoxIcon,
  //   children: [
  //     {
  //       route: {
  //         path: "create",
  //         lazy: () => import("./Image/Create"),
  //       },
  //     },
  //     {
  //       route: {
  //         path: "upload",
  //         lazy: () => import("./Image/Upload"),
  //       },
  //     },
  //   ],
  // },
  {
    path: "data",
    children: [
      {
        route: {
          path: "account/*",
          lazy: () => import("./Data/Queue"),
        },
      },
      {
        route: {
          path: "user/*",
          lazy: () => import("./Data/User"),
        },
      },
      {
        route: {
          path: "dataset/*",
          children: admindatasetRoutes,
        },
      },
    ],
  },
  {
    path: "setting",
    children: [],
    route: {
      path: "setting",
      element: <SystemSetting />,
    },
  },
];

const adminSidebarGroups: NavGroupProps[] = [
  {
    title: "资源与监控",
    items: [
      {
        title: "资源管理",
        icon: ServerIcon,
        items: [
          {
            title: "节点管理",
            url: "cluster/node",
          },
          {
            title: "节点标签",
            url: "cluster/label",
          },
          {
            title: "资源列表",
            url: "cluster/resource",
          },
        ],
      },
      {
        title: "集群监控",
        icon: BarChartBigIcon,
        items: [
          {
            title: "网络监控",
            url: "monitor/network",
          },
          {
            title: "读写监控",
            url: "monitor/io",
          },
        ],
      },
    ],
  },
  {
    title: "作业与服务",
    items: [
      {
        title: "作业管理",
        url: "job",
        icon: FlaskConicalIcon,
      },
      {
        title: "定时策略",
        url: "cron",
        icon: AlarmClockIcon,
      },
    ],
  },
  {
    title: "用户与账户",
    items: [
      {
        title: "用户管理",
        url: "user",
        icon: UserRoundIcon,
      },
      {
        title: "账户管理",
        url: "account",
        icon: UsersRoundIcon,
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
            url: "image/create",
          },
          {
            title: "镜像列表",
            url: "image/upload",
          },
        ],
      },
      {
        title: "数据管理",
        icon: DatabaseIcon,
        items: [
          {
            title: "用户文件",
            url: "data/user",
          },
          {
            title: "账户文件",
            url: "data/account",
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
    title: "设置",
    items: [
      {
        title: "平台设置",
        icon: SettingsIcon,
        url: "setting",
      },
    ],
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth(Role.Admin);
  return isAuthenticated ? children : <Navigate to="/portal" replace />;
};

export const adminRoute: RouteObject = {
  path: "/admin",
  element: (
    <Suspense fallback={<h2>Loading...</h2>}>
      <AuthedRouter>
        <DashboardLayout groups={adminSidebarGroups} />
      </AuthedRouter>
    </Suspense>
  ),
  children: [
    {
      index: true,
      element: <Navigate to="cluster/node" replace={true} />,
    },
    ...routeItems.map((item) => {
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
