import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { useEffect, useMemo, useRef, useState } from "react";
import { RouteObject, useLocation, useNavigate } from "react-router-dom";
import { Separator } from "./ui/separator";
import { ScrollArea } from "./ui/scroll-area";
import { useOnClickOutside } from "usehooks-ts";
import { getTitleByPath } from "@/utils/title";

export type SidebarSubItem = {
  route: RouteObject;
};

export type SidebarItem = {
  path: string;
  icon: React.ReactNode;
  children: SidebarSubItem[];
  route?: RouteObject;
};

export type SidebarMenu = {
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
  const location = useLocation();
  const navigate = useNavigate();

  // Get actived item from location
  const actived = useMemo(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    return {
      view: pathParts[0],
      item: pathParts[1],
      subItem: pathParts[2],
    };
  }, [location]);

  // Expand sidebar accordion
  useEffect(() => {
    setAccordion((old) => {
      if (!old.includes(actived.item)) {
        return [...old, actived.item];
      }
      return old;
    });
  }, [actived, setAccordion]);

  // Close sidebar when click outside
  const ref = useRef<HTMLDivElement>(null);
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
      {/* Calculate of ScrollArea height: */}
      {/* 100vh - (40px * 2 + 56px) */}
      <ScrollArea className="h-[calc(100vh_-_136px)] w-full pt-2">
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
                    {getTitleByPath([actived.view, item.path])}
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
                              ? `/${actived.view}/${item.path}/${subItem.route.path}`
                              : `/${actived.view}/${item.path}`;
                            navigate(url);
                          }}
                        >
                          {getTitleByPath([
                            actived.view,
                            item.path,
                            subItem.route.path || "",
                          ])}
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
                onClick={() => navigate(`/${actived.view}/${item.path}`)}
              >
                <div className="mr-3">{item.icon}</div>
                {getTitleByPath([actived.view, item.path])}
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
              onClick={() => navigate(`/${actived.view}/${item.path}`)}
              className={cn(
                "flex h-9 w-full flex-row items-center justify-start rounded-md px-4 text-base hover:bg-primary/80",
                {
                  "bg-slate-700 dark:bg-secondary": item.path === actived.item,
                },
              )}
            >
              <div className="mr-3">{item.icon}</div>
              {getTitleByPath([actived.view, item.path])}
            </button>
          ))}
        </div>
        <p
          className="select-none pt-2 text-center text-sm font-light text-muted-foreground"
          onDoubleClick={() => {
            if (actived.view === "portal") {
              navigate("/recommend");
            } else if (actived.view === "recommend") {
              navigate("/portal");
            }
          }}
        >
          v0.0.0
        </p>
      </div>
    </div>
  );
}
