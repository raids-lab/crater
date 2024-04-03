import { SidebarItem, SidebarMenu } from "@/components/layout/Sidebar";
import DatabaseIcon from "@/components/icon/DatabaseIcon";
import OverviewIcon from "@/components/icon/OverviewIcon";
import LightHouseIcon from "@/components/icon/LightHouseIcon";
import WorkBenchIcon from "@/components/icon/WorkBenchIcon";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense } from "react";
import { Navigate, RouteObject } from "react-router-dom";
import { FileTextIcon, Pencil2Icon } from "@radix-ui/react-icons";
import { Role } from "@/services/api/auth";
import DashboardLayout from "@/components/layout/Dashboard";

const sidebarItems: SidebarItem[] = [
  {
    path: "overview",
    icon: WorkBenchIcon,
    children: [],
    route: {
      path: "overview",
      lazy: () => import("./Overview"),
    },
  },
  {
    path: "job",
    icon: OverviewIcon,
    children: [
      {
        route: {
          path: "jupyter",
          lazy: () => import("./Job/Jupyter"),
        },
      },
      {
        route: {
          path: "ai/*",
          lazy: () => import("./Job/Ai"),
        },
      },
    ],
  },
  {
    path: "image",
    icon: LightHouseIcon,
    children: [],
    route: {
      path: "image",
      lazy: () => import("./Image/List"),
    },
  },
  {
    path: "data",
    icon: DatabaseIcon,
    children: [
      {
        route: {
          path: "share/*",
          lazy: () => import("./Data/Share"),
        },
      },
      {
        route: {
          path: "other",
          lazy: () => import("./Data/Other"),
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
    icon: Pencil2Icon,
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
    <Suspense fallback={<h2>Loading...</h2>}>
      <AuthedRouter>
        <DashboardLayout
          sidebarItems={sidebarItems}
          sidebarMenus={sidebarMenus}
        />
      </AuthedRouter>
    </Suspense>
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
