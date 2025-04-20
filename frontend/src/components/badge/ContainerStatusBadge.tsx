import { PhaseBadge } from "./PhaseBadge";

export enum ContainerStatus {
  Waiting = "Waiting",
  Running = "Running",
  Terminated = "Terminated",
}

export const containerStatuses = [
  {
    value: "Waiting",
    label: "等待",
    color: "text-highlight-purple bg-highlight-purple/20",
    description:
      "容器仍在运行它完成启动所需要的操作：如从某个容器镜像仓库拉取容器镜像，或者向容器应用 Secret 数据等",
  },
  {
    value: "Running",
    label: "运行中",
    color: "text-highlight-blue bg-highlight-blue/20",
    description: "容器正在执行，并且没有问题发生",
  },
  {
    value: "Terminated",
    label: "已终止",
    color: "text-highlight-slate bg-highlight-slate/20",
    description: "容器运行至正常结束，或者因为某些原因失败。",
  },
];

const getContainerStatusLabel = (
  phase: string,
): {
  label: string;
  color: string;
  description: string;
} => {
  const foundPhase = containerStatuses.find((p) => p.value === phase);
  if (foundPhase) {
    return foundPhase;
  } else {
    return {
      label: "未知",
      color: "text-highlight-slate bg-highlight-slate/20",
      description: "由于某些原因无法获取 Container 的状态信息",
    };
  }
};

const ContainerStatusBadge = ({
  containerStatus,
}: {
  containerStatus: string;
}) => {
  return (
    <PhaseBadge
      phase={containerStatus}
      getPhaseLabel={getContainerStatusLabel}
    />
  );
};

export default ContainerStatusBadge;
