import { Sidebar, SidebarItem, SidebarMenu } from "@/components/layout/Sidebar";
import { Outlet, useLocation } from "react-router-dom";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import {
  Home,
  LineChart,
  Package,
  Package2,
  PanelLeft,
  ShoppingCart,
  Users2,
} from "lucide-react";

import { Button } from "@/components/ui/button";

import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";

import CraterIcon from "@/components/icon/CraterIcon";
import CraterText from "@/components/icon/CraterText";
import ProjectSwitcher from "@/components/layout/ProjectSwitcher";
import { NavBreadcrumb } from "@/components/layout/NavBreadcrumb";
import { UserDropdownMenu } from "@/components/layout/UserDropdownMenu";
import { useMemo } from "react";

const DashboardLayout = ({
  sidebarItems,
  sidebarMenus,
}: {
  sidebarItems: SidebarItem[];
  sidebarMenus: SidebarMenu[];
}) => {
  const { pathname: rawPath } = useLocation();

  // 特殊规则，网盘路由切换时，不启用过渡动画
  const motionKey = useMemo(() => {
    // begins with /portal/share/data
    if (rawPath.startsWith("/portal/data/share")) {
      return "/portal/data/share";
    }
    return rawPath;
  }, [rawPath]);

  return (
    <div className="flex min-h-screen w-full flex-col bg-muted/40">
      <aside className="fixed inset-y-0 left-0 z-10 hidden w-14 flex-col border-r bg-background shadow md:flex xl:w-48">
        <nav className="flex flex-col items-center gap-3 md:py-5">
          <Link
            to="/portal"
            className="group flex h-7 w-full shrink-0 items-center justify-center gap-1"
          >
            <CraterIcon className="flex h-7 w-7 group-hover:scale-110" />
            <CraterText className="hidden h-3.5 xl:flex" />
            <span className="sr-only">Acme Inc</span>
          </Link>
          <Sidebar sidebarItems={sidebarItems} sidebarMenus={sidebarMenus} />
        </nav>
      </aside>
      <div className="flex flex-col md:gap-4 md:py-4 md:pl-14 xl:pl-48">
        <header className="sticky top-0 z-30 flex h-14 items-center gap-4 border-b bg-background px-4 md:static md:h-auto md:border-0 md:bg-transparent md:px-8">
          <Sheet>
            <SheetTrigger asChild>
              <Button size="icon" variant="outline" className="md:hidden">
                <PanelLeft className="h-5 w-5" />
                <span className="sr-only">Toggle Menu</span>
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="md:max-w-xs">
              <nav className="grid gap-6 text-lg font-medium">
                <Link
                  to="#"
                  className="group flex h-10 w-10 shrink-0 items-center justify-center gap-2 rounded-full bg-primary text-lg font-semibold text-primary-foreground md:text-base"
                >
                  <Package2 className="h-5 w-5 transition-all group-hover:scale-110" />
                  <span className="sr-only">Acme Inc</span>
                </Link>
                <Link
                  to="#"
                  className="flex items-center gap-4 px-2.5 text-muted-foreground hover:text-foreground"
                >
                  <Home className="h-5 w-5" />
                  Dashboard
                </Link>
                <Link
                  to="#"
                  className="flex items-center gap-4 px-2.5 text-foreground"
                >
                  <ShoppingCart className="h-5 w-5" />
                  Orders
                </Link>
                <Link
                  to="#"
                  className="flex items-center gap-4 px-2.5 text-muted-foreground hover:text-foreground"
                >
                  <Package className="h-5 w-5" />
                  Products
                </Link>
                <Link
                  to="#"
                  className="flex items-center gap-4 px-2.5 text-muted-foreground hover:text-foreground"
                >
                  <Users2 className="h-5 w-5" />
                  Customers
                </Link>
                <Link
                  to="#"
                  className="flex items-center gap-4 px-2.5 text-muted-foreground hover:text-foreground"
                >
                  <LineChart className="h-5 w-5" />
                  Settings
                </Link>
              </nav>
            </SheetContent>
          </Sheet>
          <NavBreadcrumb className="hidden md:flex" />
          {/* <div className="relative ml-auto flex-1 md:grow-0">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                type="search"
                placeholder="Search..."
                className="w-full rounded-lg bg-background pl-8 md:w-[200px] lg:w-[336px]"
              />
            </div> */}
          <ProjectSwitcher className="relative ml-auto flex-1 md:grow-0" />
          <UserDropdownMenu />
        </header>
        <motion.div
          key={motionKey}
          initial={{ opacity: 0, y: "3vh" }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ type: "spring", duration: 1.2 }}
        >
          <main className="grid flex-1 items-start gap-4 p-4 md:gap-8 md:px-8 md:py-0 lg:grid-cols-3">
            <Outlet />
          </main>
        </motion.div>
      </div>
    </div>
  );
};

export default DashboardLayout;
