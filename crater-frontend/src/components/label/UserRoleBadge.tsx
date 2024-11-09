import { Role } from "@/services/api/auth";
import { PhaseLabel } from "./PhaseLabel";

export const userRoles = [
  {
    value: Role.Admin.toString(),
    label: "管理员",
    color: "text-purple-500 border-0 bg-purple-500/10",
    description: "账户内的用户管理、资源分配等操作，将由账户管理员负责",
  },
  {
    value: Role.User.toString(),
    label: "用户",
    color: "text-blue-500 border-0 bg-blue-500/10",
    description: "账户内的普通用户，只能使用资源，无法进行管理操作",
  },
];

const getUserRoleLabel = (
  phase: string,
): {
  label: string;
  color: string;
  description: string;
} => {
  const foundPhase = userRoles.find((p) => p.value === phase);
  if (foundPhase) {
    return foundPhase;
  } else {
    return {
      label: "未知",
      color: "text-slate-500 border-0 bg-slate-500/10",
      description: "由于某些原因无法获取用户角色信息",
    };
  }
};

const UserRoleBadge = ({ role }: { role: string }) => {
  return <PhaseLabel phase={role} getPhaseLabel={getUserRoleLabel} />;
};

export default UserRoleBadge;
