import { globalUserInfo } from "@/utils/store";
import { useMemo } from "react";
import { useRecoilValue } from "recoil";

type Role = "admin" | "user" | "viewer"; // admin | power user | user | viewer

export interface UserInfo {
  id: string;
  role: Role;
}

/**
 * A hook that checks if the current user is authenticated and has the required role.
 *
 * @param requireRole - The required role to access the resource.
 * @returns A boolean indicating whether the user is authenticated and has the required role.
 */
export function useAuth(requireRole: Role) {
  const { role } = useRecoilValue(globalUserInfo);
  const isAuthenticated = useMemo(() => {
    switch (requireRole) {
      case "admin":
        return role === "admin";
      case "user":
        return role === "admin" || role === "user";
      case "viewer":
        return true;
      default:
        return true;
    }
  }, [requireRole, role]);
  return isAuthenticated;
}
