import type { FC } from "react";
import { useEffect, useMemo } from "react";
import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { apiJupyterTokenGet } from "@/services/api/vcjob";
import CraterIcon from "@/components/icon/CraterIcon";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { apiJobGetLog } from "@/services/api/aiTask";
import { ScrollArea } from "@/components/ui/scroll-area";

const Jupyter: FC = () => {
  // get param from url
  const { id } = useParams();

  const { data: taskLogs } = useQuery({
    queryKey: ["jupyter", "tasklog", id],
    queryFn: () => apiJobGetLog(id ?? ""),
    select: (res) => res.data.data,
    enabled: !!id,
  });

  const jupyterInfo = useQuery({
    queryKey: ["jupyter", id],
    queryFn: () => apiJupyterTokenGet(id ?? "0"),
    select: (res) => res.data.data,
    enabled: !!id,
  });

  const url = useMemo(() => {
    if (jupyterInfo.data) {
      return `https://crater.act.buaa.edu.cn/jupyter/${jupyterInfo.data.baseURL}?token=${jupyterInfo.data.token}`;
    } else {
      return "";
    }
  }, [jupyterInfo]);

  // set jupyter notebook icon as current page icon
  // icon url: `https://crater.act.buaa.edu.cn/jupyter/${jupyterInfo.data.baseURL}/static/favicons/favicon.ico`
  // set title to jupyter base url
  useEffect(() => {
    if (jupyterInfo.data?.baseURL) {
      const link = document.querySelector(
        "link[rel='website icon']",
      ) as HTMLLinkElement;
      if (link) {
        link.href = `https://crater.act.buaa.edu.cn/jupyter/${jupyterInfo.data.baseURL}/static/favicons/favicon.ico`;
        link.type = "image/x-icon";
      }
      document.title = jupyterInfo.data.baseURL;
    }
  }, [jupyterInfo]);

  return (
    <div className="relative h-screen w-screen">
      <iframe
        title="jupyter notebook"
        src={url}
        className="absolute bottom-0 left-0 right-0 top-0 h-screen w-screen"
      />
      <Sheet>
        <SheetTrigger asChild>
          <Button
            className="absolute bottom-10 right-4 h-10 w-10 rounded-full hover:bg-transparent"
            size="icon"
            variant={"ghost"}
            title="Jupyter 日志"
          >
            <CraterIcon />
          </Button>
        </SheetTrigger>
        {/* scroll in sheet: https://github.com/shadcn-ui/ui/issues/16 */}
        <SheetContent className="max-h-screen sm:max-w-3xl" side="left">
          <SheetHeader>
            <SheetTitle>Jupyter 日志</SheetTitle>
            <SheetDescription></SheetDescription>
          </SheetHeader>
          {taskLogs && (
            <Card className="bg-gray-100 text-muted-foreground dark:border dark:bg-transparent">
              <ScrollArea className="h-[calc(100vh_-_80px)]">
                <CardHeader className="py-3"></CardHeader>
                <CardContent>
                  <pre className="whitespace-pre-wrap text-sm">{taskLogs}</pre>
                </CardContent>
              </ScrollArea>
            </Card>
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
};

export default Jupyter;
