import { atom, useResetRecoilState } from "recoil";
import { UserInfo } from "@/hooks/useAuth";
import { localStorageEffect } from "./utils";
import { Role } from "@/services/api/auth";

/**
 * LocalStorage and Recoil Keys
 */
const USER_INFO_KEY = "user_info";
const LAST_VIEW_KEY = "last_view";
const BREAD_CRUMB_KEY = "bread_crumb";
const SIDEBAR_MINISIZE_KEY = "sidebar_mini";
const CURRENT_ACCOUNT_KEY = "current_account";

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
    role: Role.Guest,
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

export const globalSidebarMini = atom({
  key: SIDEBAR_MINISIZE_KEY,
  default: false,
  effects_UNSTABLE: [localStorageEffect(SIDEBAR_MINISIZE_KEY)],
});

export const globalCurrentAccount = atom({
  key: CURRENT_ACCOUNT_KEY,
  default: {
    label: "",
    value: "",
  },
  effects_UNSTABLE: [localStorageEffect(CURRENT_ACCOUNT_KEY)],
});

/**
 * Reset all states
 */
export const useResetStore = () => {
  const resetUserInfo = useResetRecoilState(globalUserInfo);
  const resetBreadCrumb = useResetRecoilState(globalBreadCrumb);
  const resetSidebarMini = useResetRecoilState(globalSidebarMini);
  const resetCurrentAccount = useResetRecoilState(globalCurrentAccount);

  const resetAll = () => {
    // Recoil
    resetUserInfo();
    resetBreadCrumb();
    resetSidebarMini();
    resetCurrentAccount();
    // LocalStorage
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  };

  return { resetAll };
};
