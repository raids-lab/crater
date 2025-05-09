// i18n-processed-v1.1.0 (no translatable strings)
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
import { datasetRoutes, modelRoutes, shareFileRoutes } from "./Data";
import { NavGroupProps } from "@/components/sidebar/types";
import AuthedRouter from "./AuthedRouter";
import NotFound from "@/components/layout/NotFound";
import FeedBack from "./Feedback";
import UserSettings from "./Setting/UserSetting";
import AssignmentTemplateList from "./Job/Store";
import NvidiaOverview from "@/components/monitor/NvidiaOverview";
import ResourseOverview from "@/components/monitor/ResourceOverview";
import NetworkOverview from "@/components/monitor/NetworkOverview";
import UserDetail from "@/components/custom/UserDetail";
import { useTranslation } from "react-i18next";

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

// 使用 hook 获取翻译版的侧边栏组
const useUserSidebarGroups = (): NavGroupProps[] => {
  const { t } = useTranslation();

  return [
    {
      title: t("sidebar.resourceAndMonitoring"),
      items: [
        {
          title: t("navigation.platformOverview"),
          url: "overview",
          icon: SquareChartGanttIcon,
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
              title: t("navigation.freeResources"),
              url: "monitor/node",
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
          title: t("navigation.myJobs"),
          icon: FlaskConicalIcon,
          items: [
            {
              title: t("navigation.customJobs"),
              url: "job/batch",
            },
            {
              title: t("navigation.jupyterLab"),
              url: "job/inter",
            },
          ],
        },
        {
          title: t("navigation.jobTemplates"),
          url: "modal",
          icon: ShoppingBagIcon,
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
              title: t("navigation.datasets"),
              url: "data/dataset",
            },
            {
              title: t("navigation.models"),
              url: "data/model",
            },
            {
              title: t("navigation.sharedFiles"),
              url: "data/sharefile",
            },
          ],
        },
      ],
    },
    {
      title: t("sidebar.others"),
      items: [
        {
          title: t("navigation.settings"),
          icon: SettingsIcon,
          items: [
            {
              title: t("navigation.userSettings"),
              url: "setting/user",
            },
          ],
        },
        {
          title: t("navigation.helpAndFeedback"),
          url: "feedback",
          icon: MessageSquareMoreIcon,
        },
      ],
    },
  ];
};

export const portalRoute: RouteObject = {
  path: "/portal",
  element: (
    <AuthedRouter>
      <DashboardLayoutWrapper />
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

// Create a wrapper component to use the hook
function DashboardLayoutWrapper() {
  const groups = useUserSidebarGroups();
  return <DashboardLayout groups={groups} />;
}
