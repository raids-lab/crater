import { SidebarItem, SidebarMenu } from "@/components/layout/Sidebar";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense } from "react";
import { Navigate, RouteObject } from "react-router-dom";
import { GroupUser } from "./Queue/Group";
import { Role } from "@/services/api/auth";
import DashboardLayout from "@/components/layout/Dashboard";
import { User } from "./User";
// import Volcano from "./Job/Volcano";
import Resource from "./Cluster/Resource";
import UserProjectManagement from "./Queue/ProjectUser";
import {
  ContainerIcon,
  DatabaseIcon,
  FileTextIcon,
  FlaskConicalIcon,
  MessageSquareMoreIcon,
  ServerIcon,
  SettingsIcon,
  UserRoundIcon,
  UsersRoundIcon,
} from "lucide-react";

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
    path: "account",
    icon: UsersRoundIcon,
    route: {
      path: "account/*",
      children: [
        {
          index: true,
          element: <GroupUser />,
        },
        {
          path: ":id",
          element: <UserProjectManagement />,
        },
      ],
    },
    children: [],
  },
  {
    path: "user",
    icon: UserRoundIcon,
    children: [],
    route: {
      path: "user",
      element: <User />,
    },
  },
  {
    path: "job",
    icon: FlaskConicalIcon,
    route: {
      path: "job/*",
      // element: <Volcano />,
      lazy: () => import("./Job/Volcano"),
    },
    children: [],
  },
  {
    path: "image",
    icon: ContainerIcon,
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
          path: "queue/*",
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
          lazy: () => import("./Data/Dataset"),
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
    icon: MessageSquareMoreIcon,
    route: {
      path: "feedback",
    },
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth(Role.Admin);
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
