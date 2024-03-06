import { cn } from "@/lib/utils";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { useEffect, useMemo, useRef, useState } from "react";
import { RouteObject, useLocation, useNavigate } from "react-router-dom";
import { ScrollArea } from "../ui/scroll-area";
import { useOnClickOutside } from "usehooks-ts";
import { getTitleByPath } from "@/utils/title";
import CraterIcon from "../icon/CraterIcon";
import CraterText from "../icon/CraterText";
import { toast } from "sonner";

export type SidebarSubItem = {
  route: RouteObject;
};

export type SidebarItem = {
  path: string;
  icon: React.ElementType;
  children: SidebarSubItem[];
  route?: RouteObject;
};

export type SidebarMenu = {
  path: string;
  icon: React.ElementType;
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
        "bg-sidebar text-sidebar-foreground",
        "fixed top-0 z-20 md:sticky md:top-16 md:z-0 md:w-full",
        "min-h-full w-[200px]",
        ".3s transition-transform ease-in-out md:translate-x-0",
        className,
      )}
      ref={ref}
    >
      {/* Logo */}
      <div className="flex h-14 w-full flex-row items-center justify-center pb-0 pt-3">
        <CraterIcon className="mr-1 h-7 w-7" />
        <CraterText className="h-3.5" />
      </div>
      {/* Calculate of ScrollArea height: */}
      {/* 100vh - (40px * 2 + 56px + 48px) */}
      <ScrollArea className="mt-2 h-[calc(100vh_-_193px)] w-full">
        <Accordion
          type="multiple"
          // collapsible
          className="w-full"
          value={accordion}
          onValueChange={setAccordion}
        >
          {sidebarItems.map((item) => {
            return item.children.length > 0 ? (
              <AccordionItem value={item.path} key={item.path}>
                <AccordionTrigger className="h-10">
                  <div className="flex flex-row items-center justify-start px-3 text-sm">
                    <item.icon className="mr-3" />
                    {getTitleByPath([actived.view, item.path])}
                  </div>
                </AccordionTrigger>
                {item.children && item.children.length > 0 && (
                  <AccordionContent>
                    {item.children?.map((subItem) => {
                      const subItemPath = subItem.route.path?.replace("/*", "");
                      return (
                        <button
                          key={`${item.path}-${subItemPath}`}
                          className={cn(
                            "  flex h-10 w-full flex-row items-center justify-start rounded-none pl-14 text-sm font-normal hover:bg-primary/20",
                            {
                              "bg-sidebar-item":
                                item.path === actived.item &&
                                subItemPath === actived.subItem,
                            },
                          )}
                          type="button"
                          onClick={() => {
                            const url = subItemPath
                              ? `/${actived.view}/${item.path}/${subItemPath}`
                              : `/${actived.view}/${item.path}`;
                            navigate(url);
                          }}
                        >
                          {getTitleByPath([
                            actived.view,
                            item.path,
                            subItemPath || "",
                          ])}
                        </button>
                      );
                    })}
                  </AccordionContent>
                )}
              </AccordionItem>
            ) : (
              <button
                key={`sidebar-item-${item.path}`}
                className={cn(
                  "flex h-10 w-full flex-row items-center justify-start rounded-none",
                  "px-7 text-sm font-normal hover:bg-primary/20",
                  {
                    "bg-sidebar-item": item.path === actived.item,
                  },
                )}
                onClick={() => navigate(`/${actived.view}/${item.path}`)}
              >
                <item.icon
                  className={cn("mr-3", {
                    "text-primary": item.path === actived.item,
                  })}
                />
                {getTitleByPath([actived.view, item.path])}
              </button>
            );
          })}
        </Accordion>
      </ScrollArea>
      <div className="w-full">
        {/* <Separator className="mb-4 bg-sidebar-item" /> */}
        <div className="">
          {sidebarMenus.map((item) => (
            <button
              key={`sidebar-menu-${item.path}`}
              type="button"
              onClick={() => navigate(`/${actived.view}/${item.path}`)}
              className={cn(
                "flex h-10 w-full flex-row items-center justify-start px-6 text-sm hover:bg-primary/20",
                {
                  "bg-sidebar-item": item.path === actived.item,
                },
              )}
            >
              <item.icon
                className={cn("mr-3", {
                  "text-primary": item.path === actived.item,
                })}
              />
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
            toast.success("切换至另一视图");
          }}
        >
          {import.meta.env.VITE_APP_VERSION}
        </p>
      </div>
    </div>
  );
}
