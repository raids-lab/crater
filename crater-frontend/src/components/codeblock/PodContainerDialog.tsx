// Reference: https://github.com/kubesphere/console/blob/master/packages/shared/src/stores/pod.ts#L187
import { useState, useEffect, useMemo, Fragment } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGetPodContainers, ContainerInfo } from "@/services/api/tool";
import { toast } from "sonner";
import ResourceBadges from "../badge/ResourceBadges";
import { cn } from "@/lib/utils";
import { TimeDistance } from "../custom/TimeDistance";
import { shortenImageName, shortestImageName } from "@/utils/formatter";
import { BoxIcon } from "lucide-react";
import ContainerStatusBadge, {
  ContainerStatus,
} from "../badge/ContainerStatusBadge";
import { useNamespacedState } from "@/hooks/useNamespacedState";
import LoadingCircleIcon from "../icon/LoadingCircleIcon";
import TipBadge from "../badge/TipBadge";

export interface PodNamespacedName {
  namespace: string;
  name: string;
}

export type NamespacedName = PodNamespacedName | undefined;

export interface PodContainerDialogProps {
  namespacedName: NamespacedName;
  setNamespacedName: (namespacedName: NamespacedName) => void;
}

export interface PodIngressDialogProps {
  namespacedName: NamespacedName;
  setNamespacedName: (namespacedName: NamespacedName) => void;
  userName: string;
  jobName: string;
}

