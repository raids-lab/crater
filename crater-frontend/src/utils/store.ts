import { AtomEffect, atom } from "recoil";
import { UserInfo } from "@/hooks/useAuth";

/**
 * Global UI States
 */
export const sidebarIndexes = atom({
  key: "sidebarIndexes",
  default: ["item-2"],
});

export const sidebarHighlight = atom({
  key: "sidebarHighlight",
  default: 0,
});

/**
 * User States
 */
export const localStorageEffect =
  <T>(key: string): AtomEffect<T> =>
  ({ setSelf, onSet }) => {
    const savedValue = localStorage.getItem(key);
    if (savedValue != null) {
      setSelf(JSON.parse(savedValue) as T);
    }
    onSet((newValue, _, isReset) => {
      isReset
        ? localStorage.removeItem(key)
        : localStorage.setItem(key, JSON.stringify(newValue));
    });
  };

export const currentUserState = atom({
  key: "userInfo",
  default: {
    id: "",
    role: "viewer",
    token: "",
  } as UserInfo,
  effects: [localStorageEffect("current_user")],
});
