import { userInfoState } from "@/utils/store";
import { useMemo } from "react";
import { useRecoilValue } from "recoil";

export interface UserInfo {
  id: string;
  role: "admin" | "user" | "viewer"; // admin | power user | user | viewer
}

export function useAuth(role: UserInfo["role"]) {
  const userInfo = useRecoilValue(userInfoState);
  const isAuthenticated = useMemo(() => {
    return userInfo.role === role;
  }, [userInfo, role]);
  return isAuthenticated;
}
