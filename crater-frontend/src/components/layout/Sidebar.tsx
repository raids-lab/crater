import { cn } from "@/lib/utils";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import React, { Fragment, useEffect, useState } from "react";
import { RouteObject, useLocation, useNavigate } from "react-router-dom";
import { ScrollArea } from "../ui/scroll-area";
import { getTitleByPath } from "@/utils/title";
import { toast } from "sonner";
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
    <div className={className}>
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
              <Fragment key={item.path}>
                <SidebarDropdownMenu
                  key={item.path}
                  item={item}
                  actived={actived}
                  setActived={setActived}
                  className="flex xl:hidden"
                />
                <AccordionItem
                  value={item.path}
                  key={item.path}
                  className="hidden w-44 border-0 xl:block"
                >
                  <AccordionTrigger
                    className={cn(
                      "h-9 rounded-md px-4 hover:text-primary hover:no-underline focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring [&>svg]:hover:text-primary",
                      {
                        "bg-primary/15 text-primary hover:bg-primary/15 [&>svg]:text-primary":
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
                    <div className="flex flex-row items-center justify-start pl-3 text-sm">
                      <item.icon className="mr-3" />
                      {getTitleByPath([actived.view, item.path])}
                    </div>
                  </AccordionTrigger>
                  {item.children && item.children.length > 0 && (
                    <AccordionContent className="pb-0">
                      {item.children?.map((subItem) => {
                        const subItemPath = subItem.route.path?.replace(
                          "/*",
                          "",
                        );
                        return (
                          <button
                            key={`${item.path}-${subItemPath}`}
                            className={cn(
                              "flex h-9 w-full flex-row items-center justify-start rounded-md",
                              "mt-1 pl-14 text-sm font-normal transition-all duration-200 hover:text-primary",
                              "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-ring",
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
              </Fragment>
            ) : (
              <SingleButton
                key={`sidebar-item-${item.path}`}
                title={getTitleByPath([actived.view, item.path])}
                onClick={() => navigate(`/${actived.view}/${item.path}`)}
                isActive={item.path === actived.item}
              >
                <item.icon />
              </SingleButton>
            );
          })}
        </Accordion>
      </ScrollArea>
      <div className="flex w-full flex-col items-center">
        <div className="space-y-1">
          {sidebarMenus.map((item) => (
            <SingleButton
              key={`sidebar-menu-${item.path}`}
              title={getTitleByPath([actived.view, item.path])}
              onClick={() => navigate(`/${actived.view}/${item.path}`)}
              isActive={item.path === actived.item}
            >
              <item.icon />
            </SingleButton>
          ))}
        </div>
        <p
          className="h-9 select-none pt-1 text-center text-xs font-light text-muted-foreground"
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
  onClick: () => void;
  isActive: boolean;
  children: React.ReactNode;
}> = ({ title, onClick, isActive, children }) => {
  return (
    <button
      type="button"
      className={cn(
        "flex h-9 flex-row items-center rounded-md text-sm hover:text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
        "xl:w-44 xl:justify-start xl:pl-7",
        "w-9 justify-center",
        {
          "bg-primary/15 text-primary hover:bg-primary/15": isActive,
        },
      )}
      onClick={onClick}
    >
      {children}
      <p className="hidden xl:ml-3 xl:block">{title}</p>
    </button>
  );
};

interface DropdownMenuProps {
  item: SidebarItem;
  actived: ActivedState;
  setActived: (value: React.SetStateAction<ActivedState>) => void;
  className?: string;
}

const SidebarDropdownMenu: React.FC<DropdownMenuProps> = ({
  item,
  actived,
  setActived,
  className,
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
            "flex h-9 w-9 flex-row items-center justify-center rounded-md text-sm transition-all hover:text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
            {
              "bg-primary/15 text-primary hover:bg-primary/15":
                item.path === actived.item,
            },
            className,
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
