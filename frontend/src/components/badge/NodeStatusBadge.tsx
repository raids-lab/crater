import { PhaseBadge } from "./PhaseBadge";

export const nodeStatuses = [
  {
    value: "true",
    label: "运行中",
    color: "text-highlight-blue bg-highlight-blue/20",
    description: "节点正常运行",
  },
  {
    value: "false",
    label: "异常",
    color: "text-highlight-red bg-highlight-red/20",
    description: "节点出现异常",
  },
  {
    value: "Unschedulable",
    label: "不可调度",
    color: "text-highlight-orange bg-highlight-orange/20",
    description: "节点不可调度",
  },
];

const getNodeStatusLabel = (
  status: string,
): {
  label: string;
  color: string;
  description: string;
} => {
  const foundStatus = nodeStatuses.find((s) => s.value === status);
  if (foundStatus) {
    return foundStatus;
  } else {
    return {
      label: "未知",
      color: "text-highlight-slate bg-highlight-slate/20",
      description: "无法获取节点状态信息",
    };
  }
};

const NodeStatusBadge = ({ status }: { status: string }) => {
  return <PhaseBadge phase={status} getPhaseLabel={getNodeStatusLabel} />;
};

export default NodeStatusBadge;
