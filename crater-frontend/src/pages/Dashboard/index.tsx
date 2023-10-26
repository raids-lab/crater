import { Sidebar, SidebarItem } from "@/components/Sidebar";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense } from "react";
import { Navigate, Outlet, RouteObject } from "react-router-dom";

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth("user");
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

const Layout = () => {
  return (
    <div className="grid overflow-hidden lg:grid-cols-6">
      <Sidebar sidebarItems={items} className="hidden lg:block" />
      <div className="col-span-4 h-screen overflow-auto px-6 py-6 lg:col-span-5">
        <Outlet />
      </div>
    </div>
  );
};

export const dashboardRoute: RouteObject = {
  path: "/dashboard",
  element: (
    <Suspense fallback={<h2>Loading...</h2>}>
      <AuthedRouter>
        <Layout />
      </AuthedRouter>
    </Suspense>
  ),
  children: [
    {
      index: true,
      lazy: () => import("./Training/List"),
    },
    {
      path: "training",
      children: [{ path: "list", lazy: () => import("./Training/List") }],
    },
    {
      path: "dl_training",
      children: [
        { path: "list", lazy: () => import("./DlTraining/List") },
        { path: "scheduler", lazy: () => import("./DlTraining/Scheduler") },
      ],
    },
    {
      path: "cluster",
      children: [{ path: "pvc", lazy: () => import("./Cluster/Pvc") }],
    },
  ],
};

const items: SidebarItem[] = [
  {
    title: "AI 训练任务",
    path: "training",
    children: [{ title: "任务列表", path: "list" }],
  },
  {
    title: "深度推荐训练任务",
    path: "dl_training",
    children: [
      { title: "任务列表", path: "list" },
      { title: "调度结果", path: "scheduler" },
    ],
  },
  {
    title: "集群资源查看",
    path: "cluster",
    children: [{ title: "Persistent Volume", path: "pvc" }],
  },
  {
    title: "Jupyter 管理",
    path: "jupyter",
    children: [],
  },
  {
    title: "数据集管理",
    path: "dataset",
    children: [],
  },
];
