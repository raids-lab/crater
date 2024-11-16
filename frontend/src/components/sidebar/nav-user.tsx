import {
  BadgeCheck,
  ChevronsUpDown,
  LogOut,
  Moon,
  SettingsIcon,
  Sparkles,
  Sun,
} from "lucide-react";
import Identicon from "@polkadot/react-identicon";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { useAtomValue, useSetAtom } from "jotai";
import {
  globalAccount,
  globalLastView,
  globalUserInfo,
  useResetStore,
} from "@/utils/store";
import { Link, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTheme } from "@/utils/theme";
import { toast } from "sonner";
import { Role } from "@/services/api/auth";
import { stringToSS58 } from "@/utils/ss58";
import { useIsAdmin } from "@/hooks/useAdmin";

export function NavUser() {
  const { isMobile } = useSidebar();
  const queryClient = useQueryClient();
  const { resetAll } = useResetStore();
  const user = useAtomValue(globalUserInfo);
  const account = useAtomValue(globalAccount);
  const navigate = useNavigate();
  const { theme, setTheme } = useTheme();
  const setLastView = useSetAtom(globalLastView);
  const isAdminView = useIsAdmin();

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <Avatar className="h-8 w-8">
                <AvatarImage src={user.avatar} alt="Avatar preview" />
                <AvatarFallback>
                  <Identicon
                    value={stringToSS58(user.name)}
                    size={32}
                    theme="beachball"
                    className="!cursor-default"
                  />
                </AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">{user.nickname}</span>
                <span className="truncate text-xs">{user.email}</span>
              </div>
              <ChevronsUpDown className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <Avatar className="h-8 w-8">
                  <AvatarImage src={user.avatar} alt="Avatar preview" />
                  <AvatarFallback>
                    <Identicon
                      value={stringToSS58(user.name)}
                      size={32}
                      theme="beachball"
                      className="!cursor-default"
                    />
                  </AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold">
                    {user.nickname}
                  </span>
                  <span className="truncate text-xs">{user.email}</span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            {account.rolePlatform === Role.Admin && (
              <>
                <DropdownMenuGroup>
                  <DropdownMenuItem
                    onClick={() => {
                      if (isAdminView) {
                        setLastView("portal");
                        navigate("/portal");
                        toast.success("切换至用户视图");
                      } else {
                        setLastView("admin");
                        navigate("/admin");
                        toast.success("切换至管理员视图");
                      }
                    }}
                  >
                    <Sparkles />
                    切换为{isAdminView ? "普通用户" : "管理员"}
                  </DropdownMenuItem>
                </DropdownMenuGroup>
                <DropdownMenuSeparator />
              </>
            )}
            <DropdownMenuGroup>
              <DropdownMenuItem asChild>
                <Link to="/portal/setting">
                  <BadgeCheck />
                  个人主页
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem asChild>
                <Link to="/portal/setting">
                  <SettingsIcon />
                  系统设置
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
              >
                {theme === "light" ? <Moon /> : <Sun />}
                {theme === "light" ? "深色模式" : "浅色模式"}
              </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="focus:bg-destructive focus:text-destructive-foreground"
              onClick={() => {
                setLastView(isAdminView ? "admin" : "portal");
                queryClient.clear();
                resetAll();
                toast.success("已退出");
              }}
            >
              <LogOut />
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