export function ContainerSelect({
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
      <SelectTrigger className="h-14 w-full data-[size=default]:h-14">
        <SelectValue placeholder="请选择 Container" />
      </SelectTrigger>
      <SelectContent>
        {containers.map((container) => (
          <SelectItem key={container.name} value={container.name}>
            <div className="text-muted-foreground flex items-center gap-3">
              <div
                className={cn(
                  "flex size-8 items-center justify-center rounded-full font-normal",
                  {
                    "bg-primary/15": !container.isInitContainer,
                    "bg-purple-500/15": container.isInitContainer,
                  },
                )}
              >
                <BoxIcon
                  className={cn("size-5", {
                    "text-primary": !container.isInitContainer,
                    "text-purple-500": container.isInitContainer,
                  })}
                />
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

export const TableCellForm = ({
  selectedContainer,
  appendInfos,
}: {
  selectedContainer: ContainerInfo;
  appendInfos?: {
    title: string;
    content: React.ReactNode;
  }[];
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
        <ContainerStatusBadge containerStatus={containerStatus} />
      </div>
      <p className="text-muted-foreground">名称</p>
      <p className="col-span-2 font-mono">{selectedContainer.name}</p>
      <p className="text-muted-foreground">镜像</p>
      <p
        className="col-span-2 overflow-hidden font-mono break-words whitespace-normal"
        title={selectedContainer.image}
      >
        {shortenImageName(selectedContainer.image)}
      </p>
      {appendInfos?.map((info, index) => (
        <Fragment key={index}>
          <div className="text-muted-foreground">{info.title}</div>
          <div className="col-span-2 font-mono break-all whitespace-normal">
            {info.content}
          </div>
        </Fragment>
      ))}
      {!!selectedContainer.resources && (
        <>
          <div className="text-muted-foreground">申请资源</div>
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
          <div className="col-span-2 font-mono break-all whitespace-normal">
            {selectedContainer.state.terminated.reason}
          </div>
          {!!selectedContainer.state.terminated.message && (
            <>
              <div className="text-muted-foreground">退出消息</div>
              <div className="col-span-2 font-mono break-all whitespace-normal">
                {selectedContainer.state.terminated.message}
              </div>
            </>
          )}
          <div className="text-muted-foreground">开始时间</div>
          <div className="col-span-2">
            <TimeDistance date={selectedContainer.state.terminated.startedAt} />
          </div>
          <div className="text-muted-foreground">结束时间</div>
          <div className="col-span-2">
            <TimeDistance
              date={selectedContainer.state.terminated.finishedAt}
            />
          </div>
        </>
      )}
      {selectedContainer.state.waiting && (
        <>
          {!!selectedContainer.state.waiting.reason && (
            <>
              <div className="text-muted-foreground">等待原因</div>
              <div className="col-span-2 font-mono break-all whitespace-normal">
                {selectedContainer.state.waiting.reason}
              </div>
            </>
          )}
          {!!selectedContainer.state.waiting.message && (
            <>
              <div className="text-muted-foreground">等待消息</div>
              <div className="col-span-2 font-mono break-all whitespace-normal">
                {selectedContainer.state.waiting.message}
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
};

function Content({
  namespacedName,
  ActionComponent,
}: {
  namespacedName?: PodNamespacedName;
  ActionComponent: React.ComponentType<{
    namespacedName: PodNamespacedName;
    selectedContainer: ContainerInfo;
  }>;
}) {
  const { namespace, name: podName } = namespacedName ?? {};
  const queryClient = useQueryClient();

  const [selectedContainer, setSelectedContainer] = useState<
    ContainerInfo | undefined
  >();

  const { data: containers } = useQuery({
    queryKey: ["log", "containers", namespace, podName],
    queryFn: () => apiGetPodContainers(namespace, podName),
    select: (res) => res.data.data.containers.filter((c) => c.name !== ""),
    enabled: !!namespace && !!podName,
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

  return (
    <>
      {!containers || !selectedContainer ? (
        <div className="flex h-[calc(100vh_-190px)] w-full items-center justify-center">
          <LoadingCircleIcon />
        </div>
      ) : (
        <div className="grid h-[calc(100vh_-190px)] w-[calc(100vw_-154px)] gap-6 md:grid-cols-3 xl:grid-cols-4">
          {namespacedName && selectedContainer && (
            <ActionComponent
              namespacedName={namespacedName}
              selectedContainer={selectedContainer}
            />
          )}
          <div className="space-y-4">
            <ContainerSelect
              currentContainer={selectedContainer}
              setCurrentContainer={setSelectedContainer}
              containers={containers}
            />
            <fieldset className="border-input hidden h-[calc(100vh_-264px)] max-h-full gap-6 overflow-y-auto rounded-lg border p-4 shadow-xs md:grid">
              <legend className="-ml-1 px-2 text-sm font-medium">
                {selectedContainer.isInitContainer ? "初始化容器" : "容器信息"}
              </legend>
              <TableCellForm selectedContainer={selectedContainer} />
            </fieldset>
          </div>
        </div>
      )}
    </>
  );
}

export function PodContainerDialog({
  // props
  namespacedName,
  setNamespacedName,
  ActionComponent,
  type,
}: PodContainerDialogProps & {
  ActionComponent: React.ComponentType<{
    namespacedName: PodNamespacedName;
    selectedContainer: ContainerInfo;
  }>;
  type: "shell" | "log";
}) {
  const [isOpen, setIsOpen] = useNamespacedState(
    namespacedName,
    setNamespacedName,
  );

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen} modal={true}>
      <DialogContent
        className="h-[calc(100vh_-104px)] w-[calc(100vw_-104px)] gap-5 sm:max-w-full"
        onOpenAutoFocus={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => {
          // 当类型为 shell 时，阻止 ESC 关闭对话框
          if (type === "shell") {
            e.preventDefault();
          }
        }}
        // onPointerDownOutside={(e) => {
        //   // 当类型为 shell 时，阻止点击外部关闭对话框
        //   if (type === "shell") {
        //     toast.warning("请使用右上角关闭按钮，手动关闭终端");
        //     e.preventDefault();
        //   }
        // }}
      >
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center gap-1.5 font-semibold">
            <span className="font-mono">{namespacedName?.name}</span>
            <TipBadge title={type} className="uppercase" />
          </DialogTitle>
        </DialogHeader>
        <Content
          namespacedName={namespacedName}
          ActionComponent={ActionComponent}
        />
      </DialogContent>
    </Dialog>
  );
}
