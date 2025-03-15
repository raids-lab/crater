import { PhaseBadge } from "./PhaseBadge";
import { Access } from "@/services/api/account";

export const userAccesses = [
  {
    value: Access.RW.toString(),
    label: "读写",
    color: "text-highlight-orange bg-highlight-orange/20",
    description: "拥有对数据的读写权限",
  },
  {
    value: Access.RO.toString(),
    label: "只读",
    color: "text-highlight-emerald bg-highlight-emerald/20",
    description: "只有对数据的读取权限",
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
      color: "text-highlight-slate bg-highlight-slate/20",
      description: "由于某些原因无法获取用户读写权限信息",
    };
  }
};

const UserAccessBadge = ({ access }: { access: string }) => {
  return <PhaseBadge phase={access} getPhaseLabel={getUserAccessLabel} />;
};

export default UserAccessBadge;
