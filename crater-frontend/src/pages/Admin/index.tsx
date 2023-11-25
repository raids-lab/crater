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
import { PersonalUser } from "./User/Personal";
import { GroupUser } from "./User/Group";

const sidebarItems: SidebarItem[] = [
  {
    path: "overview",
    icon: <BarChartIcon />,
    children: [],
    route: {
      path: "overview",
    },
  },
  {
    path: "user",
    icon: <PersonIcon />,
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
    icon: <OverviewIcon />,
    children: [
      {
        route: {
          path: "default",
        },
      },
      {
        route: {
          path: "ai",
        },
      },
      {
        route: {
          path: "dl",
        },
      },
    ],
  },
  {
    path: "image",
    icon: <LightHouseIcon />,
    children: [
      {
        route: {
          path: "list",
        },
      },
    ],
  },
  {
    path: "data",
    icon: <DatabaseIcon />,
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
    icon: <FileTextIcon />,
    route: {
      path: "docs",
      lazy: () => import("./Docs"),
    },
  },
  {
    path: "feedback",
    icon: <Pencil2Icon />,
    route: {
      path: "feedback",
      lazy: () => import("./Feedback"),
    },
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth("admin");
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

const Layout = () => {
  const [showSidebar, setShowSidebar] = useState(false);
  return (
    <>
      <div className="grid h-screen w-screen overflow-hidden bg-dashboard md:grid-cols-sidebar">
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
            <div className="px-6 py-6">
              <Outlet />
            </div>
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
    {
      index: true,
      element: <Navigate to="user" replace={true} />,
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
