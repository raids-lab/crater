import { SidebarItem, SidebarMenu } from "@/components/layout/Sidebar";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense } from "react";
import { Navigate, RouteObject } from "react-router-dom";
import { FileTextIcon, Pencil2Icon, PersonIcon } from "@radix-ui/react-icons";
import OverviewIcon from "@/components/icon/OverviewIcon";
import DatabaseIcon from "@/components/icon/DatabaseIcon";
import LightHouseIcon from "@/components/icon/LightHouseIcon";
import { PersonalUser } from "./User/Personal";
import { GroupUser } from "./User/Group";
import Jupyter from "./Job/Jupyter";
import Training from "./Job/Training";
import { DashboardLayout } from "../Portal";
import ServerIcon from "@/components/icon/ServerIcon";

const sidebarItems: SidebarItem[] = [
  {
    path: "cluster",
    icon: ServerIcon,
    children: [
      {
        route: {
          path: "node/*",
          lazy: () => import("./Cluster/Node"),
        },
      },
      {
        route: {
          path: "gpu",
          lazy: () => import("./Cluster/Gpu"),
        },
      },
    ],
  },
  {
    path: "user",
    icon: PersonIcon,
    children: [
      {
        route: {
          path: "personal",
          element: <PersonalUser />,
        },
      },
      {
        route: {
          path: "group",
          element: <GroupUser />,
        },
      },
    ],
  },
  {
    path: "job",
    icon: OverviewIcon,
    children: [
      {
        route: {
          path: "jupyter",
          element: <Jupyter />,
        },
      },
      {
        route: {
          path: "ai",
          element: <Training />,
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
          path: "public",
          lazy: () => import("./Image/Public"),
        },
      },
      {
        route: {
          path: "personal",
          lazy: () => import("./Image/Personal"),
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
    },
  },
  {
    path: "feedback",
    icon: Pencil2Icon,
    route: {
      path: "feedback",
    },
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth("admin");
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

export const adminRoute: RouteObject = {
  path: "/admin",
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
      element: <Navigate to="cluster/node" replace={true} />,
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
