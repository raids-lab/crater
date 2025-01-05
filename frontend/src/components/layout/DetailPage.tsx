import { LucideIcon } from "lucide-react";
import { ReactNode, useMemo } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import useFixedLayout from "@/hooks/useFixedLayout";
import { ScrollArea } from "../ui/scroll-area";
import { cn } from "@/lib/utils";

interface DetailTabProps {
  key: string;
  icon?: LucideIcon;
  label: string;
  children: ReactNode;
  scrollable?: boolean;
  hidden?: boolean;
}

interface DetailInfoProps {
  icon: LucideIcon;
  title: string;
  value: ReactNode;
  className?: string;
}

interface DetailPageProps {
  header: ReactNode;
  info: DetailInfoProps[];
  tabs: DetailTabProps[];
}

export function DetailPage({ header, info, tabs: rawTabs }: DetailPageProps) {
  useFixedLayout();

  const tabs = useMemo(() => {
    return rawTabs.filter((tab) => !tab.hidden);
  }, [rawTabs]);

  return (
    <div className="flex h-full w-full flex-col space-y-6">
      <div className="h-[132.5px] space-y-6">
        {header}
        <div className="grid grid-cols-3 gap-4 text-sm text-muted-foreground">
          {info.map((data, index) => (
            <div
              key={index}
              className={cn("flex items-center", data.className)}
            >
              <data.icon className="mr-1.5 size-4 text-muted-foreground" />
              <span className="truncate text-sm text-muted-foreground">
                {data.title}：
              </span>
              <span className="truncate">{data.value}</span>
            </div>
          ))}
        </div>
      </div>
      <Tabs
        defaultValue={tabs[0].key}
        className="w-full flex-grow overflow-hidden"
      >
        <TabsList className="tabs-list-underline">
          {tabs.map((tab) => (
            <TabsTrigger
              key={tab.key}
              className="tabs-trigger-underline"
              value={tab.key}
            >
              {tab.icon && <tab.icon className="size-4" />}
              {tab.label}
            </TabsTrigger>
          ))}
        </TabsList>
        {tabs.map((tab) => (
          <TabsContent key={tab.key} value={tab.key} className="w-full">
            {tab.scrollable ? (
              <ScrollArea className="h-[calc(100vh-_304px)] w-full">
                {tab.children}
              </ScrollArea>
            ) : (
              <>{tab.children}</>
            )}
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );
}
