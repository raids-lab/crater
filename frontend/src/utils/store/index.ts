import { atom, useResetRecoilState } from "recoil";
import { UserInfo } from "@/hooks/useAuth";
import { localStorageEffect } from "./utils";

/**
 * LocalStorage and Recoil Keys
 */
const USER_INFO_KEY = "user_info";

export const ACCESS_TOKEN_KEY = "access_token";
export const REFRESH_TOKEN_KEY = "refresh_token";
export const VITE_UI_THEME_KEY = "vite_ui_theme";

/**
 * User States
 */
export const globalUserInfo = atom({
  key: "userInfo",
  default: {
    id: "",
    role: "viewer",
  } as UserInfo,
  effects: [localStorageEffect(USER_INFO_KEY)],
});

/**
 * Reset all states
 */
export const useResetStore = () => {
  const resetUserInfo = useResetRecoilState(globalUserInfo);

  const resetAll = () => {
    // Recoil
    resetUserInfo();
    // LocalStorage
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  };

  return { resetAll };
};
