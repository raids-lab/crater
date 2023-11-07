import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { uiActivedState, useResetStore } from "@/utils/store";
import { useRecoilValue } from "recoil";
import { useEffect, useRef, useState } from "react";
// import { logger } from "@/utils/loglevel";
import { RouteObject, useNavigate } from "react-router-dom";
import { useToast } from "./ui/use-toast";
import { useQueryClient } from "@tanstack/react-query";
import { Separator } from "./ui/separator";
import { ExitIcon } from "@radix-ui/react-icons";
import { ScrollArea } from "./ui/scroll-area";
import { useOnClickOutside } from "usehooks-ts";
import { useTheme } from "@/utils/theme";

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
  closeSidebar: () => void;
}

export function Sidebar({
  className,
  sidebarItems,
  sidebarMenus,
  closeSidebar,
}: SidebarProps) {
  const [accordion, setAccordion] = useState<string[]>([]);
  const actived = useRecoilValue(uiActivedState);
  const navigate = useNavigate();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const ref = useRef<HTMLDivElement>(null);
  const { theme, setTheme } = useTheme();
  const { resetAll } = useResetStore();

  useEffect(() => {
    setAccordion((old) => {
      // if sidebarItem.path not in old, add it
      if (!old.includes(actived.item)) {
        return [...old, actived.item];
      }
      return old;
    });
  }, [actived, setAccordion]);

  useOnClickOutside(ref, closeSidebar);

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-between pb-4",
        "border-r bg-slate-800 text-white dark:bg-background",
        "fixed top-0 z-20 md:sticky md:top-16 md:z-0 md:w-full",
        "min-h-full w-[240px]",
        ".3s transition-transform ease-in-out md:translate-x-0",
        className,
      )}
      ref={ref}
    >
      <ScrollArea className="h-[calc(100vh_-_176px)] w-full pt-4">
        <Accordion
          type="multiple"
          // collapsible
          className="w-full space-y-1 px-2 py-2"
          value={accordion}
          onValueChange={setAccordion}
        >
          {sidebarItems.map((item) => {
            return item.children.length > 0 ? (
              <AccordionItem value={item.path} key={item.path}>
                <AccordionTrigger>
                  <div className="flex flex-row items-center justify-start">
                    <div className="mr-3">{item.icon}</div>
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
                            "w-full justify-start pl-11 font-normal hover:bg-primary/80",
                            {
                              "bg-slate-700 dark:bg-secondary":
                                item.path === actived.item &&
                                subItem.route.path === actived.subItem,
                            },
                          )}
                          type="button"
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
                key={`sidebar-item-${item.path}`}
                className={cn(
                  "flex h-9 w-full flex-row items-center justify-start rounded-md px-4 text-base hover:bg-primary/80",
                  {
                    "bg-slate-700 dark:bg-secondary":
                      item.path === actived.item,
                  },
                )}
                type="button"
                onClick={() => navigate(`/dashboard/${item.path}`)}
              >
                <div className="mr-3">{item.icon}</div>
                {item.title}
              </button>
            );
          })}
        </Accordion>
      </ScrollArea>
      <div className="w-full">
        <Separator className="mb-4 bg-slate-700 dark:bg-secondary" />
        <div className="space-y-1 px-2">
          {sidebarMenus.map((item) => (
            <button
              key={`sidebar-menu-${item.path}`}
              type="button"
              onClick={() => navigate(`/dashboard/${item.path}`)}
              className={cn(
                "flex h-9 w-full flex-row items-center justify-start rounded-md px-4 text-base hover:bg-primary/80",
                {
                  "bg-slate-700 dark:bg-secondary": item.path === actived.item,
                },
              )}
            >
              <div className="mr-3">{item.icon}</div>
              {item.title}
            </button>
          ))}
          <button
            className="flex h-9 w-full flex-row items-center justify-start rounded-md px-4 text-base hover:bg-destructive"
            type="button"
            onClick={() => {
              queryClient.clear();
              resetAll();
              toast({
                title: "已退出",
              });
            }}
          >
            <ExitIcon className="mr-3 h-4 w-4" />
            退出登录
          </button>
        </div>
        <p
          className="select-none pt-2 text-center text-sm font-light text-muted-foreground"
          onDoubleClick={() => setTheme(theme === "dark" ? "light" : "dark")}
        >
          v0.0.0
        </p>
        {/* <p>v0.0.1</p> */}
      </div>
    </div>
  );
}
