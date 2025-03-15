import { PhaseBadge } from "./PhaseBadge";

export const podPhases = [
  {
    value: "Pending",
    label: "等待中",
    color: "text-highlight-purple bg-highlight-purple/20",
    description: "Pod 正在等待被调度或容器镜像正在下载",
  },
  {
    value: "Running",
    label: "运行中",
    color: "text-highlight-blue bg-highlight-blue/20",
    description: "Pod 已被调度到节点上且至少一个容器正在运行",
  },
  {
    value: "Succeeded",
    label: "已完成",
    color: "text-highlight-emerald bg-highlight-emerald/20",
    description: "Pod 中的所有容器已成功终止且不会再启动",
  },
  {
    value: "Failed",
    label: "失败",
    color: "text-highlight-red bg-highlight-red/20",
    description: "Pod 中的所有容器已终止且至少一个容器因错误终止",
  },
  {
    value: "Unknown",
    label: "未知",
    color: "text-highlight-slate bg-highlight-slate/20",
    description: "由于某些原因无法获取 Pod 的状态信息",
  },
];

const getPodPhaseLabel = (
  phase: string,
): {
  label: string;
  color: string;
  description: string;
} => {
  const foundPhase = podPhases.find((p) => p.value === phase);
  if (foundPhase) {
    return foundPhase;
  } else {
    return {
      label: "未知",
      color: "text-highlight-slate bg-highlight-slate/20",
      description: "由于某些原因无法获取 Pod 的状态信息",
    };
  }
};

const PodPhaseLabel = ({ podPhase }: { podPhase: string }) => {
  return <PhaseBadge phase={podPhase} getPhaseLabel={getPodPhaseLabel} />;
};

export default PodPhaseLabel;
