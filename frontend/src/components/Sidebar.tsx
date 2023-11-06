import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { uiAccordionState, uiActivedState, userInfoState } from "@/utils/store";
import { useRecoilState, useRecoilValue, useSetRecoilState } from "recoil";
import { useEffect } from "react";
// import { logger } from "@/utils/loglevel";
import { RouteObject, useNavigate } from "react-router-dom";
import { useToast } from "./ui/use-toast";
import { useQueryClient } from "@tanstack/react-query";
import { Separator } from "./ui/separator";
import { ExitIcon, FileTextIcon, Pencil2Icon } from "@radix-ui/react-icons";

export type SidebarSubItem = {
  title: string;
  route: RouteObject;
};

export type SidebarItem = {
  title: string;
  path: string;
  icon?: React.ReactNode;
  children: SidebarSubItem[];
  route?: RouteObject;
};

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  sidebarItems: SidebarItem[];
}

export function Sidebar({ className, sidebarItems }: SidebarProps) {
  const [accordion, setAccordion] = useRecoilState(uiAccordionState);
  const actived = useRecoilValue(uiActivedState);
  const setUserInfo = useSetRecoilState(userInfoState);
  const navigate = useNavigate();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  useEffect(() => {
    setAccordion((old) => {
      // if sidebarItem.path not in old, add it
      if (!old.includes(actived.item)) {
        return [...old, actived.item];
      }
      return old;
    });
  }, [actived, setAccordion]);

  return (
    <div
      className={cn(
        className,
        "flex h-full flex-col items-center justify-between border-r bg-secondary-foreground px-2 py-2 text-white",
      )}
    >
      <Accordion
        type="multiple"
        // collapsible
        className="w-full space-y-1 py-2"
        value={accordion}
        onValueChange={setAccordion}
      >
        {sidebarItems.map((item, i) => {
          return item.children.length > 0 ? (
            <AccordionItem value={item.path} key={item.path}>
              <AccordionTrigger className="h-9 rounded-md hover:bg-primary/80">
                <div
                  key={`$button-${i}`}
                  className="flex flex-row items-center justify-start"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    className="mr-2 h-4 w-4"
                  >
                    <rect width="7" height="7" x="3" y="3" rx="1" />
                    <rect width="7" height="7" x="14" y="3" rx="1" />
                    <rect width="7" height="7" x="14" y="14" rx="1" />
                    <rect width="7" height="7" x="3" y="14" rx="1" />
                  </svg>
                  {item.title}
                </div>
              </AccordionTrigger>
              {item.children && item.children.length > 0 && (
                <AccordionContent>
                  <div className="space-y-1">
                    {item.children?.map((subItem) => (
                      <Button
                        key={`${item.path}-${subItem.route.path}`}
                        variant="colorable"
                        className={cn(
                          "w-full justify-start pl-10 font-normal hover:bg-primary/80",
                          {
                            "bg-slate-700":
                              item.path === actived.item &&
                              subItem.route.path === actived.subItem,
                          },
                        )}
                        onClick={() => {
                          const url = subItem.route.path
                            ? `/dashboard/${item.path}/${subItem.route.path}`
                            : `/dashboard/${item.path}`;
                          navigate(url);
                        }}
                      >
                        {subItem.title}
                      </Button>
                    ))}
                  </div>
                </AccordionContent>
              )}
            </AccordionItem>
          ) : (
            <button
              className={cn(
                "flex h-9 w-full flex-row items-center justify-start rounded-md px-4 text-base hover:bg-primary/80",
                {
                  "bg-slate-700": item.path === actived.item,
                },
              )}
              onClick={() => navigate(`/dashboard/${item.path}`)}
            >
              <FileTextIcon className="mr-2 h-4 w-4" />
              {item.title}
            </button>
          );
        })}
      </Accordion>
      <div className="w-full items-center justify-end space-y-1 pb-4">
        <Separator className="mb-3 bg-muted-foreground" />
        <button className="flex h-9 w-full flex-row items-center justify-start rounded-md px-4 text-base hover:bg-slate-700">
          <FileTextIcon className="mr-2 h-4 w-4" />
          使用文档
        </button>
        <button className="flex h-9 w-full flex-row items-center justify-start rounded-md px-4 text-base hover:bg-slate-700">
          <Pencil2Icon className="mr-2 h-4 w-4" />
          问题反馈
        </button>
        <button
          className="flex h-9 w-full flex-row items-center justify-start rounded-md px-4 text-base hover:bg-destructive"
          onClick={() => {
            queryClient.clear();
            setUserInfo((old) => {
              return {
                ...old,
                role: "viewer",
              };
            });
            toast({
              title: "已退出",
            });
          }}
        >
          <ExitIcon className="mr-2 h-4 w-4" />
          退出登录
        </button>
        {/* <p>v0.0.1</p> */}
      </div>
    </div>
  );
}
