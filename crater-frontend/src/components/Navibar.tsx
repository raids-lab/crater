import { useMemo, type FC, useEffect } from "react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
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
import { useRecoilState, useRecoilValue, useSetRecoilState } from "recoil";
import {
  globalBreadCrumb,
  globalLastView,
  globalUserInfo,
  useResetStore,
} from "@/utils/store";
import { Link, useLocation, useNavigate } from "react-router-dom";
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
  const [breadcrumb, setBreadcrumb] = useRecoilState(globalBreadCrumb);

  const pathParts = useMemo(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    return pathParts;
  }, [location]);

  const isAdminView = useMemo(() => {
    return pathParts[0] === "admin";
  }, [pathParts]);

  useEffect(() => {
    const titles = getBreadcrumbByPath(pathParts);
    if (titles) {
      let url = "";
      const ans = [];
      for (let i = 0; i < titles.length; i++) {
        url += `/${titles[i].path}`;
        ans.push({
          title: titles[i].title,
          path: i !== titles.length - 1 ? url : undefined,
        });
      }
      setBreadcrumb(ans.slice(1));
    }
  }, [pathParts, setBreadcrumb]);

  return (
    <div className="flex h-full flex-row items-center justify-between border-b bg-background px-6 shadow-sm">
      <div
        className="flex items-center space-x-1 md:space-x-2"
        aria-label="Breadcrumb"
      >
        {breadcrumb.map((item, index) => {
          if (item.path) {
            return (
              <div className="inline-flex items-center" key={`item${index}`}>
                {index !== 0 && (
                  <ChevronRightIcon className="mr-1 text-muted-foreground md:mr-2" />
                )}
                <Link
                  to={item.path}
                  className="inline-flex select-none items-center text-sm font-medium text-gray-700 hover:text-primary dark:text-gray-400 dark:hover:text-white"
                >
                  {item.title}
                </Link>
              </div>
            );
          } else {
            return (
              <div key={`item${index}`}>
                <div className="flex items-center">
                  {index !== 0 && (
                    <ChevronRightIcon className="mr-1 text-muted-foreground md:mr-2" />
                  )}
                  <span className="select-none text-sm font-medium text-muted-foreground dark:text-gray-400">
                    {item.title}
                  </span>
                </div>
              </div>
            );
          }
        })}
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="relative h-8 w-8 rounded-full">
            <Avatar className="h-8 w-8">
              {/* <AvatarImage src="/avatars/01.png" alt="@shadcn" /> */}
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
