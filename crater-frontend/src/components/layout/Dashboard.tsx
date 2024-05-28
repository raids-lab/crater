import { Sidebar, SidebarItem, SidebarMenu } from "@/components/layout/Sidebar";
import { Outlet, useLocation } from "react-router-dom";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import CraterIcon from "@/components/icon/CraterIcon";
import CraterText from "@/components/icon/CraterText";
import ProjectSwitcher from "@/components/layout/ProjectSwitcher";
import { NavBreadcrumb } from "@/components/layout/NavBreadcrumb";
import { UserDropdownMenu } from "@/components/layout/UserDropdownMenu";
import { useMemo, useState } from "react";
import SidebarSheet from "./SidebarSheet";
import { PanelLeft } from "lucide-react";
import { useRecoilValue } from "recoil";
import { globalAccount } from "@/utils/store";
import { Role } from "@/services/api/auth";

const DashboardLayout = ({
  sidebarItems: items,
  sidebarMenus,
}: {
  sidebarItems: SidebarItem[];
  sidebarMenus: SidebarMenu[];
}) => {
  const { pathname: rawPath } = useLocation();
  const [open, setOpen] = useState(false);
  const currentAccount = useRecoilValue(globalAccount);

  // 特殊规则，网盘路由切换时，不启用过渡动画
  const motionKey = useMemo(() => {
    // begins with /portal/share/data
    if (rawPath.startsWith("/portal/data/filesystem")) {
      return "/portal/data/filesystem";
    }
    return rawPath;
  }, [rawPath]);

  // 特殊规则，portal 个人项目或者非管理员角色，隐藏项目管理菜单
  const sidebarItems = useMemo(() => {
    if (rawPath.startsWith("/portal") && currentAccount?.role !== Role.Admin) {
      return items.filter((item) => item.path !== "project");
    }
    return items;
  }, [items, currentAccount, rawPath]);

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
            <span className="sr-only">Crater</span>
          </Link>
          <Sidebar sidebarItems={sidebarItems} sidebarMenus={sidebarMenus} />
        </nav>
      </aside>
      <div className="flex flex-col md:gap-4 md:py-4 md:pl-14 xl:pl-48">
        <header className="sticky top-0 z-30 flex h-14 items-center gap-4 border-b bg-background px-4 md:static md:h-auto md:border-0 md:bg-transparent md:px-8">
          <Sheet open={open} onOpenChange={setOpen}>
            <SheetTrigger asChild>
              <Button size="icon" variant="outline" className="md:hidden">
                <PanelLeft className="h-5 w-5" />
                <span className="sr-only">Toggle Menu</span>
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="md:max-w-xs">
              <Link
                to="/portal"
                className="group flex h-10 w-full shrink-0 items-center justify-center gap-1"
              >
                <CraterIcon className="flex h-7 w-7 group-hover:scale-110" />
                <CraterText className="h-3.5 xl:flex" />
                <span className="sr-only">Crater</span>
              </Link>
              <SidebarSheet
                sidebarItems={sidebarItems}
                sidebarMenus={sidebarMenus}
                closeSidebar={() => setOpen(false)}
              />
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
          <main className="grid flex-1 items-start gap-4 p-4 md:gap-x-6 md:gap-y-8 md:px-8 md:py-0 lg:grid-cols-3">
            <Outlet />
          </main>
        </motion.div>
      </div>
    </div>
  );
};

export default DashboardLayout;
