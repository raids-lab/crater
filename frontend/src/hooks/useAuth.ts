import { Role } from "@/services/api/auth";
import { globalAccount } from "@/utils/store";
import { useMemo } from "react";
import { useAtomValue } from "jotai";

export interface UserInfo {
  name: string; // unique username
}

/**
 * A hook that checks if the current user is authenticated and has the required role.
 *
 * @param requireRole - The required role to access the resource.
 * @returns A boolean indicating whether the user is authenticated and has the required role.
 */
export function useAuth(requireRole: Role) {
  const { rolePlatform: role } = useAtomValue(globalAccount);
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
