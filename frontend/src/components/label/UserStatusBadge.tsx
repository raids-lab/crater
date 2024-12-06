import { PhaseLabel } from "./PhaseLabel";
import { ProjectStatus } from "@/services/api/account";

export const userAccesses = [
  {
    value: ProjectStatus.Inactive.toString(),
    label: "已禁用",
    color: "text-orange-500 border-0 bg-orange-500/10",
    description: "用户已被禁用，无法使用资源",
  },
  {
    value: ProjectStatus.Active.toString(),
    label: "已激活",
    color: "text-emerald-500 border-0 bg-emerald-500/10",
    description: "用户已激活，可以使用资源",
  },
];

const getUserStatusLabel = (
  phase: string,
): {
  label: string;
  color: string;
  description: string;
} => {
  const foundPhase = userAccesses.find((p) => p.value === phase);
  if (foundPhase) {
    return foundPhase;
  } else {
    return {
      label: "未知",
      color: "text-slate-500 border-0 bg-slate-500/10",
      description: "由于某些原因无法获取用户读写权限信息",
    };
  }
};

const UserStatusBadge = ({ status }: { status: string }) => {
  return <PhaseLabel phase={status} getPhaseLabel={getUserStatusLabel} />;
};

export default UserStatusBadge;
