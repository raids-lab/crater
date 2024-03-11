import { cn } from "@/lib/utils";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import React, { useEffect, useState } from "react";
import { RouteObject, useLocation, useNavigate } from "react-router-dom";
import { ScrollArea } from "../ui/scroll-area";
import { getTitleByPath } from "@/utils/title";
import CraterIcon from "../icon/CraterIcon";
import CraterText from "../icon/CraterText";
import { toast } from "sonner";
import { ChevronDownIcon } from "@radix-ui/react-icons";
import { Button } from "../ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

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
  isMinimized: boolean;
  toggleIsMinimized: () => void;
}

interface ActivedState {
  view: string;
  item: string;
  subItem: string;
}

export function Sidebar({
  className,
  sidebarItems,
  sidebarMenus,
  isMinimized,
  toggleIsMinimized,
}: SidebarProps) {
  const [accordion, setAccordion] = useState<string>();
  const location = useLocation();
  const navigate = useNavigate();

  // Get actived item from location
  const [actived, setActived] = useState<ActivedState>({
    view: "",
    item: "",
    subItem: "",
  });

  useEffect(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    setActived({
      view: pathParts[0],
      item: pathParts[1],
      subItem: pathParts[2],
    });
  }, [location, setActived]);

  // Expand sidebar accordion
  useEffect(() => {
    // setAccordion((old) => {
    //   if (!old.includes(actived.item)) {
    //     return [...old, actived.item];
    //   }
    //   return old;
    // });
    setAccordion(actived.item);
  }, [actived, setAccordion]);

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-between",
        "bg-sidebar text-sidebar-foreground",
        "min-h-full w-[200px]",
        // "transition-all duration-300 ease-in-out",
        className,
        {
          "w-14": isMinimized,
        },
      )}
    >
      {/* Logo */}
      <div
        className="relative flex h-14 w-full flex-row items-center justify-center pb-0 pt-1"
        onDoubleClick={toggleIsMinimized}
      >
        <CraterIcon className={cn("h-7 w-7", { "mr-1": !isMinimized })} />
        <CraterText className={cn("h-3.5", { hidden: isMinimized })} />
        <div
          className={cn(
            "absolute right-0 z-10 flex h-8 w-8 translate-x-5 items-center justify-center rounded-full bg-background",
            {
              "translate-x-5": isMinimized,
            },
          )}
        >
          <Button
            size="icon"
            variant={"ghost"}
            className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 hover:bg-primary/15"
            onClick={toggleIsMinimized}
            title={isMinimized ? "展开边栏" : "收起边栏"}
          >
            <ChevronDownIcon
              className={cn(
                "h-4 w-4 rotate-90 text-primary transition-transform duration-200",
                {
                  "-rotate-90": isMinimized,
                },
              )}
            />
          </Button>
        </div>
      </div>
      {/* Calculate of ScrollArea height: */}
      {/* 100vh - (56px + 124px) */}
      <ScrollArea className="h-[calc(100vh_-_180px)] w-full">
        <Accordion
          type="single"
          // collapsible
          className="flex w-full flex-col items-center space-y-1 pt-2"
          value={accordion}
          onValueChange={setAccordion}
        >
          {sidebarItems.map((item) => {
            return item.children.length > 0 ? (
              isMinimized ? (
                <SidebarDropdownMenu
                  item={item}
                  actived={actived}
                  setActived={setActived}
                />
              ) : (
                <AccordionItem
                  value={item.path}
                  key={item.path}
                  className="w-[184px]"
                >
                  <AccordionTrigger
                    className={cn(
                      "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
                      {
                        "bg-primary/15 text-primary group-hover:bg-primary/15":
                          item.path === actived.item,
                      },
                    )}
                    onClick={() => {
                      if (accordion !== item.path) {
                        setActived((old) => ({
                          ...old,
                          item: item.path,
                          subItem: "",
                        }));
                      }
                    }}
                  >
                    <div className="flex flex-row items-center justify-start px-3 text-sm">
                      <item.icon className="mr-3" />
                      {getTitleByPath([actived.view, item.path])}
                    </div>
                  </AccordionTrigger>
                  {item.children && item.children.length > 0 && (
                    <AccordionContent>
                      {item.children?.map((subItem) => {
                        const subItemPath = subItem.route.path?.replace(
                          "/*",
                          "",
                        );
                        return (
                          <button
                            key={`${item.path}-${subItemPath}`}
                            className={cn(
                              "flex h-10 w-full flex-row items-center justify-start rounded-md",
                              "pl-14 text-sm font-normal transition-all duration-200 hover:text-primary",
                              {
                                "text-primary":
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
              )
            ) : (
              <SingleButton
                key={`sidebar-item-${item.path}`}
                title={getTitleByPath([actived.view, item.path])}
                onClick={() => navigate(`/${actived.view}/${item.path}`)}
                isActive={item.path === actived.item}
                isMinimized={isMinimized}
              >
                <item.icon />
              </SingleButton>
            );
          })}
        </Accordion>
      </ScrollArea>
      <div className="flex w-full flex-col items-center">
        {/* <Separator className="mb-4 bg-sidebar-item" /> */}
        <div className="space-y-1">
          {sidebarMenus.map((item) => (
            <SingleButton
              key={`sidebar-menu-${item.path}`}
              title={getTitleByPath([actived.view, item.path])}
              onClick={() => navigate(`/${actived.view}/${item.path}`)}
              isActive={item.path === actived.item}
              isMinimized={isMinimized}
            >
              <item.icon />
            </SingleButton>
          ))}
        </div>
        <p
          className="h-10 select-none pt-1 text-center text-xs font-light text-muted-foreground"
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

const SingleButton: React.FC<{
  title: string;
  key: string;
  onClick: () => void;
  isActive: boolean;
  children: React.ReactNode;
  isMinimized?: boolean;
}> = ({ title, key, onClick, isActive, children, isMinimized }) => {
  return (
    <button
      key={key}
      type="button"
      className={cn(
        "flex h-10 flex-row items-center rounded-md text-sm hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
        {
          "bg-primary/15 text-primary hover:bg-primary/15": isActive,
          "w-[184px] justify-start px-7": !isMinimized,
          "w-10 justify-center": isMinimized,
        },
      )}
      onClick={onClick}
    >
      {children}
      <p className={cn("ml-3", { hidden: isMinimized })}>{title}</p>
    </button>
  );
};

interface DropdownMenuProps {
  item: SidebarItem;
  actived: ActivedState;
  setActived: (value: React.SetStateAction<ActivedState>) => void;
}

const SidebarDropdownMenu: React.FC<DropdownMenuProps> = ({
  item,
  actived,
  setActived,
}) => {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const handleSubItemClick = (subItemPath: string) => {
    const url = subItemPath
      ? `/${actived.view}/${item.path}/${subItemPath}`
      : `/${actived.view}/${item.path}`;
    navigate(url);
  };

  useEffect(() => {
    if (open) {
      setActived((old) => ({
        ...old,
        item: item.path,
        subItem: "",
      }));
    }
  }, [open, setActived, item.path]);

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <button
          key={item.path}
          className={cn(
            "flex h-10 w-10 flex-row items-center justify-center rounded-md text-sm transition-all hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
            {
              "bg-primary/15 text-primary hover:bg-primary/15":
                item.path === actived.item,
            },
          )}
        >
          <item.icon />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="right" sideOffset={14} align="start">
        <DropdownMenuLabel>
          {getTitleByPath([actived.view, item.path])}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {item.children?.map((subItem) => {
          const subItemPath = subItem.route?.path?.replace("/*", "");
          return (
            <DropdownMenuItem
              key={`${item.path}-${subItemPath}`}
              onClick={() => handleSubItemClick(subItemPath || "")} // Handle empty subItemPath
            >
              {getTitleByPath([actived.view, item.path, subItemPath || ""])}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
