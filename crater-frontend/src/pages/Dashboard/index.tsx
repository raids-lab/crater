import { Sidebar, SidebarItem } from "@/components/Sidebar";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense } from "react";
import { Navigate, Outlet, RouteObject } from "react-router-dom";

const items: SidebarItem[] = [
  {
    title: "任务管理",
    path: "job",
    children: [
      {
        title: "通用任务",
        route: {
          path: "default",
          lazy: () => import("./Job/Default"),
        },
      },
      {
        title: "AI 训练任务",
        route: {
          path: "ai",
          lazy: () => import("./Job/Ai"),
        },
      },
      {
        title: "深度推荐训练任务",
        route: {
          path: "dl",
          lazy: () => import("./Job/Dl"),
        },
      },
    ],
  },
  {
    title: "数据管理",
    path: "data",
    children: [
      {
        title: "个人空间",
        route: {
          path: "personal",
          lazy: () => import("./Data/Personal"),
        },
      },
      {
        title: "数据集",
        route: {
          path: "dataset",
          lazy: () => import("./Data/Dataset"),
        },
      },
    ],
  },
  {
    title: "代码管理",
    path: "code",
    children: [
      {
        title: "关联 Gitlab 仓库",
        route: {
          path: "gitlab",
          lazy: () => import("./Code/Gitlab"),
        },
      },
    ],
  },
  {
    title: "镜像管理",
    path: "image",
    children: [
      {
        title: "镜像列表",
        route: {
          path: "list",
          lazy: () => import("./Image/List"),
        },
      },
      {
        title: "镜像制作",
        route: {
          path: "make",
          lazy: () => import("./Image/Make"),
        },
      },
    ],
  },
  {
    title: "集群监控",
    path: "cluster",
    children: [
      {
        title: "节点监控",
        route: {
          path: "node",
          lazy: () => import("./Cluster/Node"),
        },
      },
      {
        title: "任务监控",
        route: {
          path: "task",
          lazy: () => import("./Cluster/Task"),
        },
      },
    ],
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth("user");
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

export const dashboardRoute: RouteObject = {
  path: "/dashboard",
  element: (
    <Suspense fallback={<h2>Loading...</h2>}>
      <AuthedRouter>
        <div className="grid overflow-hidden lg:grid-cols-6">
          <Sidebar sidebarItems={items} className="hidden lg:block" />
          <div className="col-span-4 h-screen overflow-auto px-6 py-6 lg:col-span-5">
            <Outlet />
          </div>
        </div>
      </AuthedRouter>
    </Suspense>
  ),
  children: [
    {
      index: true,
      lazy: () => import("./Job/Ai"),
    },
    ...items.map((item) => ({
      path: item.path,
      children: item.children.map((child) => child.route),
    })),
    {
      path: "*",
      element: <h1>404</h1>,
    },
  ],
};
