// Reference: https://github.com/kubesphere/console/blob/master/packages/shared/src/stores/pod.ts#L187
import { useState, useEffect, useMemo } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import useResizeObserver from "use-resize-observer";
import { Card } from "../ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  apiGetPodContainerLog,
  apiGetPodContainers,
  ContainerInfo,
} from "@/services/api/tool";
import { toast } from "sonner";
import ResourceBadges from "../custom/ResourceBadges";
import { cn } from "@/lib/utils";
import { TableDate } from "../custom/TableDate";
import BaseCodeBlock from "./BaseCodeBlock";
import { shortenImageName, shortestImageName } from "@/utils/formatter";
import {
  BoxIcon,
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
import ContainerStatusLabel, {
  ContainerStatus,
} from "../phase/ContainerStatusLabel";

export interface PodNamespacedName {
  namespace: string;
  name: string;
}

function ContainerSelect({
  currentContainer,
  setCurrentContainer,
  containers,
}: {
  currentContainer: ContainerInfo;
  setCurrentContainer: (container: ContainerInfo) => void;
  containers: ContainerInfo[];
}) {
  return (
    <Select
      defaultValue={currentContainer.name}
      onValueChange={(name) => {
        const container = containers.find(
          (container) => container.name === name,
        );
        if (container) {
          setCurrentContainer(container);
        } else {
          toast.error("Container not found.");
        }
      }}
    >
      <SelectTrigger id="model" className={cn("h-14")}>
        <SelectValue placeholder="请选择 Container" />
      </SelectTrigger>
      <SelectContent>
        {containers.map((container) => (
          <SelectItem key={container.name} value={container.name}>
            <div className="flex items-center gap-3 text-muted-foreground">
              <div
                className={cn(
                  "flex size-8 items-center justify-center rounded-full font-normal",
                  {
                    "bg-primary/15 text-primary": !container.isInitContainer,
                    "bg-purple-500/15 text-purple-500":
                      container.isInitContainer,
                  },
                )}
              >
                <BoxIcon className="size-5" />
              </div>
              <div className="flex flex-col items-start gap-0.5">
                <p className="text-foreground">{container.name}</p>
                <p className="truncate font-mono text-xs" data-description>
                  {shortestImageName(container.image)}
                </p>
              </div>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

const TableCellForm = ({
  selectedContainer,
}: {
  selectedContainer: ContainerInfo;
}) => {
  const containerStatus: ContainerStatus = useMemo(() => {
    if (selectedContainer.state.running) {
      return ContainerStatus.Running;
    } else if (selectedContainer.state.terminated) {
      return ContainerStatus.Terminated;
    } else {
      return ContainerStatus.Waiting;
    }
  }, [selectedContainer]);

  return (
    <div className="grid grid-cols-3 content-start gap-4 px-1 text-sm">
      <p className="text-muted-foreground">状态</p>
      <div className="group col-span-2 flex flex-row items-center justify-start gap-1">
        <ContainerStatusLabel containerStatus={containerStatus} />
      </div>
      <p className="text-muted-foreground">名称</p>
      <p className="col-span-2 font-mono">{selectedContainer.name}</p>
      <p className="text-muted-foreground">镜像</p>
      <p
        className="col-span-2 overflow-hidden whitespace-normal break-words font-mono"
        title={selectedContainer.image}
      >
        {shortenImageName(selectedContainer.image)}
      </p>
      {!!selectedContainer.resources && (
        <>
          <div className="text-muted-foreground">资源预算</div>
          <div className="col-span-2">
            <ResourceBadges resources={selectedContainer.resources} />
          </div>
        </>
      )}
      <div className="text-muted-foreground">重启次数</div>
      <div className="col-span-2 font-mono">
        {selectedContainer.restartCount}
      </div>
      {selectedContainer.state.terminated && (
        <>
          <div className="text-muted-foreground">退出代码</div>
          <div className="col-span-2 font-mono">
            {selectedContainer.state.terminated.exitCode}
          </div>
          <div className="text-muted-foreground">退出原因</div>
          <div className="col-span-2">
            {selectedContainer.state.terminated.reason}
          </div>
          <div className="text-muted-foreground">退出消息</div>
          <div className="col-span-2">
            {selectedContainer.state.terminated.message}
          </div>
          <div className="text-muted-foreground">开始时间</div>
          <div className="col-span-2">
            <TableDate date={selectedContainer.state.terminated.startedAt} />
          </div>
          <div className="text-muted-foreground">结束时间</div>
          <div className="col-span-2">
            <TableDate date={selectedContainer.state.terminated.finishedAt} />
          </div>
        </>
      )}
      {selectedContainer.state.waiting && (
        <>
          <div className="text-muted-foreground">等待原因</div>
          <div className="col-span-2">
            {selectedContainer.state.waiting.reason}
          </div>
          <div className="text-muted-foreground">等待消息</div>
          <div className="col-span-2">
            {selectedContainer.state.waiting.message}
          </div>
        </>
      )}
    </div>
  );
};

function Content({
  PodNamespacedName,
}: {
  PodNamespacedName?: PodNamespacedName;
}) {
  const { namespace, name: podName } = PodNamespacedName ?? {};
  const queryClient = useQueryClient();
  const [copiedText, copy] = useCopyToClipboard();
  const { ref: refRoot, width, height } = useResizeObserver();
  const [tailLines, setTailLines] = useState(500);
  const [timestamps, setTimestamps] = useState(false);
  const [selectedContainer, setSelectedContainer] = useState<
    ContainerInfo | undefined
  >();

  // const { data: containers } = useQuery({
  //   queryKey: ["terminal", podName, "containers"],
  //   queryFn: () => apiGetPodContainers(podName),
  //   select: (data) => data.data.data.containers,
  //   enabled: !!podName,
  // });

  const { data: containers } = useQuery({
    queryKey: ["log", "containers", namespace, podName],
    queryFn: () => apiGetPodContainers(namespace, podName),
    select: (res) => res.data.data.containers,
    enabled: !!namespace && !!podName,
  });

  const { data: logText } = useQuery({
    queryKey: [
      "logtext",
      namespace,
      podName,
      selectedContainer?.name,
      "log",
      tailLines,
      timestamps,
    ],
    queryFn: () =>
      apiGetPodContainerLog(namespace, podName, selectedContainer?.name ?? "", {
        tailLines: tailLines,
        timestamps: timestamps,
        follow: false,
      }),
    select: (res) => {
      // convert base64 to string
      return atob(res.data.data);
    },
    enabled:
      !!namespace && !!podName && !!selectedContainer?.name && !!tailLines,
  });

  useEffect(() => {
    for (const container of containers || []) {
      if (!container.isInitContainer) {
        setSelectedContainer((prev) => {
          return prev || container;
        });
        void queryClient.invalidateQueries({ queryKey: ["logtext"] });
        break;
      }
    }
  }, [containers, queryClient]);

  const copyCode = () => {
    // Logic to copy `code`
    copy(logText ?? "")
      .then(() => {
        toast.success("已复制到剪贴板");
      })
      .catch(() => {
        toast.error("复制失败");
      });
  };

  const handleRefresh = () => {
    setTailLines(1000);
    void queryClient.invalidateQueries({ queryKey: ["logtext"] });
  };

  const handleDownload = () => {
    const blob = new Blob([logText ?? ""], {
      type: "text/plain;charset=utf-8",
    });
    const filename = selectedContainer?.name ?? "log";
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
  };

  if (!containers || !selectedContainer) {
    return <></>;
  }

  return (
    <>
      <DialogHeader className="flex h-[48px] flex-row items-center justify-start border-b px-5">
        <DialogTitle className="text-lg font-semibold">
          <span className="font-mono">{podName}</span>
        </DialogTitle>
      </DialogHeader>
      <div className="mx-5 grid h-[calc(100vh_-176px)] w-[calc(100vw_-144px)] gap-5 pb-5 md:grid-cols-3 xl:grid-cols-4">
        <Card
          className="relative h-full overflow-hidden rounded-md bg-slate-900 p-1 text-white dark:border md:col-span-2 xl:col-span-3"
          ref={refRoot}
        >
          <ScrollArea style={{ width, height }}>
            <BaseCodeBlock code={logText ?? ""} language="python" />
          </ScrollArea>
          <ButtonGroup className="absolute right-6 top-4 text-foreground">
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
        <div className="space-y-3">
          <ContainerSelect
            currentContainer={selectedContainer}
            setCurrentContainer={setSelectedContainer}
            containers={containers}
          />

          <fieldset className="hidden h-[calc(100vh_-264px)] max-h-full gap-6 overflow-y-auto rounded-lg border border-input p-4 shadow-sm md:grid">
            <legend className="-ml-1 px-2 text-sm font-medium">容器信息</legend>
            <TableCellForm selectedContainer={selectedContainer} />
          </fieldset>
        </div>
      </div>
    </>
  );
}

export default function LogDialog({
  // props
  namespacedName,
  setNamespacedName,
}: {
  // props types
  namespacedName?: PodNamespacedName;
  setNamespacedName: (namespacedName: PodNamespacedName | undefined) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);

  // if namespacedName is set, open the dialog
  // if dialog is closed, reset the namespacedName
  useEffect(() => {
    if (namespacedName) {
      setIsOpen(true);
    }
  }, [namespacedName]);

  useEffect(() => {
    if (!isOpen) {
      setNamespacedName(undefined);
    }
  }, [isOpen, setNamespacedName]);

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent
        className="h-[calc(100vh_-104px)] w-[calc(100vw_-104px)] max-w-full gap-5 p-0"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <Content PodNamespacedName={namespacedName} />
      </DialogContent>
    </Dialog>
  );
}
