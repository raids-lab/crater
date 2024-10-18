import { useMemo, type FC } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  // DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import { Button } from "../ui/button";
import { useAtomValue, useSetAtom } from "jotai";
import {
  globalAccount,
  globalLastView,
  globalUserInfo,
  useResetStore,
} from "@/utils/store";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTheme } from "@/utils/theme";
import { toast } from "sonner";
import {
  ContactRound,
  LogOutIcon,
  MoonIcon,
  SettingsIcon,
  SunIcon,
  UserRound,
} from "lucide-react";
import { Role } from "@/services/api/auth";

export const UserDropdownMenu: FC = () => {
  const queryClient = useQueryClient();
  const { resetAll } = useResetStore();
  const userInfo = useAtomValue(globalUserInfo);
  const account = useAtomValue(globalAccount);
  const navigate = useNavigate();
  const location = useLocation();
  const { theme, setTheme } = useTheme();
  const setLastView = useSetAtom(globalLastView);

  const pathParts = useMemo(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    return pathParts;
  }, [location]);

  const isAdminView = useMemo(() => {
    return pathParts[0] === "admin";
  }, [pathParts]);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="icon"
          className="overflow-hidden rounded-full"
        >
          <UserRound className="h-5 w-5" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-44" align="end" forceMount>
        <DropdownMenuLabel className="font-normal">
          <div className="flex flex-col space-y-1">
            <p className="text-sm font-medium leading-none">
              {userInfo.nickname}
            </p>
            <p className="overflow-clip text-ellipsis text-xs text-muted-foreground">
              {userInfo.email}
            </p>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {account.rolePlatform === Role.Admin && (
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
            <ContactRound className="mr-2 h-4 w-4" />
            <span>切换为{isAdminView ? "普通用户" : "管理员"}</span>
          </DropdownMenuItem>
        )}
        <DropdownMenuItem asChild>
          <Link to="/portal/setting">
            <SettingsIcon className="mr-2 h-4 w-4" />
            系统设置
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
        >
          {theme === "light" ? (
            <MoonIcon className="mr-2 h-4 w-4" />
          ) : (
            <SunIcon className="mr-2 h-4 w-4" />
          )}
          {theme === "light" ? "深色模式" : "浅色模式"}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className="focus:bg-destructive focus:text-destructive-foreground"
          onClick={() => {
            setLastView(pathParts[0]);
            queryClient.clear();
            resetAll();
            toast.success("已退出");
          }}
        >
          <LogOutIcon className="mr-2 h-4 w-4" />
          <span>退出登录</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
