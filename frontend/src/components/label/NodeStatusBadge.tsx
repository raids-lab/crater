import { PhaseLabel } from "./PhaseLabel";

export const nodeStatuses = [
  {
    value: "true",
    label: "运行中",
    color: "text-blue-500 border-0 bg-blue-500/10",
    description: "节点正常运行",
  },
  {
    value: "false",
    label: "异常",
    color: "text-red-500 border-0 bg-red-500/10",
    description: "节点出现异常",
  },
  {
    value: "Unschedulable",
    label: "不可调度",
    color: "text-orange-500 border-0 bg-orange-500/10",
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
      color: "text-slate-500 border-0 bg-slate-500/10",
      description: "无法获取节点状态信息",
    };
  }
};

const NodeStatusBadge = ({ status }: { status: string }) => {
  return <PhaseLabel phase={status} getPhaseLabel={getNodeStatusLabel} />;
};

export default NodeStatusBadge;
