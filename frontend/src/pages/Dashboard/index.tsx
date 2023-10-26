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
      <div className="col-span-4 h-screen overflow-auto lg:col-span-5">
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
      lazy: () => import("./Cluster"),
    },
    {
      path: "cluster",
      lazy: () => import("./Cluster"),
    },
    {
      path: "task",
      lazy: () => import("./Training"),
    },
    {
      path: "ai_task",
      lazy: () => import("./Training"),
    },
  ],
};

const items: SidebarItem[] = [
  {
    title: "训练任务管理",
    path: "task",
    children: [
      { title: "通用任务", path: "list" },
      { title: "AI训练任务", path: "new" },
      { title: "深度推荐训练任务", path: "monitor" },
    ],
  },
  {
    title: "集群资源查看",
    path: "cluster",
    children: [{ title: "PVC", path: "pvc" }],
  },
  {
    title: "资源配额",
    path: "resource",
    children: [
      { title: "训练任务列表", path: "list" },
      { title: "新建训练任务", path: "new" },
      { title: "任务监控", path: "monitor" },
    ],
  },
  {
    title: "Jupyter 管理",
    path: "jupyter",
    children: [
      { title: "训练任务列表", path: "list" },
      { title: "新建训练任务", path: "new" },
      { title: "任务监控", path: "monitor" },
    ],
  },
];
