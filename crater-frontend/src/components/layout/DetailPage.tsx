import { LucideIcon } from "lucide-react";
import { ReactNode, useMemo } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import useFixedLayout from "@/hooks/useFixedLayout";
import { ScrollArea, ScrollBar } from "../ui/scroll-area";
import { cn } from "@/lib/utils";
import { useSearchParams } from "react-router-dom";

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
  const [searchParams, setSearchParams] = useSearchParams();

  const tabs = useMemo(() => {
    return rawTabs.filter((tab) => !tab.hidden);
  }, [rawTabs]);

  // 从 URL 查询参数中获取当前标签
  const currentTab = useMemo(() => {
    const tabFromUrl = searchParams.get("tab");
    // 检查 URL 中的标签是否存在且可见
    if (tabFromUrl && tabs.some((tab) => tab.key === tabFromUrl)) {
      return tabFromUrl;
    }
    // 默认使用第一个可见标签
    return tabs.length > 0 ? tabs[0].key : "";
  }, [searchParams, tabs]);

  // 处理标签切换
  const handleTabChange = (value: string) => {
    setSearchParams(
      (params) => {
        params.set("tab", value);
        return params;
      },
      { replace: true },
    );
  };

  return (
    <div className="flex h-full w-full flex-col space-y-6">
      <div className="h-32 space-y-6">
        {header}
        <div className="text-muted-foreground grid grid-cols-3 gap-3 text-sm">
          {info.map((data, index) => (
            <div
              key={index}
              className={cn("flex items-center", data.className)}
            >
              <data.icon className="text-muted-foreground mr-1.5 size-4" />
              <span className="text-muted-foreground truncate text-sm">
                {data.title}:
              </span>
              <span className="truncate">{data.value}</span>
            </div>
          ))}
        </div>
      </div>
      <Tabs
        value={currentTab}
        onValueChange={handleTabChange}
        className="w-full grow overflow-hidden"
      >
        <TabsList className="tabs-list-underline">
          {tabs.map((tab) => (
            <TabsTrigger
              key={tab.key}
              className="tabs-trigger-underline"
              value={tab.key}
            >
              {tab.icon && <tab.icon className="size-4" />}
              <p className="hidden md:block">{tab.label}</p>
            </TabsTrigger>
          ))}
        </TabsList>
        {tabs.map((tab) => (
          <TabsContent key={tab.key} value={tab.key} className="w-full">
            {tab.scrollable ? (
              <ScrollArea className="h-[calc(100vh_-_300px)] w-full">
                {tab.children}
                <ScrollBar orientation="horizontal" />
              </ScrollArea>
            ) : (
              <div className="h-[calc(100vh_-_300px)] w-full">
                {tab.children}
              </div>
            )}
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );
}
