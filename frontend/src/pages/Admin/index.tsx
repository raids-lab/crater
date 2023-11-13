import { Sidebar, SidebarItem, SidebarMenu } from "@/components/Sidebar";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense, useState } from "react";
import { Navigate, Outlet, RouteObject } from "react-router-dom";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import Navibar from "@/components/Navibar";
import {
  BarChartIcon,
  FileTextIcon,
  Pencil2Icon,
  PersonIcon,
} from "@radix-ui/react-icons";
import OverviewIcon from "@/components/icon/OverviewIcon";
import DatabaseIcon from "@/components/icon/DatabaseIcon";
import LightHouseIcon from "@/components/icon/LightHouseIcon";

const sidebarItems: SidebarItem[] = [
  {
    title: "资源监控",
    path: "overview",
    icon: <BarChartIcon />,
    children: [],
    route: {
      path: "overview",
    },
  },
  {
    title: "用户管理",
    path: "user",
    icon: <PersonIcon />,
    children: [],
    route: {
      path: "user",
      lazy: () => import("./User"),
    },
  },
  {
    title: "任务管理",
    path: "job",
    icon: <OverviewIcon />,
    children: [
      {
        title: "通用任务",
        route: {
          path: "default",
        },
      },
      {
        title: "AI 训练任务",
        route: {
          path: "ai",
        },
      },
      {
        title: "深度推荐训练任务",
        route: {
          path: "dl",
        },
      },
    ],
  },
  {
    title: "镜像管理",
    path: "image",
    icon: <LightHouseIcon />,
    children: [
      {
        title: "镜像列表",
        route: {
          path: "list",
        },
      },
    ],
  },
  {
    title: "数据管理",
    path: "data",
    icon: <DatabaseIcon />,
    children: [
      {
        title: "公开数据集",
        route: {
          path: "dataset",
        },
      },
    ],
  },
];

const sidebarMenus: SidebarMenu[] = [
  {
    title: "文档编辑",
    path: "docs",
    icon: <FileTextIcon />,
    route: {
      path: "docs",
      lazy: () => import("./Docs"),
    },
  },
  {
    title: "问题收集",
    path: "feedback",
    icon: <Pencil2Icon />,
    route: {
      path: "feedback",
      lazy: () => import("./Feedback"),
    },
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth("viewer");
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

const Layout = () => {
  const [showSidebar, setShowSidebar] = useState(false);
  return (
    <>
      <div className="grid h-screen w-screen overflow-hidden bg-zinc-100 dark:bg-background md:grid-cols-sidebar">
        <Sidebar
          sidebarItems={sidebarItems}
          sidebarMenus={sidebarMenus}
          closeSidebar={() => setShowSidebar(false)}
          className={cn({
            "-translate-x-full": !showSidebar,
          })}
        />
        <ScrollArea className="h-screen w-full overflow-y-hidden">
          <div className="grid w-full grid-rows-header">
            <Navibar />
            <Outlet />
          </div>
        </ScrollArea>
      </div>
      {/* When screen size is smaller than md, show a float button to open and close sidebar */}
      {/* See https://reacthustle.com/blog/next-js-tailwind-responsive-sidebar-layout*/}
      <Button
        title="Sidebar Controller"
        variant={"default"}
        className={cn(
          "fixed bottom-4 right-4 h-12 w-12 rounded-full backdrop-blur-md backdrop-filter",
          ".3s translate-x-0 transition-transform ease-in-out md:hidden",
          {
            " translate-x-20": showSidebar,
          },
        )}
        type="button"
        onClick={() => {
          setShowSidebar((prev) => !prev);
        }}
      >
        <svg
          className="h-6 w-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          {/* Bar Icon */}
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M4 6h16M4 12h8m-8 6h16"
          />
        </svg>
      </Button>
    </>
  );
};

export const adminRoute: RouteObject = {
  path: "/admin",
  element: (
    <Suspense fallback={<h2>Loading...</h2>}>
      <AuthedRouter>
        <Layout />
      </AuthedRouter>
    </Suspense>
  ),
  children: [
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
