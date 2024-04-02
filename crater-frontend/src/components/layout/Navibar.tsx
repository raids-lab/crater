import { useMemo, type FC, useEffect, Fragment } from "react";
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
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Button } from "../ui/button";
import { useRecoilState, useRecoilValue, useSetRecoilState } from "recoil";
import {
  globalBreadCrumb,
  globalLastView,
  globalUserInfo,
  useResetStore,
} from "@/utils/store";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTheme } from "@/utils/theme";
import { getBreadcrumbByPath } from "@/utils/title";
import { toast } from "sonner";
import TeamSwitcher from "./TeamSwitcher";
import { cn } from "@/lib/utils";
import { UserRound } from "lucide-react";

export const NavBreadcrumb = ({ className }: { className: string }) => {
  const location = useLocation();
  const pathParts = useMemo(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    return pathParts;
  }, [location]);
  const [breadcrumb, setBreadcrumb] = useRecoilState(globalBreadCrumb);
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
      if (ans.length > 2) {
        ans[ans.length - 2].path = undefined;
      }
      setBreadcrumb(ans.slice(1));
    }
  }, [pathParts, setBreadcrumb]);

  return (
    <Breadcrumb className={className}>
      <BreadcrumbList>
        {breadcrumb.map((item, index) => {
          return (
            <Fragment key={`bread-${index}`}>
              {index !== 0 && (
                <BreadcrumbSeparator key={`bread-separator-${index}`} />
              )}
              {index === 0 && (
                <BreadcrumbPage
                  className={cn({
                    "text-muted-foreground": breadcrumb.length > 1,
                  })}
                >
                  {item.title}
                </BreadcrumbPage>
              )}
              {index !== 0 && (
                <BreadcrumbItem key={`bread-item-${index}`}>
                  {item.path && (
                    <BreadcrumbLink asChild>
                      <Link to={item.path}>{item.title}</Link>
                    </BreadcrumbLink>
                  )}
                  {!item.path && <BreadcrumbPage>{item.title}</BreadcrumbPage>}
                </BreadcrumbItem>
              )}
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
};

export const ProjectSelector = ({ className }: { className: string }) => {
  const groups = {
    personal: {
      label: "Liyilong",
      value: "personal",
    },
    teams: [
      {
        label: "Serverless",
        value: "acme-inc",
      },
      {
        label: "GPU Sched",
        value: "monsters",
      },
    ],
  };

  return (
    <div className={className}>
      <TeamSwitcher projects={groups} />
    </div>
  );
};

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
            <p className="text-sm font-medium leading-none">{userInfo.id}</p>
            <p className="text-xs leading-none text-muted-foreground">
              {userInfo.id}@act.buaa.edu.cn
            </p>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {isAdminView ? (
          <DropdownMenuGroup>
            <DropdownMenuItem
              onClick={() => {
                navigate("/portal");
                toast.success("切换至用户视图");
              }}
            >
              切换为普通用户
            </DropdownMenuItem>
          </DropdownMenuGroup>
        ) : (
          <DropdownMenuGroup>
            {userInfo.role === "admin" && (
              <DropdownMenuItem
                onClick={() => {
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
