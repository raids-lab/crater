import { PhaseLabel } from "./PhaseLabel";

export enum ContainerStatus {
  Waiting = "Waiting",
  Running = "Running",
  Terminated = "Terminated",
}

export const containerStatuses = [
  {
    value: "Waiting",
    label: "等待",
    color: "text-purple-500 border-0 bg-purple-500/10",
    description:
      "如果容器并不处在 Running 或 Terminated 状态之一，它就处在 Waiting 状态。 处于 Waiting 状态的容器仍在运行它完成启动所需要的操作：例如， 从某个容器镜像仓库拉取容器镜像，或者向容器应用 Secret 数据等等。",
  },
  {
    value: "Running",
    label: "运行中",
    color: "text-blue-500 border-0 bg-blue-500/10",
    description: "Running 状态表明容器正在执行状态并且没有问题发生。",
  },
  {
    value: "Terminated",
    label: "已终止",
    color: "text-slate-500 border-0 bg-slate-500/10",
    description:
      "处于 Terminated 状态的容器开始执行后，或者运行至正常结束或者因为某些原因失败。",
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
      color: "text-slate-500 border-slate-500 bg-slate-500/10",
      description: "由于某些原因无法获取 Container 的状态信息",
    };
  }
};

const ContainerStatusLabel = ({
  containerStatus,
}: {
  containerStatus: string;
}) => {
  return (
    <PhaseLabel
      phase={containerStatus}
      getPhaseLabel={getContainerStatusLabel}
    />
  );
};

export default ContainerStatusLabel;
