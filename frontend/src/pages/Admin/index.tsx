import { SidebarItem, SidebarMenu } from "@/components/layout/Sidebar";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense } from "react";
import { Navigate, RouteObject } from "react-router-dom";
import { Role } from "@/services/api/auth";
import DashboardLayout from "@/components/layout/Dashboard";
import { User } from "./User";
// import Volcano from "./Job/Volcano";
import Resource from "./Cluster/Resource";
import UserProjectManagement from "./Account/ProjectUser";
import {
  BookOpenIcon,
  BoxIcon,
  DatabaseIcon,
  FileTextIcon,
  FlaskConicalIcon,
  MessageSquareMoreIcon,
  ServerIcon,
  UserRoundIcon,
  UsersRoundIcon,
} from "lucide-react";
import { Account } from "./Account";
import admindatasetRoutes from "./Data";
import { NavGroupProps } from "@/components/sidebar/nav-main";

const sidebarItems: SidebarItem[] = [
  {
    path: "cluster",
    icon: ServerIcon,
    children: [
      {
        route: {
          path: "node",
          element: <h1>WIP</h1>,
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
          element: <Account />,
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
    icon: BoxIcon,
    children: [
      {
        route: {
          path: "create",
          lazy: () => import("./Image/Create"),
        },
      },
      {
        route: {
          path: "upload",
          lazy: () => import("./Image/Upload"),
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
          children: admindatasetRoutes,
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
    icon: MessageSquareMoreIcon,
    route: {
      path: "feedback",
    },
  },
];

const adminSidebarGroups: NavGroupProps[] = [
  {
    title: "作业与服务",
    items: [
      {
        title: "集群管理",
        isActive: true,
        url: "cluster",
        icon: ServerIcon,
        items: [
          {
            title: "节点信息",
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
        title: "作业管理",
        url: "job",
        icon: FlaskConicalIcon,
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
        url: "image",
        icon: BoxIcon,
        items: [
          {
            title: "公共镜像",
            url: "image/public",
          },
          {
            title: "私有镜像",
            url: "image/personal",
          },
        ],
      },
      {
        title: "数据管理",
        url: "data",
        icon: DatabaseIcon,
        items: [
          {
            title: "用户文件",
            url: "data/user",
          },
          {
            title: "账户文件",
            url: "data/queue",
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
        title: "文档编辑",
        url: "docs",
        icon: BookOpenIcon,
      },
      {
        title: "问题反馈",
        icon: MessageSquareMoreIcon,
        url: "feedback",
      },
    ],
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
        <DashboardLayout groups={adminSidebarGroups} />
      </AuthedRouter>
    </Suspense>
  ),
  children: [
    {
      index: true,
      element: <Navigate to="account" replace={true} />,
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
