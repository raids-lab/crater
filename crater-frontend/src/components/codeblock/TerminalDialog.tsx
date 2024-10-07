import { useState, useEffect } from "react";
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
import { useQuery } from "@tanstack/react-query";
import { apiGetPodContainers, ContainerStatus } from "@/services/api/tool";
import { toast } from "sonner";
import ResourceBadges from "../custom/ResourceBadges";
import { cn } from "@/lib/utils";
import { TableDate } from "../custom/TableDate";

function ContainerSelect({
  currentContainer,
  setCurrentContainer,
  containers,
}: {
  currentContainer: ContainerStatus;
  setCurrentContainer: (container: ContainerStatus) => void;
  containers: ContainerStatus[];
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
      <SelectTrigger className="w-full">
        <SelectValue placeholder="Select a worker" />
      </SelectTrigger>
      <SelectContent>
        {containers.map((container) => (
          <SelectItem key={container.name} value={container.name}>
            {container.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function Content({ podName }: { podName: string }) {
  const { ref: refRoot, width, height } = useResizeObserver();
  const [selectedContainer, setSelectedContainer] = useState<
    ContainerStatus | undefined
  >();

  const { data: containers } = useQuery({
    queryKey: ["terminal", podName, "containers"],
    queryFn: () => apiGetPodContainers(podName),
    select: (data) => {
      // mock data
      if (
        !data?.data.data.containers ||
        data.data.data.containers.length === 0
      ) {
        return [];
      }
      return [
        data?.data.data.containers[0],
        {
          name: "mock-init-container",
          image: "busybox:latest",
          resources: {
            cpu: "10",
            memory: "20Gi",
            "nvidia.com/gpu": "1",
          },
          isInitContainer: true,
          state: {
            running: undefined,
            waiting: undefined,
            terminated: {
              exitCode: 0,
              signal: 0,
              reason: "Completed",
              message: "aaa",
              startedAt: "1717-07-07T07:07:07Z",
              finishedAt: "2021-07-07T07:07:07Z",
              containerID: "",
            },
          },
          restartCount: 2,
        },
      ];
    },
    enabled: !!podName,
  });

  useEffect(() => {
    for (const container of containers || []) {
      if (!container.isInitContainer) {
        setSelectedContainer((prev) => {
          return prev || container;
        });
        break;
      }
    }
  }, [containers]);

  // const [isOpen, setIsOpen] = useState(true);
  const [terminalOutput, setTerminalOutput] = useState<string[]>([
    "Connecting...",
  ]);

  useEffect(() => {
    // Simulate terminal connection and output
    const timer = setTimeout(() => {
      setTerminalOutput((prev) => [
        ...prev,
        `Connected to ${podName}.`,
        "root@container:~# ",
      ]);
    }, 2000);

    return () => clearTimeout(timer);
  }, [podName]);

  if (!containers || !selectedContainer) {
    return <></>;
  }

  return (
    <>
      <DialogHeader className="flex h-[60px] flex-row items-center justify-start border-b px-5">
        <DialogTitle className="text-lg font-semibold">
          Web Terminal
        </DialogTitle>
      </DialogHeader>
      <div className="mx-5 grid h-[calc(100vh_-188px)] w-[calc(100vw_-144px)] gap-5 pb-5 md:grid-cols-3 xl:grid-cols-4">
        <Card
          className="h-full overflow-hidden rounded-md bg-slate-900 p-1 text-white dark:border md:col-span-2 xl:col-span-3"
          ref={refRoot}
        >
          <ScrollArea style={{ width, height }} className="p-3">
            {terminalOutput.map((line, index) => (
              <p key={index} className="font-mono text-sm">
                {line}
              </p>
            ))}
          </ScrollArea>
        </Card>
        <div className="space-y-4">
          <ContainerSelect
            currentContainer={selectedContainer}
            setCurrentContainer={setSelectedContainer}
            containers={containers}
          />
          <div className="hidden space-y-4 md:block">
            <h4 className="font-semibold">基本信息</h4>
            <div className="grid grid-cols-3 gap-4 px-2 text-sm">
              <div className="text-muted-foreground">状态:</div>
              <div className="group col-span-2 flex flex-row items-center justify-start gap-1">
                <div
                  className={cn("flex h-3 w-3 rounded-full", {
                    "bg-red-500 group-hover:bg-red-400":
                      !!selectedContainer.state.terminated,
                    "bg-purple-500 group-hover:bg-purple-400":
                      !!selectedContainer.state.waiting,
                    "bg-emerald-500 group-hover:bg-emerald-400":
                      !!selectedContainer.state.running,
                  })}
                />
                {!!selectedContainer.state.running && "运行中"}
                {!!selectedContainer.state.waiting && "等待中"}
                {!!selectedContainer.state.terminated && "已终止"}
              </div>
              <div className="text-muted-foreground">名称:</div>
              <div className="col-span-2 font-mono">
                {selectedContainer.name}
              </div>
              <div className="text-muted-foreground">镜像:</div>
              <div
                className="col-span-2 overflow-hidden truncate text-ellipsis font-mono"
                title={selectedContainer.image}
              >
                {selectedContainer.image}
              </div>
              <div className="text-muted-foreground">资源预算:</div>
              <div className="col-span-2">
                <ResourceBadges resources={selectedContainer.resources} />
              </div>
              <div className="text-muted-foreground">重启次数:</div>
              <div className="col-span-2 font-mono">
                {selectedContainer.restartCount}
              </div>
              {selectedContainer.state.terminated && (
                <>
                  <div className="text-muted-foreground">退出代码:</div>
                  <div className="col-span-2">
                    {selectedContainer.state.terminated.exitCode}
                  </div>
                  <div className="text-muted-foreground">退出原因:</div>
                  <div className="col-span-2">
                    {selectedContainer.state.terminated.reason}
                  </div>
                  <div className="text-muted-foreground">退出消息:</div>
                  <div className="col-span-2">
                    {selectedContainer.state.terminated.message}
                  </div>
                  <div className="text-muted-foreground">开始时间:</div>
                  <div className="col-span-2">
                    <TableDate
                      date={selectedContainer.state.terminated.startedAt}
                    />
                  </div>
                  <div className="text-muted-foreground">结束时间:</div>
                  <div className="col-span-2">
                    <TableDate
                      date={selectedContainer.state.terminated.finishedAt}
                    />
                  </div>
                </>
              )}
              {selectedContainer.state.waiting && (
                <>
                  <div className="text-muted-foreground">等待原因:</div>
                  <div className="col-span-2">
                    {selectedContainer.state.waiting.reason}
                  </div>
                  <div className="text-muted-foreground">等待消息:</div>
                  <div className="col-span-2">
                    {selectedContainer.state.waiting.message}
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

export default function TerminalDialog({
  // props
  isOpen,
  setIsOpen,
  podName,
}: {
  // props types
  isOpen: boolean;
  setIsOpen: (value: boolean) => void;
  podName: string;
}) {
  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      {/*  */}
      <DialogContent
        className="h-[calc(100vh_-104px)] w-[calc(100vw_-104px)] max-w-full gap-5 p-0"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <Content podName={podName} />
      </DialogContent>
    </Dialog>
  );
}
