import { ScrollArea } from "@/components/ui/scroll-area";
import useResizeObserver from "use-resize-observer";
import { Card } from "../ui/card";
import { apiGetPodContainerLog, ContainerInfo } from "@/services/api/tool";
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
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import LoadingCircleIcon from "../icon/LoadingCircleIcon";

const DEFAULT_TAIL_LINES = 500;

function LogCard({
  namespacedName,
  selectedContainer,
}: {
  namespacedName: PodNamespacedName;
  selectedContainer: ContainerInfo;
}) {
  const queryClient = useQueryClient();
  const [tailLines, setTailLines] = useState(DEFAULT_TAIL_LINES);
  const [timestamps, setTimestamps] = useState(false);
  const [copied, setCopied] = useState(false);
  const [, copy] = useCopyToClipboard();
  const { ref: refRoot, width, height } = useResizeObserver();
  const logAreaRef = useRef<HTMLDivElement>(null);

  const { data: logText } = useQuery({
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
          previous: false,
        },
      ),
    select: (res) => {
      // convert base64 to string
      return atob(res.data.data);
    },
    enabled:
      !selectedContainer.state.waiting &&
      !!namespacedName.namespace &&
      !!namespacedName.name &&
      !!selectedContainer?.name &&
      !!tailLines,
  });

  const { mutate: DownloadTotalLog } = useMutation({
    mutationFn: () =>
      apiGetPodContainerLog(
        namespacedName.namespace,
        namespacedName.name,
        selectedContainer.name,
        {
          tailLines: undefined,
          timestamps: timestamps,
          follow: false,
          previous: false,
        },
      ),
    onSuccess: (res) => {
      const logText = atob(res.data.data);
      const blob = new Blob([logText], {
        type: "text/plain;charset=utf-8",
      });
      const filename = selectedContainer?.name ?? "log";
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      a.click();
    },
  });

  const copyCode = () => {
    if (!logText) {
      toast.warning("没有日志可供复制");
      return;
    }
    copy(logText)
      .then(() => {
        setCopied(true);
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
    DownloadTotalLog();
  };

  const handleRefresh = () => {
    setTailLines(DEFAULT_TAIL_LINES);
    setCopied(false);
    setTimestamps(false);
    void queryClient.invalidateQueries({ queryKey: ["logtext"] });
  };

  // scroll to bottom when init
  const [showLog, setShowLog] = useState(false);
  useEffect(() => {
    if (tailLines === DEFAULT_TAIL_LINES) {
      logAreaRef.current?.scrollIntoView(false);
    }
    setShowLog(true);
    return () => {
      setShowLog(false);
    };
  }, [logText, tailLines]);

  const showMore = useMemo(() => {
    if (logText) {
      const lines = logText.split("\n").length;
      return lines > tailLines;
    }
    return false;
  }, [logText, tailLines]);

  const handleShowMore = () => {
    setTailLines((prev) => prev + DEFAULT_TAIL_LINES);
  };

  return (
    <Card
      className="relative h-full overflow-hidden rounded-md bg-slate-900 p-1 text-white dark:border dark:bg-muted/30 md:col-span-2 xl:col-span-3"
      ref={refRoot}
    >
      {showLog ? (
        <>
          <ScrollArea style={{ width, height }}>
            <div ref={logAreaRef}>
              {showMore && (
                <div className="flex select-none justify-center pt-5">
                  <Button size="sm" onClick={handleShowMore}>
                    显示更多
                  </Button>
                </div>
              )}
              <pre className="whitespace-pre-wrap break-words px-3 py-3 text-sm text-cyan-200 dark:text-blue-300">
                {logText}
              </pre>
            </div>
          </ScrollArea>
          <ButtonGroup className="absolute right-5 top-5 rounded-md border border-input bg-background text-foreground">
            <Button
              onClick={handleRefresh}
              className="border-0 border-r hover:text-primary focus-visible:ring-0"
              variant="ghost"
              size="icon"
              title="刷新"
            >
              <RefreshCcw className="h-4 w-4" />
            </Button>
            <Button
              onClick={() => setTimestamps((prev) => !prev)}
              className="border-0 border-r hover:text-primary focus-visible:ring-0"
              variant="ghost"
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
              variant="ghost"
              size="icon"
              title="复制"
            >
              {copied ? (
                <CopyCheckIcon className="h-4 w-4" />
              ) : (
                <CopyIcon className="h-4 w-4" />
              )}
            </Button>
            <Button
              onClick={handleDownload}
              className="border-0 hover:text-primary focus-visible:ring-0"
              variant="ghost"
              size="icon"
            >
              <DownloadIcon className="h-4 w-4" />
            </Button>
          </ButtonGroup>
        </>
      ) : (
        <LoadingCircleIcon />
      )}
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
