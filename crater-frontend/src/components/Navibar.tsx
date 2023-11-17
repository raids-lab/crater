import { useMemo, type FC } from "react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  // DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import { Button } from "./ui/button";
import { useRecoilValue, useSetRecoilState } from "recoil";
import { globalLastView, globalUserInfo, useResetStore } from "@/utils/store";
import { useLocation, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useToast } from "./ui/use-toast";
import { useTheme } from "@/utils/theme";
import { ChevronRightIcon } from "@radix-ui/react-icons";
import { getBreadcrumbByPath } from "@/utils/title";

const Navibar: FC = () => {
  const queryClient = useQueryClient();
  const { toast } = useToast();
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

  const breadcrumb = useMemo(() => {
    return getBreadcrumbByPath(pathParts);
  }, [pathParts]);

  return (
    <div className="flex h-full flex-row items-center justify-between border-b bg-background px-6 shadow-sm">
      <nav className="flex" aria-label="Breadcrumb">
        <ol className="inline-flex items-center space-x-1 rtl:space-x-reverse md:space-x-2">
          {breadcrumb.slice(1).map((item, index) => {
            if (index === 0) {
              return (
                <li className="inline-flex items-center" key={item.path}>
                  <a
                    href="#"
                    className="inline-flex items-center text-sm font-medium text-gray-700 hover:text-primary dark:text-gray-400 dark:hover:text-white"
                  >
                    {item.title}
                  </a>
                </li>
              );
            }
            if (index === breadcrumb.length - 1) {
              return (
                <li key={item.path}>
                  <div className="flex items-center">
                    <ChevronRightIcon className="text-muted-foreground" />
                    <span className="ms-1 text-sm font-medium text-muted-foreground dark:text-gray-400 md:ms-2">
                      {item.title}
                    </span>
                  </div>
                </li>
              );
            }
            return (
              <li key={item.path}>
                <div className="flex items-center">
                  <ChevronRightIcon className="text-muted-foreground" />
                  <a
                    href="#"
                    className="ms-1 text-sm font-medium text-gray-700 hover:text-primary dark:text-gray-400 dark:hover:text-white md:ms-2"
                  >
                    {item.title}
                  </a>
                </div>
              </li>
            );
          })}
        </ol>
      </nav>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="relative h-8 w-8 rounded-full">
            <Avatar className="h-8 w-8">
              <AvatarImage src="/avatars/01.png" alt="@shadcn" />
              <AvatarFallback className="select-none">
                {
                  // Get first two char of userInfo.id and upper case
                  userInfo.id.slice(0, 2).toUpperCase()
                }
              </AvatarFallback>
            </Avatar>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-44" align="end" forceMount>
          <DropdownMenuLabel className="font-normal">
            <div className="flex flex-col space-y-1">
              <p className="text-sm font-medium leading-none">{userInfo.id}</p>
              <p className="text-xs leading-none text-muted-foreground">
                {userInfo.id}@act.buaa.edu.cn
              </p>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          {isAdminView ? (
            <DropdownMenuGroup>
              <DropdownMenuItem onClick={() => navigate("/portal")}>
                切换为普通用户
              </DropdownMenuItem>
            </DropdownMenuGroup>
          ) : (
            <DropdownMenuGroup>
              {userInfo.role === "admin" && (
                <DropdownMenuItem onClick={() => navigate("/admin/user")}>
                  切换为管理员
                </DropdownMenuItem>
              )}
              <DropdownMenuItem>个人空间</DropdownMenuItem>
            </DropdownMenuGroup>
          )}
          <DropdownMenuItem>系统设置</DropdownMenuItem>
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
              toast({
                title: "已退出",
              });
            }}
          >
            退出登录
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};

export default Navibar;
