import { SidebarItem, SidebarMenu } from "@/components/layout/Sidebar";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren } from "react";
import { Navigate, RouteObject } from "react-router-dom";
import { Role } from "@/services/api/auth";
import DashboardLayout from "@/components/layout/Dashboard";
import {
  ArchiveIcon,
  BarChartBigIcon,
  BriefcaseIcon,
  ContainerIcon,
  DatabaseIcon,
  FileTextIcon,
  FlaskConicalIcon,
  MessageSquareMoreIcon,
} from "lucide-react";

const sidebarItems: SidebarItem[] = [
  {
    path: "overview",
    icon: BarChartBigIcon,
    children: [],
    route: {
      path: "overview",
      lazy: () => import("./Overview"),
    },
  },
  {
    path: "job",
    icon: FlaskConicalIcon,
    children: [
      {
        route: {
          path: "ai/*",
          lazy: () => import("./Job/Ai"),
        },
      },
      {
        route: {
          path: "jupyter/*",
          lazy: () => import("./Job/Jupyter"),
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
    icon: ContainerIcon,
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
          lazy: () => import("./Data/Share"),
        },
      },
      {
        route: {
          path: "dataset",
        },
      },
    ],
  },
  {
    path: "project",
    icon: ArchiveIcon,
    children: [
      {
        route: {
          path: "member",
        },
      },
      {
        route: {
          path: "config",
        },
      },
    ],
  },
];

const sidebarMenus: SidebarMenu[] = [
  {
    path: "docs",
    icon: FileTextIcon,
    route: {
      path: "docs",
      lazy: () => import("./Docs"),
    },
  },
  {
    path: "feedback",
    icon: MessageSquareMoreIcon,
    route: {
      path: "feedback",
      lazy: () => import("./Feedback"),
    },
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth(Role.User);
  // const isAuthenticated = true;
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

export const portalRoute: RouteObject = {
  path: "/portal",
  element: (
    <AuthedRouter>
      <DashboardLayout
        sidebarItems={sidebarItems}
        sidebarMenus={sidebarMenus}
      />
    </AuthedRouter>
  ),
  children: [
    {
      index: true,
      element: <Navigate to="job/jupyter" replace />,
    },
    ...sidebarItems.map((item) => {
      return (
        item.route ?? {
          path: item.path,
          children: item.children.map((child) => child.route),
        }
      );
    }),
    ...sidebarMenus.map((item) => item.route),
    {
      path: "*",
      element: <h1>404</h1>,
    },
  ],
};
