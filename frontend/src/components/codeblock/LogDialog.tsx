import { ScrollArea } from "@/components/ui/scroll-area";
import useResizeObserver from "use-resize-observer";
import { Card } from "../ui/card";
import { apiGetPodContainerLog, ContainerInfo } from "@/services/api/tool";
import BaseCodeBlock from "./BaseCodeBlock";
import {
  CalendarArrowDown,
  CalendarOff,
  CopyCheckIcon,
  CopyIcon,
  DownloadIcon,
  RefreshCcw,
} from "lucide-react";
import { useCopyToClipboard } from "usehooks-ts";
import { ButtonGroup } from "../ui-custom/button-group";
import { Button } from "../ui/button";
import { PodContainerDialog, PodNamespacedName } from "./PodContainerDialog";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import LoadingCircleIcon from "../icon/LoadingCircleIcon";

const DEFAULT_TAIL_LINES = 500;

const scrollToBottom = (container: HTMLElement | null, smooth = false) => {
  if (container?.children.length) {
    const lastElement = container?.querySelector("pre")
      ?.lastChild as HTMLElement;
    lastElement?.scrollIntoView({
      behavior: smooth ? "smooth" : "auto",
      block: "end",
      inline: "nearest",
    });
  }
};

const getLineCount = (container: HTMLElement | null) => {
  // <div><pre>{children}</pre></div>
  // get the line count of the children inside pre tag
  return container?.querySelector("pre")?.children.length;
};

function LogCard({
  namespacedName,
  selectedContainer,
}: {
  namespacedName: PodNamespacedName;
  selectedContainer: ContainerInfo;
}) {
  const queryClient = useQueryClient();
  const [tailLines, setTailLines] = useState(DEFAULT_TAIL_LINES);
  const [showMore, setShowMore] = useState(false);
  const [timestamps, setTimestamps] = useState(false);
  const [copiedText, copy] = useCopyToClipboard();
  const { ref: refRoot, width, height } = useResizeObserver();
  const logAreaRef = useRef<HTMLDivElement>(null);

  const { data: logText, isLoading } = useQuery({
    queryKey: [
      "logtext",
      namespacedName.namespace,
      namespacedName.name,
      selectedContainer.name,
      "log",
      tailLines,
      timestamps,
    ],
    queryFn: () =>
      apiGetPodContainerLog(
        namespacedName.namespace,
        namespacedName.name,
        selectedContainer.name,
        {
          tailLines: tailLines,
          timestamps: timestamps,
          follow: false,
        },
      ),
    select: (res) => {
      // convert base64 to string
      return atob(res.data.data);
    },
    enabled:
      !!namespacedName.namespace &&
      !!namespacedName.name &&
      !!selectedContainer?.name &&
      !!tailLines,
  });

  const copyCode = () => {
    if (!logText) {
      toast.warning("没有日志可供复制");
      return;
    }
    copy(logText)
      .then(() => {
        toast.success("已复制到剪贴板");
      })
      .catch(() => {
        toast.error("复制失败");
      });
  };

  const handleDownload = () => {
    if (!logText) {
      toast.warning("没有日志可供下载");
      return;
    }
    const blob = new Blob([logText], {
      type: "text/plain;charset=utf-8",
    });
    const filename = selectedContainer?.name ?? "log";
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
  };

  const handleRefresh = () => {
    setTailLines(DEFAULT_TAIL_LINES);
    void queryClient.invalidateQueries({ queryKey: ["logtext"] });
  };

  const handleShowMore = () => {
    setTailLines((prev) => prev + DEFAULT_TAIL_LINES);
  };

  useEffect(() => {
    if (getLineCount(logAreaRef.current) === tailLines) {
      setShowMore(true);
    }
  }, [logText, tailLines]);

  // scroll to bottom when init
  useEffect(() => {
    scrollToBottom(logAreaRef.current);
  }, [logText]);

  if (!logText) {
    // TODO: (liyilong) loading or error
    return (
      <Card className="flex h-full items-center justify-center overflow-hidden rounded-md bg-slate-900 p-1 text-white dark:border md:col-span-2 xl:col-span-3">
        {isLoading && <LoadingCircleIcon />}
        {!isLoading && (
          <div className="font-normal text-slate-400">暂无日志</div>
        )}
      </Card>
    );
  }

  return (
    <Card
      className="relative h-full overflow-hidden rounded-md bg-slate-900 p-1 text-white dark:border md:col-span-2 xl:col-span-3"
      ref={refRoot}
    >
      <ScrollArea style={{ width, height }}>
        {showMore && (
          <div className="flex justify-center pt-2">
            <Button onClick={handleShowMore} size="sm">
              显示更多
            </Button>
          </div>
        )}
        <BaseCodeBlock code={logText} language="python" ref={logAreaRef} />
      </ScrollArea>
      <ButtonGroup className="absolute right-5 top-5 text-foreground">
        <Button
          onClick={handleRefresh}
          className="border-0 border-r hover:text-primary focus-visible:ring-0"
          variant="outline"
          size="icon"
          title="刷新"
        >
          <RefreshCcw className="h-4 w-4" />
        </Button>
        <Button
          onClick={() => setTimestamps((prev) => !prev)}
          className="border-0 border-r hover:text-primary focus-visible:ring-0"
          variant="outline"
          size="icon"
          title="显示时间戳"
        >
          {timestamps ? (
            <CalendarOff className="h-4 w-4" />
          ) : (
            <CalendarArrowDown className="h-4 w-4" />
          )}
        </Button>
        <Button
          onClick={copyCode}
          className="border-0 border-r hover:text-primary focus-visible:ring-0"
          variant="outline"
          size="icon"
          title="复制"
        >
          {copiedText ? (
            <CopyCheckIcon className="h-4 w-4" />
          ) : (
            <CopyIcon className="h-4 w-4" />
          )}
        </Button>
        <Button
          onClick={handleDownload}
          className="border-0 hover:text-primary focus-visible:ring-0"
          variant="outline"
          size="icon"
        >
          <DownloadIcon className="h-4 w-4" />
        </Button>
      </ButtonGroup>
    </Card>
  );
}

export default function LogDialog({
  namespacedName,
  setNamespacedName,
}: {
  namespacedName?: PodNamespacedName;
  setNamespacedName: (namespacedName: PodNamespacedName | undefined) => void;
}) {
  return (
    <PodContainerDialog
      namespacedName={namespacedName}
      setNamespacedName={setNamespacedName}
      ActionComponent={LogCard}
    />
  );
}
