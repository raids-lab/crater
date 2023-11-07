import { atom, useResetRecoilState } from "recoil";
import { UserInfo } from "@/hooks/useAuth";
import { localStorageEffect } from "./utils";

/**
 * LocalStorage and Recoil Keys
 */
const UI_INDEX_KEY = "ui_index";
const USER_INFO_KEY = "user_info";

export const ACCESS_TOKEN_KEY = "access_token";
export const REFRESH_TOKEN_KEY = "refresh_token";
export const VITE_UI_THEME_KEY = "vite_ui_theme";

/**
 * Global UI States
 */

export const uiActivedState = atom({
  key: UI_INDEX_KEY,
  default: {
    item: "",
    subItem: "",
  },
  // effects: [localStorageEffect(UI_INDEX_KEY)],
});

/**
 * User States
 */
export const userInfoState = atom({
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
  const resetUIActived = useResetRecoilState(uiActivedState);
  const resetUserInfo = useResetRecoilState(userInfoState);

  const resetAll = () => {
    // Recoil
    resetUIActived();
    resetUserInfo();
    // LocalStorage
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  };

  return { resetAll };
};
