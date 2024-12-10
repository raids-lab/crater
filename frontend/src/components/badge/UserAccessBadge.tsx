import { PhaseBadge } from "./PhaseBadge";
import { Access } from "@/services/api/account";

export const userAccesses = [
  {
    value: Access.RW.toString(),
    label: "读写",
    color: "text-orange-500 border-0 bg-orange-500/10",
    description: "账户内的用户管理、资源分配等操作，将由账户管理员负责",
  },
  {
    value: Access.RO.toString(),
    label: "只读",
    color: "text-emerald-500 border-0 bg-emerald-500/10",
    description: "账户内的普通用户，只能使用资源，无法进行管理操作",
  },
];

const getUserAccessLabel = (
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

const UserAccessBadge = ({ access }: { access: string }) => {
  return <PhaseBadge phase={access} getPhaseLabel={getUserAccessLabel} />;
};

export default UserAccessBadge;
