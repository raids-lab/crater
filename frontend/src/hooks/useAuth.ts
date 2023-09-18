import { currentUserState } from "@/utils/store";
import { useMemo } from "react";
import { useRecoilValue } from "recoil";

export interface UserInfo {
  id: string;
  role: "admin" | "user" | "viewer"; // admin | power user | user | viewer
  token: string;
}

export function useAuth(role: UserInfo["role"]) {
  const userInfo = useRecoilValue(currentUserState);
  const isAuthenticated = useMemo(() => {
    return userInfo.role === role;
  }, [userInfo, role]);
  return isAuthenticated;
}
