import { PhaseBadge } from "./PhaseBadge";
import { ProjectStatus } from "@/services/api/account";

export const userAccesses = [
  {
    value: ProjectStatus.Inactive.toString(),
    label: "已禁用",
    color: "text-highlight-orange bg-highlight-orange/20",
    description: "用户已被禁用，无法使用资源",
  },
  {
    value: ProjectStatus.Active.toString(),
    label: "已激活",
    color: "text-highlight-emerald bg-highlight-emerald/20",
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
      color: "text-highlight-slate bg-highlight-slate/20",
      description: "由于某些原因无法获取用户读写权限信息",
    };
  }
};

const UserStatusBadge = ({ status }: { status: string }) => {
  return <PhaseBadge phase={status} getPhaseLabel={getUserStatusLabel} />;
};

export default UserStatusBadge;
