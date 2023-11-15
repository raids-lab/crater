import { Sidebar, SidebarItem, SidebarMenu } from "@/components/Sidebar";
import DatabaseIcon from "@/components/icon/DatabaseIcon";
import OverviewIcon from "@/components/icon/OverviewIcon";
import CodeOneIcon from "@/components/icon/CodeOneIcon";
import LightHouseIcon from "@/components/icon/LightHouseIcon";
import WorkBenchIcon from "@/components/icon/WorkBenchIcon";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense, useState } from "react";
import { Navigate, Outlet, RouteObject } from "react-router-dom";
import { FileTextIcon, Pencil2Icon } from "@radix-ui/react-icons";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import Navibar from "@/components/Navibar";

const sidebarItems: SidebarItem[] = [
  {
    title: "概览",
    path: "overview",
    icon: <WorkBenchIcon />,
    children: [],
    route: {
      path: "overview",
      lazy: () => import("./Overview"),
    },
  },
  {
    title: "Jupyter 管理",
    path: "jupyter",
    icon: <CodeOneIcon />,
    children: [],
    route: {
      path: "jupyter",
      lazy: () => import("./Jupyter"),
    },
  },
  {
    title: "任务管理",
    path: "job",
    icon: <OverviewIcon />,
    children: [
      {
        title: "AI 训练任务",
        route: {
          path: "ai",
          lazy: () => import("./Job/Ai"),
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
    title: "数据管理",
    path: "data",
    icon: <DatabaseIcon />,
    children: [
      {
        title: "共享空间",
        route: {
          path: "share",
          lazy: () => import("./Data/Share"),
        },
      },
      {
        title: "代码",
        route: {
          path: "code",
          lazy: () => import("./Data/Code"),
        },
      },
      {
        title: "其他",
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
    title: "使用文档",
    path: "docs",
    icon: <FileTextIcon />,
    route: {
      path: "docs",
      lazy: () => import("./Docs"),
    },
  },
  {
    title: "问题反馈",
    path: "feedback",
    icon: <Pencil2Icon />,
    route: {
      path: "feedback",
      lazy: () => import("./Feedback"),
    },
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth("user");
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

export const portalRoute: RouteObject = {
  path: "/portal",
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
      element: <Navigate to="job/ai" replace />,
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
