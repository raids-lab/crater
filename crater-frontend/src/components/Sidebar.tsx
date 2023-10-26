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
import clsx from "clsx";
import { useNavigate } from "react-router-dom";

export type SidebarSubItem = {
  title: string;
  path: string;
};

export type SidebarItem = {
  title: string;
  path: string;
  icon?: React.ReactNode;
  children: SidebarSubItem[];
};

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  sidebarItems: SidebarItem[];
}

export function Sidebar({ className, sidebarItems }: SidebarProps) {
  const [accordion, setAccordion] = useRecoilState(uiAccordionState);
  const actived = useRecoilValue(uiActivedState);
  const setUserInfo = useSetRecoilState(userInfoState);
  const navigate = useNavigate();

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
    <div className={cn("h-full border-r", className)}>
      <div
        className={cn(
          "flex h-full flex-col items-stretch justify-between px-2 py-2",
        )}
      >
        <Accordion
          type="multiple"
          // collapsible
          className="w-full py-2"
          defaultValue={[]}
          value={accordion}
          onValueChange={setAccordion}
        >
          {sidebarItems.map((item, i) => (
            <AccordionItem value={item.path} key={item.path}>
              <AccordionTrigger>
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
              <AccordionContent>
                {item.children?.map((subItem) => (
                  <Button
                    key={`${item.path}-${subItem.path}`}
                    variant="ghost"
                    className={clsx("w-full justify-start pl-10", {
                      "text-sky-600":
                        item.path === actived.item &&
                        subItem.path === actived.subItem,
                    })}
                    onClick={() => {
                      navigate(`/dashboard/${item.path}`);
                    }}
                  >
                    {subItem.title}
                  </Button>
                ))}
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
        <Button
          variant="outline"
          onClick={() => {
            setUserInfo((old) => {
              return {
                ...old,
                role: "viewer",
              };
            });
          }}
        >
          退出登录
        </Button>
      </div>
    </div>
  );
}
