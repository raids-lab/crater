import { Outlet, useLocation } from "react-router-dom";
import { motion } from "framer-motion";
import { NavBreadcrumb } from "@/components/layout/NavBreadcrumb";
import { useMemo } from "react";

import { AppSidebar } from "@/components/sidebar/app-sidebar";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { NavGroupProps } from "../sidebar/types";
import { useAtomValue } from "jotai";
import { globalFixedLayout, globalSettings } from "@/utils/store";
import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";
import { CogIcon } from "lucide-react";

const DashboardLayout = ({ groups }: { groups: NavGroupProps[] }) => {
  const { pathname: rawPath } = useLocation();
  const fixedLayout = useAtomValue(globalFixedLayout);
  const scheduler = useAtomValue(globalSettings).scheduler;

  // 特殊规则，网盘路由切换时，不启用过渡动画
  const motionKey = useMemo(() => {
    // begins with /portal/share/data
    if (rawPath.startsWith("/portal/data/filesystem/")) {
      return "/portal/data/filesystem/";
    }
    return rawPath;
  }, [rawPath]);

  return (
    <SidebarProvider>
      <AppSidebar groups={groups} />
      <SidebarInset>
        <header
          className={cn(
            "flex h-16 shrink-0 items-center justify-between gap-2 px-6 transition-[width,height] ease-linear",
            // "group-has-data-[collapsible=icon]/sidebar-wrapper:h-16",
            fixedLayout &&
              "header-fixed peer/header fixed z-50 w-[inherit] rounded-md",
          )}
        >
          <div className="flex items-center gap-2">
            <SidebarTrigger className="-ml-1" />
            <NavBreadcrumb className="hidden md:flex" />
          </div>
          {scheduler !== "volcano" && (
            <Badge variant="secondary" className="uppercase">
              <CogIcon />
              {scheduler}
            </Badge>
          )}
        </header>
        <motion.div
          key={motionKey}
          initial={{ opacity: 0, y: "3vh" }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ type: "spring", duration: 1.2 }}
          className={cn(
            "flex flex-col gap-6 p-6 pt-0",
            fixedLayout &&
              "absolute top-0 right-0 bottom-0 left-0 w-full grow overflow-hidden peer-[.header-fixed]/header:mt-16",
          )}
        >
          <Outlet />
        </motion.div>
      </SidebarInset>
    </SidebarProvider>
  );
};

export default DashboardLayout;
