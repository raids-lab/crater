import { Navigate, RouteObject } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren } from "react";
import { Role } from "@/services/api/auth";
import DashboardLayout from "@/components/layout/Dashboard";
import { User } from "./User";
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
import adminJobRoutes from "./Job";
import NotFound from "@/components/layout/NotFound";
import UserDetail from "@/components/custom/UserDetail";
import SystemSetting from "../Portal/Setting/SystemSetting";
import CronPolicy from "./Job/CronPolicy";
import NetworkOverview from "@/components/monitor/NetworkOverview";
import NvidiaOverview from "@/components/monitor/NvidiaOverview";
import { useTranslation } from "react-i18next";

const routeItems: RouteObject[] = [
  {
    path: "cluster",
    children: [
      {
        path: "node/*",
        lazy: () => import("./Cluster/Node"),
      },
      {
        path: "resource",
        element: <Resource />,
      },
    ],
  },
  {
    path: "monitor",
    children: [
      {
        path: "network",
        element: <NetworkOverview />,
      },
      {
        path: "gpu",
        element: <NvidiaOverview />,
      },
    ],
  },
  {
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
  {
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
  {
    path: "job/*",
    children: adminJobRoutes,
  },
  {
    path: "cron",
    element: <CronPolicy />,
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
        children: admindatasetRoutes,
      },
    ],
  },
  {
    path: "setting",
    element: <SystemSetting />,
  },
];

const useAdminSidebarGroups = (): NavGroupProps[] => {
  const { t } = useTranslation();

  return [
    {
      title: t("sidebar.resourceAndMonitoring"),
      items: [
        {
          title: t("navigation.resourceManagement"),
          icon: ServerIcon,
          items: [
            {
              title: t("navigation.nodeManagement"),
              url: "cluster/node",
            },
            {
              title: t("navigation.resourceManagement"),
              url: "cluster/resource",
            },
          ],
        },
        {
          title: t("navigation.clusterMonitoring"),
          icon: BarChartBigIcon,
          items: [
            {
              title: t("navigation.gpuMonitoring"),
              url: "monitor/gpu",
            },
            {
              title: t("navigation.networkMonitoring"),
              url: "monitor/network",
            },
          ],
        },
      ],
    },
    {
      title: t("sidebar.jobsAndServices"),
      items: [
        {
          title: t("navigation.jobManagement"),
          url: "job",
          icon: FlaskConicalIcon,
        },
        {
          title: t("navigation.cronPolicy"),
          url: "cron",
          icon: AlarmClockIcon,
        },
      ],
    },
    {
      title: t("sidebar.usersAndAccounts"),
      items: [
        {
          title: t("navigation.userManagement"),
          url: "user",
          icon: UserRoundIcon,
        },
        {
          title: t("navigation.accountManagement"),
          url: "account",
          icon: UsersRoundIcon,
        },
      ],
    },
    {
      title: t("sidebar.dataAndImages"),
      items: [
        {
          title: t("navigation.imageManagement"),
          icon: BoxIcon,
          items: [
            {
              title: t("navigation.imageCreation"),
              url: "image/createimage",
            },
            {
              title: t("navigation.imageList"),
              url: "image/uploadimage",
            },
          ],
        },
        {
          title: t("navigation.dataManagement"),
          icon: DatabaseIcon,
          items: [
            {
              title: t("navigation.fileSystem"),
              url: "data/filesystem",
            },
            {
              title: t("sidebar.data"),
              url: "data/dataset",
            },
          ],
        },
      ],
    },
    {
      title: t("navigation.settings"),
      items: [
        {
          title: t("navigation.platformSettings"),
          icon: SettingsIcon,
          url: "setting",
        },
      ],
    },
  ];
};

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth(Role.Admin);
  return isAuthenticated ? children : <Navigate to="/portal" replace />;
};

export const adminRoute: RouteObject = {
  path: "/admin",
  element: (
    <AuthedRouter>
      <DashboardLayoutWrapper />
    </AuthedRouter>
  ),
  children: [
    {
      index: true,
      element: <Navigate to="cluster/node" replace={true} />,
    },
    ...routeItems,
    {
      path: "*",
      element: <NotFound />,
    },
  ],
};

function DashboardLayoutWrapper() {
  const groups = useAdminSidebarGroups();
  return <DashboardLayout groups={groups} />;
}
