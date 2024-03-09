import { SidebarItem, SidebarMenu } from "@/components/layout/Sidebar";
import DatabaseIcon from "@/components/icon/DatabaseIcon";
import OverviewIcon from "@/components/icon/OverviewIcon";
import LightHouseIcon from "@/components/icon/LightHouseIcon";
import WorkBenchIcon from "@/components/icon/WorkBenchIcon";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense } from "react";
import { Navigate, RouteObject } from "react-router-dom";
import { FileTextIcon, Pencil2Icon } from "@radix-ui/react-icons";
import { DashboardLayout } from "../Portal";

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
          path: "dl/*",
          lazy: () => import("./Job/Dl"),
        },
      },
    ],
  },
  {
    path: "image",
    icon: LightHouseIcon,
    children: [
      {
        route: {
          path: "list",
          lazy: () => import("../Portal/Image/List"),
        },
      },
      {
        route: {
          path: "make",
          lazy: () => import("../Portal/Image/Make"),
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
          path: "dataset",
          lazy: () => import("./Data/Dataset"),
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
      lazy: () => import("../Portal/Docs"),
    },
  },
  {
    path: "feedback",
    icon: Pencil2Icon,
    route: {
      path: "feedback",
      lazy: () => import("../Portal/Feedback"),
    },
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth("user");
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

export const recommendRoute: RouteObject = {
  path: "/recommend",
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
      element: <Navigate to="job/dl" replace />,
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
