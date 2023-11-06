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
import { ExitIcon } from "@radix-ui/react-icons";
import { ScrollArea } from "./ui/scroll-area";

export type SidebarSubItem = {
  title: string;
  route: RouteObject;
};

export type SidebarItem = {
  title: string;
  path: string;
  icon: React.ReactNode;
  children: SidebarSubItem[];
  route?: RouteObject;
};

export type SidebarMenu = {
  title: string;
  path: string;
  icon: React.ReactNode;
  route: RouteObject;
};

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  sidebarItems: SidebarItem[];
  sidebarMenus: SidebarMenu[];
}

export function Sidebar({
  className,
  sidebarItems,
  sidebarMenus,
}: SidebarProps) {
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
        "flex flex-col items-center justify-between py-4",
        "border-r bg-secondary-foreground  text-white",
        "fixed top-0 z-20 md:sticky md:top-16 md:z-0 md:w-full",
        "min-h-full w-[300px]",
        ".3s transition-transform ease-in-out md:translate-x-0",
        className,
      )}
    >
      <ScrollArea className="h-[calc(100vh_-_192px)] w-full">
        <Accordion
          type="multiple"
          // collapsible
          className="w-full space-y-1 px-2 py-2"
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
                    <div className="mr-2">{item.icon}</div>
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
                <div className="mr-2">{item.icon}</div>
                {item.title}
              </button>
            );
          })}
        </Accordion>
      </ScrollArea>
      <div className="w-full space-y-1 px-2">
        <Separator className="mb-3 bg-slate-800" />
        {sidebarMenus.map((item) => (
          <button
            key={`sidebar-menu-${item.path}`}
            onClick={() => navigate(`/dashboard/${item.path}`)}
            className={cn(
              "flex h-9 w-full flex-row items-center justify-start rounded-md px-4 text-base hover:bg-primary/80",
              {
                "bg-slate-700": item.path === actived.item,
              },
            )}
          >
            <div className="mr-2">{item.icon}</div>
            {item.title}
          </button>
        ))}
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
        <p className="pt-2 text-center text-sm font-light text-muted-foreground">
          v0.0.0
        </p>
        {/* <p>v0.0.1</p> */}
      </div>
    </div>
  );
}
