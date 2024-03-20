import { Sidebar, SidebarItem, SidebarMenu } from "@/components/layout/Sidebar";
import DatabaseIcon from "@/components/icon/DatabaseIcon";
import OverviewIcon from "@/components/icon/OverviewIcon";
import LightHouseIcon from "@/components/icon/LightHouseIcon";
import WorkBenchIcon from "@/components/icon/WorkBenchIcon";
import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren, Suspense } from "react";
import { Navigate, Outlet, RouteObject, useLocation } from "react-router-dom";
import { FileTextIcon, Pencil2Icon } from "@radix-ui/react-icons";
import { cn } from "@/lib/utils";
import Navibar from "@/components/layout/Navibar";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { useRecoilState } from "recoil";
import { globalSidebarMini } from "@/utils/store";
import { motion } from "framer-motion";

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
          path: "jupyter",
          lazy: () => import("./Job/Jupyter"),
        },
      },
      {
        route: {
          path: "ai/*",
          lazy: () => import("./Job/Ai"),
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
          lazy: () => import("./Image/List"),
        },
      },
      {
        route: {
          path: "make",
          lazy: () => import("./Image/Make"),
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
          path: "share",
          lazy: () => import("./Data/Share"),
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
      lazy: () => import("./Docs"),
    },
  },
  {
    path: "feedback",
    icon: Pencil2Icon,
    route: {
      path: "feedback",
      lazy: () => import("./Feedback"),
    },
  },
];

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth("user");
  // const isAuthenticated = true;
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

export const DashboardLayout = ({
  sidebarItems,
  sidebarMenus,
}: {
  sidebarItems: SidebarItem[];
  sidebarMenus: SidebarMenu[];
}) => {
  const [isMinimized, setIsMinimized] = useRecoilState(globalSidebarMini);
  const { pathname } = useLocation();

  return (
    <>
      <div className="relative h-screen w-screen overflow-hidden">
        <div
          className={cn("absolute bottom-0 left-0 top-0 z-10 w-[200px]", {
            "w-14": isMinimized,
          })}
        >
          <Sidebar
            sidebarItems={sidebarItems}
            sidebarMenus={sidebarMenus}
            isMinimized={isMinimized}
            toggleIsMinimized={() => setIsMinimized((prev) => !prev)}
          />
        </div>
        <div
          className={cn(
            "absolute bottom-0 right-0 top-0 w-[calc(100vw_-_200px)]",
            {
              "w-[calc(100vw_-_56px)]": isMinimized,
            },
          )}
        >
          <ScrollArea
            className={cn("h-screen w-[calc(100vw_-_200px)]", {
              "w-[calc(100vw_-_56px)]": isMinimized,
            })}
          >
            <div className="grid w-full grid-rows-header px-6">
              <Navibar />
              <div className="py-6">
                <motion.div
                  key={pathname}
                  initial={{ opacity: 0, y: "3vh" }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ type: "spring", duration: 1.2 }}
                >
                  <Outlet />
                </motion.div>
              </div>
            </div>
            <ScrollBar orientation="horizontal" />
          </ScrollArea>
        </div>
      </div>
    </>
  );
};

export const portalRoute: RouteObject = {
  path: "/portal",
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
      element: <Navigate to="job/jupyter" replace />,
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
