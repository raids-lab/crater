import { atom } from "recoil";
import { UserInfo } from "@/hooks/useAuth";
import { localStorageEffect } from "./utils";

/**
 * LocalStorage and Recoil Keys
 */
const UI_ACCORDION_KEY = "ui_accordion";
const UI_INDEX_KEY = "ui_index";

const USER_INFO_KEY = "user_info";

/**
 * Global UI States
 */
export const uiAccordionState = atom({
  key: UI_ACCORDION_KEY,
  default: ["item-2"],
});

export const uiIndexState = atom({
  key: UI_INDEX_KEY,
  default: 0,
  effects: [localStorageEffect(UI_INDEX_KEY)],
});

/**
 * User States
 */
export const userInfoState = atom({
  key: "userInfo",
  default: {
    id: "",
    role: "viewer",
    token: "",
  } as UserInfo,
  effects: [localStorageEffect(USER_INFO_KEY)],
});
