import { ProjectBasic, Role, UserContext } from "@/services/api/auth";
import { globalUserInfo } from "@/utils/store";
import { useMemo } from "react";
import { useRecoilValue } from "recoil";

export interface UserInfo {
  name: string; // unique username
  context: UserContext; // user context
  projects: ProjectBasic[]; // user projects
}

/**
 * A hook that checks if the current user is authenticated and has the required role.
 *
 * @param requireRole - The required role to access the resource.
 * @returns A boolean indicating whether the user is authenticated and has the required role.
 */
export function useAuth(requireRole: Role) {
  const { context } = useRecoilValue(globalUserInfo);
  const role = context.platformRole;
  const isAuthenticated = useMemo(() => {
    switch (requireRole) {
      case Role.Admin:
        return role === Role.Admin;
      case Role.User:
        return role === Role.Admin || role === Role.User;
      case Role.Guest:
        return true;
      default:
        return true;
    }
  }, [requireRole, role]);
  return isAuthenticated;
}
