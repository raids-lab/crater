import { atom, useResetRecoilState } from "recoil";
import { UserInfo } from "@/hooks/useAuth";
import { localStorageEffect } from "./utils";

/**
 * LocalStorage and Recoil Keys
 */
const USER_INFO_KEY = "user_info";
const LAST_VIEW_KEY = "last_view";
const BREAD_CRUMB_KEY = "bread_crumb";

export const ACCESS_TOKEN_KEY = "access_token";
export const REFRESH_TOKEN_KEY = "refresh_token";
export const VITE_UI_THEME_KEY = "vite_ui_theme";

/**
 * User States
 */
export const globalUserInfo = atom({
  key: USER_INFO_KEY,
  default: {
    id: "",
    role: "viewer",
  } as UserInfo,
  effects: [localStorageEffect(USER_INFO_KEY)],
});

/**
 * Remember the last view.
 * Will not be cleared when logout.
 */
export const globalLastView = atom({
  key: LAST_VIEW_KEY,
  default: "",
  effects: [localStorageEffect(LAST_VIEW_KEY)],
});

/**
 * Navigation BreadCrumb
 */
type BreadCrumbItem = {
  title: string;
  path?: string;
};

export const globalBreadCrumb = atom({
  key: BREAD_CRUMB_KEY,
  default: [] as BreadCrumbItem[],
});

/**
 * Reset all states
 */
export const useResetStore = () => {
  const resetUserInfo = useResetRecoilState(globalUserInfo);
  const resetBreadCrumb = useResetRecoilState(globalBreadCrumb);

  const resetAll = () => {
    // Recoil
    resetUserInfo();
    resetBreadCrumb();
    // LocalStorage
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  };

  return { resetAll };
};
