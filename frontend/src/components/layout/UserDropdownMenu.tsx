import { useMemo, type FC } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  // DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import { Button } from "../ui/button";
import { useRecoilValue, useSetRecoilState } from "recoil";
import { globalLastView, globalUserInfo, useResetStore } from "@/utils/store";
import { useLocation, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTheme } from "@/utils/theme";
import { toast } from "sonner";
import { UserRound } from "lucide-react";
import { Role } from "@/services/api/auth";

export const UserDropdownMenu: FC = () => {
  const queryClient = useQueryClient();
  const { resetAll } = useResetStore();
  const userInfo = useRecoilValue(globalUserInfo);
  const navigate = useNavigate();
  const location = useLocation();
  const { theme, setTheme } = useTheme();
  const setLastView = useSetRecoilState(globalLastView);

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
            <p className="text-sm font-medium leading-none">{userInfo.name}</p>
            <p className="text-xs leading-none text-muted-foreground">
              {userInfo.name}@act.buaa.edu.cn
            </p>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {isAdminView ? (
          <DropdownMenuGroup>
            <DropdownMenuItem
              onClick={() => {
                setLastView("portal");
                navigate("/portal");
                toast.success("切换至用户视图");
              }}
            >
              切换为普通用户
            </DropdownMenuItem>
          </DropdownMenuGroup>
        ) : (
          <DropdownMenuGroup>
            {userInfo.platformRole === Role.Admin && (
              <DropdownMenuItem
                onClick={() => {
                  setLastView("admin");
                  navigate("/admin");
                  toast.success("切换至管理员视图");
                }}
              >
                切换为管理员
              </DropdownMenuItem>
            )}
            <DropdownMenuItem disabled>个人空间</DropdownMenuItem>
          </DropdownMenuGroup>
        )}
        <DropdownMenuItem disabled>系统设置</DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
        >
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
          退出登录
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
